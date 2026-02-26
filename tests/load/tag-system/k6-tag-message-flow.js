import http from 'k6/http';
import exec from 'k6/execution';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';

const sendLatency = new Trend('tag_send_latency_ms', true);
const botLag = new Trend('tag_bot_lag_ms', true);
const botObservedRate = new Rate('tag_bot_observed_rate');
const sendSuccessRate = new Rate('tag_send_success_rate');

const status2xx = new Counter('tag_http_2xx');
const status4xx = new Counter('tag_http_4xx');
const status5xx = new Counter('tag_http_5xx');
const authErrors = new Counter('tag_auth_errors');
const validationErrors = new Counter('tag_validation_errors');
const timeoutErrors = new Counter('tag_timeout_errors');
const networkErrors = new Counter('tag_network_errors');
const parseErrors = new Counter('tag_parse_errors');

const DEFAULT_HEADERS = { 'Content-Type': 'application/json' };

function toBool(value, defaultValue) {
  if (value === undefined || value === null || value === '') {
    return defaultValue;
  }
  return String(value).toLowerCase() === 'true';
}

function toInt(value, defaultValue) {
  if (value === undefined || value === null || value === '') {
    return defaultValue;
  }
  const parsed = parseInt(value, 10);
  return Number.isFinite(parsed) ? parsed : defaultValue;
}

function toFloat(value, defaultValue) {
  if (value === undefined || value === null || value === '') {
    return defaultValue;
  }
  const parsed = parseFloat(value);
  return Number.isFinite(parsed) ? parsed : defaultValue;
}

function splitCSV(value) {
  if (!value) {
    return [];
  }
  return String(value)
    .split(',')
    .map((v) => v.trim())
    .filter((v) => v.length > 0);
}

function nowISO() {
  return new Date().toISOString();
}

function randomSuffix() {
  return Math.random().toString(36).slice(2, 10);
}

function profileDefaults(profile) {
  switch ((profile || 'smoke').toLowerCase()) {
    case 'moderate':
      return { executor: 'constant-vus', vus: 10, duration: '2m' };
    case 'high':
      return { executor: 'constant-vus', vus: 50, duration: '3m' };
    case 'burst':
      return {
        executor: 'ramping-vus',
        startVUs: 0,
        stages: [
          { duration: '30s', target: 10 },
          { duration: '45s', target: 75 },
          { duration: '45s', target: 75 },
          { duration: '30s', target: 10 },
          { duration: '15s', target: 0 },
        ],
      };
    case 'smoke':
    default:
      return { executor: 'constant-vus', vus: 2, duration: '30s' };
  }
}

function buildScenario(profile) {
  const defaults = profileDefaults(profile);
  const scenario = {
    executor: defaults.executor,
    exec: 'runTagFlow',
    gracefulStop: __ENV.GRACEFUL_STOP || '5s',
    tags: {
      profile: (profile || 'smoke').toLowerCase(),
      mode: (__ENV.MODE || 'shared').toLowerCase(),
    },
  };

  if (defaults.executor === 'constant-vus') {
    scenario.vus = toInt(__ENV.VUS, defaults.vus);
    scenario.duration = __ENV.DURATION || defaults.duration;
  } else if (defaults.executor === 'ramping-vus') {
    scenario.startVUs = toInt(__ENV.START_VUS, defaults.startVUs);
    if (__ENV.STAGES) {
      try {
        scenario.stages = JSON.parse(__ENV.STAGES);
      } catch (_) {
        scenario.stages = defaults.stages;
      }
    } else {
      scenario.stages = defaults.stages;
    }
  }

  return scenario;
}

function buildThresholds() {
  const thresholds = {
    http_req_failed: ['rate<0.10'],
    http_req_duration: ['p(95)<2000', 'p(99)<5000'],
    tag_send_success_rate: ['rate>0.90'],
    tag_send_latency_ms: ['p(95)<1500', 'p(99)<4000'],
  };

  if (__ENV.POLL_BOT === 'true') {
    thresholds.tag_bot_observed_rate = ['rate>0.60'];
    thresholds.tag_bot_lag_ms = ['p(95)<5000'];
  }

  return thresholds;
}

export const options = {
  scenarios: {
    tag_flow: buildScenario(__ENV.PROFILE || 'smoke'),
  },
  thresholds: buildThresholds(),
  summaryTrendStats: ['avg', 'min', 'med', 'p(90)', 'p(95)', 'p(99)', 'max'],
};

