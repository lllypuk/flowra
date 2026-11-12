// Command agentbench measures Flowra under the "few people, many writing agents" profile over HTTP.
// It is a measurement tool, not part of the product: it creates its own workspace and task chats.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

type config struct {
	base      string
	token     string
	scenario  string
	agents    int
	duration  time.Duration
	rate      float64
	bodyBytes int
	thread    int
	statusPct int
}

type client struct {
	base  string
	token string
	http  *http.Client
}

type envelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (c *client) do(ctx context.Context, method, path string, body any) (int, json.RawMessage, error) {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+"/api/v1"+path, reader)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}
	var env envelope
	if json.Unmarshal(raw, &env) == nil && env.Success {
		return resp.StatusCode, env.Data, nil
	}
	return resp.StatusCode, raw, nil
}

// must runs a setup call that has to succeed.
func (c *client) must(ctx context.Context, method, path string, body any, into any) {
	status, data, err := c.do(ctx, method, path, body)
	if err != nil || status >= http.StatusBadRequest {
		fmt.Fprintf(os.Stderr, "setup %s %s failed: status %d err %v body %.200s\n", method, path, status, err, data)
		os.Exit(1)
	}
	if into != nil {
		if err = json.Unmarshal(data, into); err != nil {
			fmt.Fprintf(os.Stderr, "setup %s %s: decode: %v\n", method, path, err)
			os.Exit(1)
		}
	}
}

type sample struct {
	op      string
	status  int
	latency time.Duration
	failed  bool
}

type recorder struct {
	mu      sync.Mutex
	samples []sample
}

func (r *recorder) add(s sample) {
	r.mu.Lock()
	r.samples = append(r.samples, s)
	r.mu.Unlock()
}

type opStats struct {
	Op       string      `json:"op"`
	Count    int         `json:"count"`
	Errors   int         `json:"errors"`
	PerSec   float64     `json:"per_sec"`
	P50ms    float64     `json:"p50_ms"`
	P95ms    float64     `json:"p95_ms"`
	P99ms    float64     `json:"p99_ms"`
	MaxMs    float64     `json:"max_ms"`
	Statuses map[int]int `json:"statuses"`
}

func (r *recorder) stats(elapsed time.Duration) []opStats {
	byOp := map[string][]sample{}
	for _, s := range r.samples {
		byOp[s.op] = append(byOp[s.op], s)
	}
	names := make([]string, 0, len(byOp))
	for name := range byOp {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]opStats, 0, len(names))
	for _, name := range names {
		samples := byOp[name]
		lat := make([]time.Duration, 0, len(samples))
		st := opStats{Op: name, Count: len(samples), Statuses: map[int]int{}}
		for _, s := range samples {
			st.Statuses[s.status]++
			if s.failed {
				st.Errors++
				continue
			}
			lat = append(lat, s.latency)
		}
		sort.Slice(lat, func(i, j int) bool { return lat[i] < lat[j] })
		pick := func(q float64) float64 {
			if len(lat) == 0 {
				return 0
			}
			return float64(lat[min(len(lat)-1, int(q*float64(len(lat))))].Microseconds()) / 1000
		}
		st.PerSec = float64(len(lat)) / elapsed.Seconds()
		st.P50ms, st.P95ms, st.P99ms, st.MaxMs = pick(0.50), pick(0.95), pick(0.99), pick(1)
		out = append(out, st)
	}
	return out
}

