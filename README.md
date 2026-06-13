# tablero-informacion-universal

A universal information dashboard. Users create **boards** containing **post-its** — configurable cards that fetch data from external APIs, transform it with [gojq](https://github.com/itchyny/gojq) queries, and display the result on a [SvelteFlow](https://svelteflow.dev/) canvas.

## Repo layout

```
backend/         Go API (Gin + MongoDB + Redis)
frontend/        SvelteKit SPA (Svelte 5 runes, Tailwind 4, TypeScript)
nginx.conf       Edge proxy: /api/* → backend, /* → frontend
docker-compose.yaml
```

## Quick start (Docker, recommended)

One command brings up the whole stack — frontend, backend, MongoDB, Redis, edge proxy.

### 1. Prerequisites

- Docker + Docker Compose v2
- Free ports: `80` (UI + API), `27017` (Mongo), `6379` (Redis), `31126` (backend direct)

### 2. Create `.env`

`docker-compose.yaml` reads three Mongo variables. The backend connects with a hardcoded URL `mongodb://user:password@mongo:27017/?authSource=admin` (see `backend/common/ports/mongo/mon.go`), so the values must match:

```bash
cat > .env <<'EOF'
MONGO_INITDB_ROOT_USERNAME=user
MONGO_INITDB_ROOT_PASSWORD=password
MONGO_INITDB_DATABASE=prod
EOF
```

### 3. Bring it up

```bash
docker compose up --build
```

When the containers settle:

- **UI**: <http://localhost/>
- **API** (through the edge proxy): `http://localhost/api/v1/...`
- **API** (direct to backend): `http://localhost:31126/v1/...`

### 4. Try the UI

1. Click the menu icon (top-left) to open the sidebar.
2. On the home page, click **Create a new board**, give it a name, confirm. The new board appears in the sidebar and you're navigated into it.
3. Drag a **Note** or **Smart Card** from the dock onto the canvas.

### 5. Try the API

There's no auth yet — `Messi` is the placeholder user id used by the UI and the examples below.

```bash
# create a board
curl -X POST http://localhost/api/v1/boards \
  -H 'Content-Type: application/json' \
  -d '{"name":"my first board","owner":"Messi"}'

# list boards for that user
curl 'http://localhost/api/v1/boards?cognito_id=Messi'

# fetch a single board
curl http://localhost/api/v1/boards/<board-id>
```

### 6. Tear down

```bash
docker compose down            # stop containers, keep data
docker compose down -v         # also drop the Mongo volume
```

## Running pieces individually (dev loop)

Use these when you want hot reload or to iterate on one side without rebuilding containers.

### Backend (Go)

```bash
cd backend
go run main.go                  # listens on 0.0.0.0:31126
```

The server expects Mongo at `mongo:27017` and Redis at `redis:6379` (hostnames are hardcoded in `backend/common/ports/mongo/mon.go` and `backend/common/ports/redis/redis.go`). The simplest setup:

```bash
docker compose up mongo redis
```

…and add `127.0.0.1 mongo redis` to `/etc/hosts`, or edit those two constants to `localhost`.

Tests + coverage:

```bash
./run_backend_coverage.sh        # macOS / Linux
.\run_backend_coverage.ps1       # Windows
```

### Frontend (SvelteKit)

```bash
cd frontend
pnpm install
pnpm dev                         # http://localhost:5173/
```

The dev server needs to reach the API. Two options:

- **Easiest** — keep the docker stack up (`docker compose up backend mongo redis alb`) and access the UI through the edge proxy at `http://localhost/`. The Vite dev server on `:5173` does not proxy `/v1/*` out of the box.
- **Direct** — change the base URL in `src/lib/modules/api.svelte.ts` to point at `http://localhost:31126`.

Other useful scripts:

```bash
pnpm build         # production build to ./build (adapter-static)
pnpm preview       # serve the production build locally
pnpm test          # vitest
pnpm check         # svelte-check + tsc
pnpm storybook     # component playground on :6006
```

## API surface

All routes are prefixed with `/v1` on the backend, or `/api/v1` through the edge proxy.

| Method   | Path                                   | Purpose                                  |
|----------|----------------------------------------|------------------------------------------|
| `GET`    | `/v1/boards?cognito_id=...`            | List boards for a user                   |
| `POST`   | `/v1/boards`                           | Create a board                           |
| `GET`    | `/v1/boards/:id`                       | Get one board                            |
| `DELETE` | `/v1/boards/:id`                       | Delete a board                           |
| `PATCH`  | `/v1/boards/:id/name`                  | Rename a board                           |
| `GET`    | `/v1/boards/:id/post-its`              | List post-its on a board                 |
| `POST`   | `/v1/boards/:id/collaborators`         | Share a board                            |
| `DELETE` | `/v1/boards/:id/collaborators`         | Unshare a board                          |
| `POST`   | `/v1/boards/:id/strands`               | Connect two post-its                     |
| `DELETE` | `/v1/boards/:id/strands`               | Disconnect two post-its                  |
| `POST`   | `/v1/post-its`                         | Create a post-it                         |
| `GET`    | `/v1/post-its/:id`                     | Execute a post-it (fetch + jq transform) |
| `DELETE` | `/v1/post-its/:id`                     | Delete a post-it                         |
| `GET`    | `/v1/post-its/:id/settings`            | Get post-it config                       |
| `PATCH`  | `/v1/post-its/:id/settings`            | Update post-it config                    |
| `PATCH`  | `/v1/post-its/:id/position`            | Move a post-it on the canvas             |

## Troubleshooting

- **`docker compose up` errors about missing Mongo env vars** — you forgot the `.env` file (step 2).
- **Frontend loads but API calls 404 / CORS-fail in dev** — you're hitting Vite directly (`:5173`), which doesn't proxy `/v1/*`. Use the edge proxy at `http://localhost/` or set up a Vite proxy.
- **`Cannot read properties of undefined (reading 'data')` on the production build** — usually a stale browser cache. Hard-reload (Cmd-Shift-R) or open in incognito.
- **Mongo/Redis "connection refused" when running `go run main.go` directly** — the URLs are hardcoded to the docker hostnames; see the backend dev setup above.