function cfg() {
  const baseUrl = (__ENV.BASE_URL || 'http://127.0.0.1:8080').replace(/\/+$/, '');
  const token = __ENV.AUTH_TOKEN || '';
  const mode = (__ENV.MODE || 'shared').toLowerCase();
  const pollBot = toBool(__ENV.POLL_BOT, false);
  const thinkMs = toInt(__ENV.THINK_MS, 0);
  const invalidRatio = Math.max(0, Math.min(1, toFloat(__ENV.INVALID_RATIO, 0.05)));
  const multiRatio = Math.max(0, Math.min(1, toFloat(__ENV.MULTI_TAG_RATIO, 0.20)));
  const botPollIntervalMs = toInt(__ENV.BOT_POLL_INTERVAL_MS, 100);
  const botPollTimeoutMs = toInt(__ENV.BOT_POLL_TIMEOUT_MS, 5000);
  const botSearchLimit = toInt(__ENV.BOT_SEARCH_LIMIT, 30);
  const autoSetup = toBool(__ENV.AUTO_SETUP, false);
  const autoSetupChatCount = toInt(__ENV.AUTO_SETUP_CHAT_COUNT, 20);

  return {
    baseUrl,
    token,
    mode,
    pollBot,
    thinkMs,
    invalidRatio,
    multiRatio,
    botPollIntervalMs,
    botPollTimeoutMs,
    botSearchLimit,
    autoSetup,
    autoSetupChatCount,
    sharedChatId: __ENV.SHARED_CHAT_ID || '',
    workspaceId: __ENV.WORKSPACE_ID || '',
    chatIds: splitCSV(__ENV.CHAT_IDS),
  };
}

function authHeaders(token) {
  const headers = Object.assign({}, DEFAULT_HEADERS);
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  return headers;
}

function parseJSONOrNull(response) {
  try {
    return response.json();
  } catch (_) {
    parseErrors.add(1);
    return null;
  }
}

function classifyHTTPFailure(response) {
  if (!response) {
    networkErrors.add(1);
    return;
  }
  if (response.status >= 200 && response.status < 300) {
    status2xx.add(1);
    return;
  }
  if (response.status >= 400 && response.status < 500) {
    status4xx.add(1);
  } else if (response.status >= 500) {
    status5xx.add(1);
  }

  const body = parseJSONOrNull(response);
  const code = body && body.error && body.error.code ? body.error.code : '';
  if (response.status === 401 || response.status === 403) {
    authErrors.add(1);
  }
  if (code === 'VALIDATION_ERROR') {
    validationErrors.add(1);
  }
  if (response.status === 408 || response.status === 429) {
    timeoutErrors.add(1);
  }
}

function mustHaveAuth(config) {
  if (!config.token) {
    throw new Error('AUTH_TOKEN is required');
  }
}

function apiPost(config, path, payload, tags) {
  let response;
  try {
    response = http.post(`${config.baseUrl}${path}`, JSON.stringify(payload), {
      headers: authHeaders(config.token),
      tags: tags || {},
      timeout: __ENV.REQUEST_TIMEOUT || '10s',
    });
  } catch (err) {
    networkErrors.add(1);
    throw err;
  }
  return response;
}

function apiGet(config, path, tags) {
  let response;
  try {
    response = http.get(`${config.baseUrl}${path}`, {
      headers: authHeaders(config.token),
      tags: tags || {},
      timeout: __ENV.REQUEST_TIMEOUT || '10s',
    });
  } catch (err) {
    networkErrors.add(1);
    throw err;
  }
  return response;
}

function createWorkspace(config) {
  const resp = apiPost(
    config,
    '/api/v1/workspaces',
    {
      name: `Tag Load ${randomSuffix()}`,
      description: `k6 tag-system load test ${nowISO()}`,
    },
    { endpoint: 'workspace_create' }
  );
  const body = parseJSONOrNull(resp);
  if (resp.status !== 201 || !body || !body.data || !body.data.id) {
    classifyHTTPFailure(resp);
    throw new Error(`failed to create workspace: status=${resp.status} body=${resp.body}`);
  }
  status2xx.add(1);
  return body.data.id;
}

function createTaskChat(config, workspaceId, idx) {
  const resp = apiPost(
    config,
    `/api/v1/workspaces/${workspaceId}/chats`,
    {
      name: `Tag Load Task ${idx + 1}`,
      type: 'task',
      is_public: true,
    },
    { endpoint: 'chat_create' }
  );
  const body = parseJSONOrNull(resp);
  if (resp.status !== 201 || !body || !body.data || !body.data.id) {
    classifyHTTPFailure(resp);
    throw new Error(`failed to create task chat: status=${resp.status} body=${resp.body}`);
  }
  status2xx.add(1);
  return body.data.id;
}