func main() {
	var cfg config
	var tokenFile string
	flag.StringVar(&cfg.base, "base", "http://127.0.0.1:8091", "API base URL")
	flag.StringVar(&tokenFile, "token-file", "", "file with the owner token")
	flag.StringVar(&cfg.scenario, "scenario", "write-own", "write-own | write-shared | contend | read")
	flag.IntVar(&cfg.agents, "agents", 10, "concurrent agents")
	flag.DurationVar(&cfg.duration, "duration", 15*time.Second, "measured time")
	flag.Float64Var(&cfg.rate, "rate", 0, "operations per second per agent; 0 means as fast as possible")
	flag.IntVar(&cfg.bodyBytes, "body", 1024, "message body size in bytes")
	flag.IntVar(&cfg.thread, "thread", 50, "thread length for the read scenario")
	flag.IntVar(&cfg.statusPct, "status-pct", 10, "share of status changes among writes, percent")
	flag.Parse()

	raw, err := os.ReadFile(tokenFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "agentbench: -token-file:", err)
		os.Exit(2)
	}
	cfg.token = strings.TrimSpace(string(raw))

	c := &client{base: cfg.base, token: cfg.token, http: &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns: cfg.agents * 2, MaxIdleConnsPerHost: cfg.agents * 2, MaxConnsPerHost: cfg.agents * 2,
		},
	}}

	ctx := context.Background()
	var ws struct {
		ID string `json:"id"`
	}
	c.must(ctx, http.MethodPost, "/workspaces", map[string]string{
		"name": fmt.Sprintf("bench-%s-%d", cfg.scenario, time.Now().UnixNano()),
	}, &ws)

	rec := &recorder{}
	var extra map[string]any
	start := time.Now()
	switch cfg.scenario {
	case "write-own":
		extra = runWrites(ctx, c, cfg, ws.ID, cfg.agents, rec)
	case "write-shared":
		extra = runWrites(ctx, c, cfg, ws.ID, 1, rec)
	case "contend":
		extra = runContend(ctx, c, cfg, ws.ID, rec)
	case "read":
		start, extra = runReads(ctx, c, cfg, ws.ID, rec)
	default:
		fmt.Fprintln(os.Stderr, "agentbench: unknown scenario", cfg.scenario)
		os.Exit(2)
	}
	elapsed := min(time.Since(start), cfg.duration+5*time.Second)
	if measured, ok := extra["measured_seconds"].(float64); ok {
		elapsed = time.Duration(measured * float64(time.Second))
	}

	out, _ := json.Marshal(map[string]any{
		"scenario": cfg.scenario, "agents": cfg.agents, "rate_per_agent": cfg.rate, "body_bytes": cfg.bodyBytes,
		"thread": cfg.thread, "seconds": elapsed.Seconds(), "ops": rec.stats(elapsed), "extra": extra,
	})
	fmt.Println(string(out))
}

func createTaskChats(ctx context.Context, c *client, wsID string, n int) []string {
	ids := make([]string, n)
	for i := range n {
		var chat struct {
			ID string `json:"id"`
		}
		c.must(ctx, http.MethodPost, "/workspaces/"+wsID+"/chats", map[string]any{
			"name": fmt.Sprintf("bench task %d", i), "type": "task", "is_public": true,
		}, &chat)
		ids[i] = chat.ID
	}
	return ids
}

func body(n int) string {
	const words = "agent report build test passed failed review plan task status commit branch "
	var b strings.Builder
	for b.Len() < n {
		b.WriteString(words)
	}
	return b.String()[:n]
}

// pace sleeps so that an agent issues ops at the configured rate; it returns false when time is up.
func pace(deadline time.Time, next *time.Time, rate float64) bool {
	if rate > 0 {
		if wait := time.Until(*next); wait > 0 {
			time.Sleep(wait)
		}
		*next = next.Add(time.Duration(float64(time.Second) / rate))
	}
	return time.Now().Before(deadline)
}

// runWrites lets every agent post messages and change status; chats is how many task threads they share.
func runWrites(ctx context.Context, c *client, cfg config, wsID string, chats int, rec *recorder) map[string]any {
	chatIDs := createTaskChats(ctx, c, wsID, chats)
	statuses := []string{"In Progress", "Done", "To Do"}
	final := make([]string, cfg.agents)
	text := body(cfg.bodyBytes)

	begin := time.Now()
	deadline := begin.Add(cfg.duration)
	var wg sync.WaitGroup
	for agent := range cfg.agents {
		wg.Go(func() {
			chatID := chatIDs[agent%len(chatIDs)]
			rng := rand.New(rand.NewPCG(uint64(agent), 42))
			// Agents start spread over one interval so that paced runs do not fire in bursts.
			next := begin
			if cfg.rate > 0 {
				next = begin.Add(time.Duration(rng.Float64() * float64(time.Second) / cfg.rate))
			}
			turn := 0
			for pace(deadline, &next, cfg.rate) {
				op, path, payload := "message", "/workspaces/"+wsID+"/chats/"+chatID+"/messages", any(map[string]string{"content": text})
				if rng.IntN(100) < cfg.statusPct {
					turn++
					final[agent] = statuses[turn%len(statuses)]
					op, path = "status", "/workspaces/"+wsID+"/chats/"+chatID+"/actions/status"
					payload = map[string]string{"status": final[agent]}
				}
				started := time.Now()
				status, _, err := c.do(ctx, http.MethodPost, path, payload)
				rec.add(sample{op: op, status: status, latency: time.Since(started),
					failed: err != nil || status >= http.StatusBadRequest})
			}
		})
	}
	wg.Wait()
	measured := time.Since(begin)

	extra := map[string]any{"measured_seconds": measured.Seconds(), "chats": chats}
	if chats == cfg.agents {
		settle, pending := settleStatuses(ctx, c, wsID, chatIDs, final)
		extra["status_settle_seconds"], extra["status_not_applied"] = settle.Seconds(), pending
	}
	return extra
}

