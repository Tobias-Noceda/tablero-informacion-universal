# Measurements

Scripts that measure the running app for RNF-01 and chapter 8 of the report. Each run writes a JSON file to `results/` (git-ignored) with what the report needs to say where a number comes from: the time, the commit, the machine and the settings. The plan behind them is in `.claude/docs/plan-informe-final.md` (§3.2 bis): each measurement at least 3 times, reporting p50, p95 and p99 and how much the runs differ.

| Script | Measures | Criterion |
|---|---|---|
| `latency.mjs` | RNF-01a: an edit reaching the other members (N sockets on one board, a move per second) and two simultaneous moves of the same card | p95 < 1 s with up to 50 clients (local), < 2 s with 5 (production) |
| `k6/cards.js` | RNF-01b: a card's answer with the cache cold and warm, and that warm reads never reach the source | warm p95 < 50 ms, cold p95 < source delay + 300 ms |
| `k6/load.js` | RNF-01c: the API under 10 → 50 → 100 → 200 users opening a board and running its cards | where p95 takes off (validates the 500 req/s of chapter 8) |
| `capacity.mjs` | Board sockets one realtime task holds (chapter 8 assumes 10,000) | how many open before one is refused |

## Local Docker

The overlay `docker-compose.perf.yaml` adds a fake data source (`source/server.mjs`: answers `/value` after `?delay=` ms and counts what it served), the k6 runner and a Node runner. The backend reads card sources through its SSRF guard, which refuses internal addresses, so the source sits on 203.0.113.0/24 (TEST-NET-3, reserved for documentation and never routed on the internet) and the backend joins that network. The backend itself does not change.

```bash
# From the repository root. Mails go to the log, so account.mjs can read the link; a database of its own.
RESEND_API_KEY= MAIL_FROM= MONGO_DATABASE=perf_local AUTH_COOKIE_SECURE=false \
  docker compose -f docker-compose.yaml -f perf/docker-compose.perf.yaml up -d --build

cd perf && npm install
MONGO_DATABASE=perf_local node account.mjs > .perf.env   # a verified Max account (git-ignored file)
set -a; . ./.perf.env; set +a; cd ..

PERF="docker compose -f docker-compose.yaml -f perf/docker-compose.perf.yaml"
$PERF run --rm -e SAME_CLOCK=1 node latency.mjs
$PERF run --rm k6 run --summary-export=/results/cards.json /scripts/cards.js
$PERF run --rm k6 run --summary-export=/results/load.json /scripts/load.js
$PERF run --rm node capacity.mjs
```

On Git Bash, prefix the `k6` lines with `MSYS_NO_PATHCONV=1`, or it turns `/scripts/...` into a Windows path. Afterwards drop the database (`db.getSiblingDB('perf_local').dropDatabase()` in `mongosh`) and bring the stack back up without the overlay.

Why the Node runner: it runs the scripts from inside the Docker network. Docker Desktop's port forwarding stops at 2000 connections from the host, which `capacity.mjs` would measure instead of the stack. `latency.mjs` can also run from the host (`node latency.mjs`, `BASE_URL=http://localhost`), adding the forwarding as a real browser would see it.

Settings, all optional:

- `latency.mjs`: `CLIENTS=1,10,50,100`, `MOVES=200`, `INTERVAL_MS=1000`, `CONFLICTS=20`. `SAME_CLOCK=1` also splits each latency at the realtime's emission time (`ts`): `to_emit` (backend, Mongo and the change stream) and `to_client` (socket.io). It needs one wall clock on both sides: the Node runner (same kernel as realtime), Docker on Linux, or realtime run natively. From a Windows or macOS host, Docker Desktop's VM clock may differ from the host's; the script notices a negative leg and says to ignore the split.
- `k6/cards.js`: `COLD=50`, `WARM=10`, `WARM_VUS=10`, `WARM_SECONDS=60`, `SOURCE_DELAY_MS=100`.
- `k6/load.js`: `STAGES=10,50,100,200`, `STAGE_SECONDS=120`, `CARDS=10`, `THINK_SECONDS=10`. Under 15 minutes in all: one access token.
- `capacity.mjs`: `MAX=10000`, `STEP=250`, `STEP_LIMIT_MS=30000`.

## Production (Vercel)

Read the cautions in the plan first (§5.7 D: a spending limit on Vercel, Redis Cloud's free tier takes 30 connections). Use a real account on Max (`PUT /system/users/:id/plan` as an admin), never `account.mjs`.

```bash
cd perf
BASE_URL=https://<the app> PERF_EMAIL=… PERF_PASSWORD=… CLIENTS=5,10,25 MOVES=100 node latency.mjs
k6 run -e BASE_URL=https://<the app> -e CARD_KIND=note -e STAGES=10,25 -e STAGE_SECONDS=300 -e PERF_EMAIL=… -e PERF_PASSWORD=… k6/load.js
```

`CARD_KIND=note` uses notes, which run without a request: Vercel's backend cannot reach the fake source, and a third-party API would be tested against its terms. `capacity.mjs` is for local Docker only. Every script deletes the board it made, even when it fails halfway.

## Findings so far

- **The Docker ALB refused board sockets past about 250** (`512 worker_connections are not enough`): `nginx.conf` had an empty `events {}`, so a single worker with 512 connections, two per proxied WebSocket. It now sets `worker_processes auto` and 16,384 connections per worker. After that, one realtime held 10,000 sockets (handshake p95 grows to about 6 s by then: every handshake asks the backend for the board).
- `latency.mjs`, one full run on a laptop (Node runner, 200 moves per level, 0 lost, 0 diverged in 20 conflict rounds each):

  | Clients | from_send p50 / p95 / p99 (ms) | from_ok p95 (ms) |
  |---|---|---|
  | 1 | 5.5 / 7.2 / 8.7 | 0.8 |
  | 10 | 5.3 / 7.1 / 7.9 | 0.8 |
  | 50 | 6.1 / 8.2 / 9.3 | 2.0 |
  | 100 | 6.9 / 9.2 / 10.2 | 3.6 |

  Split (short run): about 5 ms from the request to the realtime's emission (backend, Mongo, change stream) and 1–3 ms from there to the clients. Two orders of magnitude under the 1 s criterion; the report still needs the 3 runs and production.
- **Production** (`https://corcheria.company`, 2026-11-22, from a laptop in Buenos Aires over home internet): one full `latency.mjs` run, 100 moves per level, 0 lost, 0 diverged in 20 conflict rounds each. The plan's production criterion is p95 < 2 s.

  | Clients | from_send p50 / p95 / p99 (ms) | from_ok p95 (ms) |
  |---|---|---|
  | 5 | 69 / 92 / 167 | 3.6 |
  | 10 | 64 / 90 / 111 | 3.0 |
  | 25 | 67 / 114 / 174 | 3.5 |

  A first smoke run (5 clients, 20 moves) had p95 ≈ 500 ms: cold starts. A k6 smoke run (2 users, 30 s, notes only) had p95 ≈ 95 ms and no errors. The team stopped there; the other runs were not needed. Raw JSON stays in `results/` on the machine that ran them.
- `k6/cards.js` on a laptop: cold p95 ≈ source delay + 5 ms, warm p95 ≈ 12 ms, and the source saw exactly one request per cold card.
