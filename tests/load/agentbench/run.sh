#!/usr/bin/env bash
# Runs the agentbench baseline against an isolated stack: its own MongoDB and Redis containers on
# non-default ports and its own API process. It never touches a stack on 8080/27017/6379.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$ROOT_DIR"

MONGO_PORT="${BENCH_MONGO_PORT:-27117}"
REDIS_PORT="${BENCH_REDIS_PORT:-6479}"
API_PORT="${BENCH_API_PORT:-8091}"
OUT="${BENCH_OUT:-$(mktemp -d)}"
DURATION="${BENCH_DURATION:-15s}"

cleanup() {
	[[ -n "${api_pid:-}" ]] && kill "$api_pid" 2>/dev/null || true
	docker rm -f flowra-bench-mongo flowra-bench-redis >/dev/null 2>&1 || true
}
trap cleanup EXIT

docker rm -f flowra-bench-mongo flowra-bench-redis >/dev/null 2>&1 || true
# The replica set is named and addressed by the bench port, so a driver can never fall through to another mongod.
docker run -d --name flowra-bench-mongo -p "127.0.0.1:$MONGO_PORT:$MONGO_PORT" mongo:6.0 \
	--replSet rsbench --port "$MONGO_PORT" --bind_ip_all >/dev/null
docker run -d --name flowra-bench-redis -p "127.0.0.1:$REDIS_PORT:6379" redis:7-alpine >/dev/null
for _ in $(seq 1 60); do
	docker exec flowra-bench-mongo mongosh --quiet --port "$MONGO_PORT" --eval 'db.adminCommand("ping").ok' >/dev/null 2>&1 && break
	sleep 1
done
docker exec flowra-bench-mongo mongosh --quiet --port "$MONGO_PORT" --eval \
	"rs.initiate({_id:'rsbench',members:[{_id:0,host:'127.0.0.1:$MONGO_PORT'}]}).ok" >/dev/null
for _ in $(seq 1 60); do
	docker exec flowra-bench-mongo mongosh --quiet --port "$MONGO_PORT" --eval 'db.hello().isWritablePrimary' 2>/dev/null | grep -q true && break
	sleep 1
done

go build -o "$OUT/api" ./cmd/api
go build -o "$OUT/agentbench" ./tests/load/agentbench
go build -o "$OUT/leadfeed" ./cmd/leadfeed

token="$(openssl rand -hex 32)"
printf %s "$token" > "$OUT/token"
chmod 600 "$OUT/token"

export MONGODB_URI="mongodb://127.0.0.1:$MONGO_PORT/?replicaSet=rsbench"
export MONGODB_DATABASE=flowra_bench
SERVER_PORT="$API_PORT" REDIS_ADDR="127.0.0.1:$REDIS_PORT" CONFIG_PATH=configs/config.token.yaml \
	AUTH_OWNER_TOKEN_SHA256="$(printf %s "$token" | sha256sum | cut -d' ' -f1)" \
	FLOWRA_WORKER=true LOG_LEVEL=warn "$OUT/api" > "$OUT/api.log" 2>&1 &
api_pid=$!
for _ in $(seq 1 60); do
	curl -fsS -o /dev/null "http://127.0.0.1:$API_PORT/health" 2>/dev/null && break
	sleep 1
done

"$OUT/leadfeed" import tests/fixtures/leadfeed/full.jsonl >/dev/null

mem() {
	printf '{"point":"%s","api_rss_mb":%d,"mongo_mem":"%s","redis_mem":"%s","load1":"%s"}\n' "$1" \
		"$(( $(ps -o rss= -p "$api_pid") / 1024 ))" \
		"$(docker stats --no-stream --format '{{.MemUsage}}' flowra-bench-mongo | cut -d/ -f1 | tr -d ' ')" \
		"$(docker stats --no-stream --format '{{.MemUsage}}' flowra-bench-redis | cut -d/ -f1 | tr -d ' ')" \
		"$(cut -d' ' -f1 /proc/loadavg)"
}
bench() { "$OUT/agentbench" -base "http://127.0.0.1:$API_PORT" -token-file "$OUT/token" -duration "$DURATION" "$@"; }

# contend prints the bench line, then how many of the accepted due-date changes of one shared task left an event.
contend() {
	local line chat accepted applied
	line="$(bench -scenario contend "$@")"
	echo "$line"
	chat="$(sed -E 's/.*"chat_id":"([^"]+)".*/\1/' <<<"$line")"
	accepted="$(sed -E 's/.*"statuses":\{"200":([0-9]+)\}.*/\1/' <<<"$line")"
	sleep 8 # tag execution retries for up to ~3 s inside a 5 s detached timeout
	applied="$(docker exec flowra-bench-mongo mongosh --quiet --port "$MONGO_PORT" flowra_bench --eval \
		"db.events.countDocuments({aggregate_id:'$chat',event_type:'chat.due_date_set'})")"
	printf '{"point":"contend-%s","accepted":%s,"applied":%s,"lost":%s}\n' "$*" "$accepted" "$applied" "$((accepted - applied))"
}

contention() {
	# Lost updates: many agents change one field of one shared task.
	contend -agents 1 -rate 1
	contend -agents 2 -rate 1
	contend -agents 10 -rate 1
	contend -agents 50 -rate 1
}

if [[ "${BENCH_ONLY:-}" == "contend" ]]; then
	contention | tee "$OUT/results.jsonl"
	exit 0
fi

{
	mem idle
	# Profile points: nominal (50 agents, one write per 10 s) and peak (100 agents, one write per second).
	bench -scenario write-own -agents 50 -rate 0.1
	bench -scenario write-own -agents 100 -rate 1
	mem after-paced
	# Ceiling: closed loop, no think time.
	for agents in 1 10 50 100; do bench -scenario write-own -agents "$agents"; done
	bench -scenario write-shared -agents 50
	contention
	mem after-writes
	bench -scenario read -agents 10 -thread 50
	bench -scenario read -agents 10 -thread 2000
	mem after-reads
	docker exec flowra-bench-mongo mongosh --quiet --port "$MONGO_PORT" flowra_bench --eval \
		'const s=db.stats(); print(JSON.stringify({point:"db", messages:db.messages.countDocuments(), events:db.events.countDocuments(), dataSizeMB:Math.round(s.dataSize/1048576), storageSizeMB:Math.round(s.storageSize/1048576), indexSizeMB:Math.round(s.indexSize/1048576)}))'
	printf '{"point":"api-log","version_conflicts":%s,"write_conflict_errors":%s}\n' \
		"$(grep -c 'concurrency conflict in event store' "$OUT/api.log" || true)" \
		"$(grep -c 'WriteConflict' "$OUT/api.log" || true)"
} | tee "$OUT/results.jsonl"

echo "results: $OUT/results.jsonl" >&2
