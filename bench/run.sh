#!/usr/bin/env bash
# Benchmark runner: seed N stub monitors, run W workers, measure for D seconds.
# Usage: ./run.sh [N=100] [W=2] [D=600]
# Output: results/bench_N${N}_W${W}.json + summary table on stdout.
set -euo pipefail
cd "$(dirname "$0")/.."

N=${1:-100}
W=${2:-2}
D=${3:-600}
RES="bench/results/bench_N${N}_W${W}.json"
mkdir -p bench/results

echo "=== pulse bench N=$N W=$W D=${D}s ==="

echo "--- up core + stub"
docker compose up -d postgres redis api scheduler >/dev/null
docker compose --profile bench up -d stub >/dev/null
sleep 3

echo "--- bench user + clean + seed"
EMAIL="bench${N}_${W}_$(date +%s)@pulse.local"
TOKEN=$(curl -s -X POST localhost:8080/api/v1/auth/register -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"password\":\"bench1234\",\"name\":\"Bench\"}" | python3 -c "import sys,json;print(json.load(sys.stdin)['data']['token'])")
USER_ID=$(curl -s localhost:8080/api/v1/auth/me -H "Authorization: Bearer $TOKEN" | python3 -c "import sys,json;print(json.load(sys.stdin)['data']['id'])")
docker compose exec -T postgres psql -U pulse -d pulse -c "DELETE FROM monitors WHERE name LIKE 'bench-%';" >/dev/null
docker compose exec -T postgres psql -U pulse -d pulse -v N="$N" -v USER_ID="$USER_ID" -f - < bench/seed.sql

echo "--- workers: stop all, start $W, restart scheduler (zero counters)"
docker compose stop worker-1 worker-2 worker-3 worker-4 2>/dev/null || true
for i in $(seq 1 "$W"); do docker compose start "worker-$i" >/dev/null; done
docker compose restart scheduler >/dev/null
sleep 5

echo "--- sample metrics every 30s for ${D}s"
PORTS=""
for i in $(seq 1 "$W"); do PORTS="$PORTS $((9101 + i))"; done
# worker-N -> 9101+N ; api=9101
python3 - "bench/results/samples_N${N}_W${W}.jsonl" "$D" $PORTS <<'EOF'
import json, sys, time, urllib.request
out, dur = sys.argv[1], int(sys.argv[2])
ports = [int(p) for p in sys.argv[3:]]
t0 = time.time()
with open(out, 'w') as f:
    while time.time() - t0 < dur:
        snap = {"t": round(time.time() - t0, 1), "workers": {}}
        for p in ports:
            try:
                txt = urllib.request.urlopen(f"http://localhost:{p}/metrics", timeout=10).read().decode()
                snap["workers"][p] = txt
            except Exception as e:
                snap["workers"][p] = f"ERROR {e}"
        f.write(json.dumps(snap) + "\n")
        f.flush()
        time.sleep(30)
print("samples done")
EOF

echo "--- analyze"
docker compose exec -T postgres psql -U pulse -d pulse -t -A -c \
  "SELECT count(*) FROM monitors WHERE name LIKE 'bench-%' AND last_checked_at < now() - interval '120 seconds';" > /tmp/overdue.txt
docker compose exec -T postgres psql -U pulse -d pulse -t -A -c \
  "SELECT count(*) FROM incidents i JOIN monitors m ON m.id=i.monitor_id WHERE m.name LIKE 'bench-ok-%';" > /tmp/inc_ok.txt
docker compose exec -T postgres psql -U pulse -d pulse -t -A -c \
  "SELECT count(DISTINCT m.id) FROM incidents i JOIN monitors m ON m.id=i.monitor_id WHERE m.name LIKE 'bench-timeout-%' AND i.status='OPEN';" > /tmp/inc_to.txt
python3 - bench/results/samples_N${N}_W${W}.jsonl "$RES" "$N" "$W" "$(cat /tmp/overdue.txt)" "$(cat /tmp/inc_ok.txt)" "$(cat /tmp/inc_to.txt)" <<'EOF'
import json, sys, re
samples_f, res_f, N, W, overdue, inc_ok, inc_to = sys.argv[1], sys.argv[2], int(sys.argv[3]), int(sys.argv[4]), int(sys.argv[5]), int(sys.argv[6]), int(sys.argv[7])

def parse(txt):
    m = {}
    for line in txt.splitlines():
        if line.startswith('#') or 'ERROR' in line:
            continue
        mm = re.match(r'(\w+)(\{[^}]*\})?\s+([0-9.eE+-]+)', line)
        if mm:
            m.setdefault(mm.group(1) + (mm.group(2) or ''), float(mm.group(3)))
    return m

first, last = None, None
maxdepth, maxinflight = 0, 0
for line in open(samples_f):
    s = json.loads(line)
    agg = {}
    for p, txt in s["workers"].items():
        for k, v in parse(txt).items():
            agg[k] = agg.get(k, 0) + v
    if first is None:
        first = agg
    last = agg
    maxdepth = max(maxdepth, agg.get("queue_depth", 0))
    maxinflight = max(maxinflight, agg.get("worker_jobs_inflight", 0))

def delta(k):
    return last.get(k, 0) - first.get(k, 0)

checks = sum(delta(k) for k in last if k.startswith("monitor_checks_total"))
fails = sum(delta(k) for k in last if k.startswith("monitor_checks_total") and "UP" not in k)
dur = (last and first) and None
# elapsed from sample timestamps
ts = [json.loads(l)["t"] for l in open(samples_f)]
el = max(ts) - min(ts) if len(ts) > 1 else 1

res = {
    "N": N, "workers": W, "elapsed_s": round(el, 1),
    "checks_total": int(checks), "checks_per_s": round(checks / el, 2),
    "fail_ratio": round(fails / max(checks, 1), 3),
    "queue_depth_max": int(maxdepth), "inflight_max": int(maxinflight),
    "overdue_monitors": overdue, "ok_monitors_with_incidents": inc_ok,
    "timeout_monitors_open": inc_to,
    "scheduler_enqueued": int(delta("scheduler_enqueued_total")),
}
json.dump(res, open(res_f, "w"), indent=1)
print(f"N={N} W={W}: {res['checks_per_s']}/s, fail={res['fail_ratio']}, qmax={res['queue_depth_max']}, overdue={overdue}, ok_inc={inc_ok}, timeout_open={inc_to}")
EOF
echo "wrote $RES"