function resolveTargetData(config) {
  if (config.autoSetup) {
    const workspaceId = createWorkspace(config);
    const chatCount = Math.max(
      config.mode === 'distributed' ? 2 : 1,
      config.autoSetupChatCount
    );
    const chatIds = [];
    for (let i = 0; i < chatCount; i++) {
      chatIds.push(createTaskChat(config, workspaceId, i));
    }
    return {
      workspaceId,
      chatIds,
      sharedChatId: chatIds[0],
      setupMode: 'auto',
    };
  }

  const chatIds = config.chatIds.slice();
  if (config.sharedChatId && chatIds.indexOf(config.sharedChatId) === -1) {
    chatIds.unshift(config.sharedChatId);
  }

  if (!config.workspaceId) {
    throw new Error('WORKSPACE_ID is required unless AUTO_SETUP=true');
  }
  if (chatIds.length === 0) {
    throw new Error('Provide SHARED_CHAT_ID or CHAT_IDS unless AUTO_SETUP=true');
  }

  return {
    workspaceId: config.workspaceId,
    chatIds,
    sharedChatId: config.sharedChatId || chatIds[0],
    setupMode: 'manual',
  };
}

function pickChatId(runData) {
  if (runData.mode === 'shared') {
    return runData.sharedChatId;
  }
  const idx = (exec.vu.idInTest + exec.scenario.iterationInTest) % runData.chatIds.length;
  return runData.chatIds[idx];
}

function choosePayload(config) {
  const r = Math.random();
  if (r < config.invalidRatio) {
    return {
      content: '#status Completed',
      kind: 'invalid',
      expectedBotFragments: ['invalid status'],
    };
  }
  if (r < config.invalidRatio + config.multiRatio) {
    const status = Math.random() < 0.5 ? 'Done' : 'In Progress';
    const priority = Math.random() < 0.5 ? 'High' : 'Low';
    return {
      content: `#status ${status} #priority ${priority}`,
      kind: 'multi',
      expectedBotFragments: ['status changed to', 'priority changed to'],
    };
  }
  const statuses = ['To Do', 'In Progress', 'Done'];
  const status = statuses[(exec.scenario.iterationInTest + exec.vu.idInTest) % statuses.length];
  return {
    content: `#status ${status}`,
    kind: 'status',
    expectedBotFragments: ['status changed to'],
  };
}

function observeBotMessage(runData, chatId, expectedFragments, startedAtMs) {
  if (!runData.pollBot) {
    return false;
  }

  const deadline = Date.now() + runData.botPollTimeoutMs;
  const expected = expectedFragments.map((s) => String(s).toLowerCase());

  while (Date.now() < deadline) {
    const resp = apiGet(
      runData,
      `/api/v1/workspaces/${runData.workspaceId}/chats/${chatId}/messages?limit=${runData.botSearchLimit}`,
      { endpoint: 'message_list_poll' }
    );

    if (resp.status !== 200) {
      classifyHTTPFailure(resp);
      sleep(runData.botPollIntervalMs / 1000);
      continue;
    }

    status2xx.add(1);
    const body = parseJSONOrNull(resp);
    const messages =
      body && body.data && Array.isArray(body.data.messages) ? body.data.messages : [];

    for (let i = messages.length - 1; i >= 0; i--) {
      const msg = messages[i];
      const msgType = msg && msg.type ? String(msg.type).toLowerCase() : '';
      if (msgType !== 'bot') {
        continue;
      }
      const content = msg && msg.content ? String(msg.content).toLowerCase() : '';
      let matches = true;
      for (let j = 0; j < expected.length; j++) {
        if (content.indexOf(expected[j]) === -1) {
          matches = false;
          break;
        }
      }
      if (!matches) {
        continue;
      }

      botLag.add(Date.now() - startedAtMs);
      botObservedRate.add(true);
      return true;
    }

    sleep(runData.botPollIntervalMs / 1000);
  }

  botObservedRate.add(false);
  return false;
}

export function setup() {
  const config = cfg();
  mustHaveAuth(config);

  const target = resolveTargetData(config);
  return {
    baseUrl: config.baseUrl,
    token: config.token,
    mode: config.mode,
    pollBot: config.pollBot,
    thinkMs: config.thinkMs,
    invalidRatio: config.invalidRatio,
    multiRatio: config.multiRatio,
    botPollIntervalMs: config.botPollIntervalMs,
    botPollTimeoutMs: config.botPollTimeoutMs,
    botSearchLimit: config.botSearchLimit,
    workspaceId: target.workspaceId,
    chatIds: target.chatIds,
    sharedChatId: target.sharedChatId,
    setupMode: target.setupMode,
  };
}