// runContend makes every agent set a unique due date on one shared task.
// Each accepted request must leave exactly one chat.due_date_set event, so the caller can count lost updates.
// (The rename action cannot serve here: it posts a confirmation message without renaming the chat.)
func runContend(ctx context.Context, c *client, cfg config, wsID string, rec *recorder) map[string]any {
	chatID := createTaskChats(ctx, c, wsID, 1)[0]
	path := "/workspaces/" + wsID + "/chats/" + chatID + "/actions/due-date"
	tomorrow := time.Now().AddDate(0, 0, 1)

	begin := time.Now()
	deadline := begin.Add(cfg.duration)
	var wg sync.WaitGroup
	for agent := range cfg.agents {
		wg.Go(func() {
			next := begin
			for n := 0; pace(deadline, &next, cfg.rate); n++ {
				started := time.Now()
				due := tomorrow.AddDate(0, 0, n*cfg.agents+agent).Format("2006-01-02")
				status, _, err := c.do(ctx, http.MethodPost, path, map[string]string{"due_date": due})
				rec.add(sample{op: "due_date", status: status, latency: time.Since(started),
					failed: err != nil || status >= http.StatusBadRequest})
			}
		})
	}
	wg.Wait()

	return map[string]any{"measured_seconds": time.Since(begin).Seconds(), "chat_id": chatID}
}

// settleStatuses waits until each chat shows the last status its agent asked for; status changes are applied
// after the HTTP response, so the answer to "did it land" is only visible later.
func settleStatuses(ctx context.Context, c *client, wsID string, chatIDs, final []string) (time.Duration, int) {
	started := time.Now()
	pending := 0
	for time.Since(started) < 30*time.Second {
		pending = 0
		for i, chatID := range chatIDs {
			if final[i] == "" {
				continue
			}
			var chat struct {
				Status *string `json:"status"`
			}
			status, data, err := c.do(ctx, http.MethodGet, "/workspaces/"+wsID+"/chats/"+chatID, nil)
			if err != nil || status != http.StatusOK || json.Unmarshal(data, &chat) != nil ||
				chat.Status == nil || *chat.Status != final[i] {
				pending++
			}
		}
		if pending == 0 {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	return time.Since(started), pending
}

// runReads fills one thread to cfg.thread messages, then measures thread pages and lists.
func runReads(ctx context.Context, c *client, cfg config, wsID string, rec *recorder) (time.Time, map[string]any) {
	chatIDs := createTaskChats(ctx, c, wsID, 20)
	text := body(cfg.bodyBytes)
	path := "/workspaces/" + wsID + "/chats/" + chatIDs[0] + "/messages"

	fillStart := time.Now()
	var wg sync.WaitGroup
	jobs := make(chan struct{}, cfg.thread)
	for range cfg.thread {
		jobs <- struct{}{}
	}
	close(jobs)
	var failed errGroupCount
	for range 16 {
		wg.Go(func() {
			for range jobs {
				status, _, err := c.do(ctx, http.MethodPost, path, map[string]string{"content": text})
				if err != nil || status >= http.StatusBadRequest {
					failed.inc()
				}
			}
		})
	}
	wg.Wait()
	fill := time.Since(fillStart)

	targets := []struct{ op, path string }{
		{"thread_page_50_newest", path + "?limit=50"},
		{"thread_page_50_deep", fmt.Sprintf("%s?limit=50&offset=%d", path, max(0, cfg.thread-50))},
		{"chat_list", "/workspaces/" + wsID + "/chats"},
		{"task_list", "/workspaces/" + wsID + "/tasks"},
		{"feed_tasks", "/feed/tasks"},
	}

	begin := time.Now()
	deadline := begin.Add(cfg.duration)
	for agent := range cfg.agents {
		wg.Go(func() {
			next := begin
			for i := agent; pace(deadline, &next, cfg.rate); i++ {
				target := targets[i%len(targets)]
				started := time.Now()
				status, _, err := c.do(ctx, http.MethodGet, target.path, nil)
				rec.add(sample{op: target.op, status: status, latency: time.Since(started),
					failed: err != nil || status >= http.StatusBadRequest})
			}
		})
	}
	wg.Wait()

	return begin, map[string]any{
		"measured_seconds": time.Since(begin).Seconds(), "fill_seconds": fill.Seconds(), "fill_failed": failed.get(),
	}
}

type errGroupCount struct {
	mu sync.Mutex
	n  int
}

func (e *errGroupCount) inc() { e.mu.Lock(); e.n++; e.mu.Unlock() }

func (e *errGroupCount) get() int { e.mu.Lock(); defer e.mu.Unlock(); return e.n }