export function runTagFlow(runData) {
  const chatId = pickChatId(runData);
  const payload = choosePayload(runData);
  const startedAt = Date.now();

  let resp;
  try {
    resp = apiPost(
      runData,
      `/api/v1/workspaces/${runData.workspaceId}/chats/${chatId}/messages`,
      { content: payload.content },
      { endpoint: 'message_send', tag_kind: payload.kind, mode: runData.mode }
    );
  } catch (_) {
    sendSuccessRate.add(false);
    if (runData.thinkMs > 0) {
      sleep(runData.thinkMs / 1000);
    }
    return;
  }

  sendLatency.add(resp.timings.duration);
  const ok = check(resp, {
    'message send status is 201': (r) => r.status === 201,
  });
  sendSuccessRate.add(ok);

  if (resp.status === 201) {
    status2xx.add(1);
    observeBotMessage(runData, chatId, payload.expectedBotFragments, startedAt);
  } else {
    classifyHTTPFailure(resp);
  }

  if (runData.thinkMs > 0) {
    sleep(runData.thinkMs / 1000);
  }
}

export default runTagFlow;

function metricValue(data, metricName, statName) {
  if (!data || !data.metrics || !data.metrics[metricName]) {
    return 'n/a';
  }
  const metric = data.metrics[metricName];
  if (!metric.values || metric.values[statName] === undefined) {
    return 'n/a';
  }
  const value = metric.values[statName];
  if (typeof value === 'number') {
    return Number.isInteger(value) ? String(value) : value.toFixed(2);
  }
  return String(value);
}

export function handleSummary(data) {
  const lines = [];
  lines.push('Tag system load test summary');
  lines.push(`profile=${(__ENV.PROFILE || 'smoke').toLowerCase()} mode=${(__ENV.MODE || 'shared').toLowerCase()}`);
  lines.push(`base_url=${__ENV.BASE_URL || 'http://127.0.0.1:8080'} auto_setup=${toBool(__ENV.AUTO_SETUP, false)}`);
  lines.push(`poll_bot=${toBool(__ENV.POLL_BOT, false)} invalid_ratio=${toFloat(__ENV.INVALID_RATIO, 0.05)} multi_ratio=${toFloat(__ENV.MULTI_TAG_RATIO, 0.20)}`);
  lines.push('');
  lines.push(`http_req_failed(rate): ${metricValue(data, 'http_req_failed', 'rate')}`);
  lines.push(`http_req_duration p95/p99 (ms): ${metricValue(data, 'http_req_duration', 'p(95)')} / ${metricValue(data, 'http_req_duration', 'p(99)')}`);
  lines.push(`tag_send_success_rate: ${metricValue(data, 'tag_send_success_rate', 'rate')}`);
  lines.push(`tag_send_latency_ms p95/p99: ${metricValue(data, 'tag_send_latency_ms', 'p(95)')} / ${metricValue(data, 'tag_send_latency_ms', 'p(99)')}`);
  lines.push(`tag_http_2xx count: ${metricValue(data, 'tag_http_2xx', 'count')}`);
  lines.push(`tag_http_4xx count: ${metricValue(data, 'tag_http_4xx', 'count')}`);
  lines.push(`tag_http_5xx count: ${metricValue(data, 'tag_http_5xx', 'count')}`);
  lines.push(`tag_auth_errors count: ${metricValue(data, 'tag_auth_errors', 'count')}`);
  lines.push(`tag_validation_errors count: ${metricValue(data, 'tag_validation_errors', 'count')}`);
  lines.push(`tag_timeout_errors count: ${metricValue(data, 'tag_timeout_errors', 'count')}`);
  lines.push(`tag_network_errors count: ${metricValue(data, 'tag_network_errors', 'count')}`);

  if (toBool(__ENV.POLL_BOT, false)) {
    lines.push(`tag_bot_observed_rate: ${metricValue(data, 'tag_bot_observed_rate', 'rate')}`);
    lines.push(`tag_bot_lag_ms p95/p99: ${metricValue(data, 'tag_bot_lag_ms', 'p(95)')} / ${metricValue(data, 'tag_bot_lag_ms', 'p(99)')}`);
  }

  const summary = {
    stdout: `${lines.join('\n')}\n\n`,
  };

  if (__ENV.SUMMARY_JSON) {
    summary[__ENV.SUMMARY_JSON] = JSON.stringify(data, null, 2);
  }

  return summary;
}
