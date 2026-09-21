# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this project is

A universal information dashboard ("tablero de información universal") where users create boards containing **post-its** — configurable cards that fetch data from external APIs, transform it via [gojq](https://github.com/itchyny/gojq) queries, and display it on an [XyFlow](https://svelteflow.dev/) canvas. Boards update live for every connected user.

## Monorepo layout

```
backend/     Go API (Gin + MongoDB + Redis), all routes under /api/v1
realtime/    Node service (socket.io + Mongo change streams + Redis presence), path /ws
frontend/    SvelteKit SPA (Svelte 5 runes, Tailwind CSS 4, TypeScript)
nginx.conf   Docker ALB: /api/ → backend:31126, /ws/ → realtime:3000, / → frontend:80
vercel.json  Vercel ALB equivalent: /api/ → backend, /ws/ → realtime, / → frontend
docker-compose.yaml
```

### Two deployment targets, one topology

The app runs on **Docker (compose)** and on **Vercel**. Both put the three services behind a single origin and route by path prefix (`/api`, `/ws`, `/`). The frontend relies on this: it never needs to know a backend host (see "How the frontend finds the backend").

| | Docker | Vercel |
|---|---|---|
| Router | `alb` (nginx.conf) | `vercel.json` rewrites |
| Frontend build | `ADAPTER=static` → `build/` served by nginx | default `adapter-vercel` → `.vercel/output` |
| Config source | root `.env` (compose builds the connection URLs) | per-service env vars in the Vercel project |

The SvelteKit adapter is selected in `frontend/svelte.config.js`: **Vercel is the default**, and `frontend/dist/Dockerfile` sets `ENV ADAPTER=static`. Keep it this way round so a missing variable can never break the Vercel deployment.

## Running locally

### Full stack (Docker)
```bash
cp .env.example .env   # set the passwords and SECRETS_MASTER_KEYS
docker compose up --build
# App at http://localhost (port 80), or http://<your-LAN-IP> from other machines
```
- Only `alb` publishes a port (80). Mongo, Redis, backend and realtime are reachable only inside docker networks; `mongo` and `redis` networks are `internal: true`.
- To debug a DB from the host, bind it to loopback only, e.g. mongo `ports: ["127.0.0.1:27017:27017"]`. Use `&directConnection=true` in the URI from the host.
- Mongo runs as a single-node replica set (`rs0`, keyfile auth) because change streams require one. The healthcheck initiates it on first boot. An old standalone `mongo_data` volume must be removed (`docker compose down -v`).
- Passwords are embedded in connection URLs, keep them URL-safe (letters, digits, `- _ .`).

### Backend only
```bash
cd backend
go run .                # listens on :$PORT (default 31126)
```
Needs MongoDB, Redis and `SECRETS_MASTER_KEYS` (see `backend/.env.example`). The main package spans `main.go`, `app.go` and `maintenance.go`, so build it as a package (`go run .`, `go build .`), never `go run main.go`.

Maintenance (one-shot, exits afterwards): `go run . -rewrap-keys`, `go run . -rotate-key <kind>:<owner>`, `go run . -rotate-all-keys`.

### Realtime only
```bash
cd realtime
pnpm install
pnpm build:dev && pnpm start:dev   # tsc → node dist/server.js, listens on :$PORT (default 3000)
```
Needs `MONGODB_URI` (replica set) and `REDIS_URL` (see `realtime/.env.example`). `package.json` pins pnpm via `devEngines` (`^12.3.4`, downloaded automatically).

### Frontend only
```bash
cd frontend
pnpm install
pnpm dev                # Vite dev server
```
`pnpm dev` is not behind the ALB, so `/api` and `/ws` do not exist on the dev origin. To use the Docker stack as the API, set `VITE_API_URL=http://localhost` in `frontend/.env` (the ALB on port 80 routes `/api` and `/ws`; leave `VITE_REALTIME_URL` empty). Docker does not publish the backend (31126) or realtime (3000) ports. If you run those services yourself instead, use `VITE_API_URL=http://localhost:31126` and `VITE_REALTIME_URL=http://localhost:3000` (websocket-only transport, so no CORS is needed on realtime). Use `pnpm dev --host` and your LAN IP to open the dev server from another machine.

## Environment variables

| Service | Variable | Notes |
|---|---|---|
| root (compose) | `MONGO_INITDB_ROOT_USERNAME`, `MONGO_INITDB_ROOT_PASSWORD` | Required. Compose builds `MONGODB_URI` from them |
| root (compose) | `MONGO_DATABASE` | Default `prod` |
| root (compose) | `REDIS_PASSWORD` | Required. Compose builds `REDIS_URL` from it |
| root (compose) | `SECRETS_MASTER_KEYS` | Required (backend). `<version>:<base64 of 32 bytes>`, comma-separated for rotation |
| backend | `PORT`, `MONGODB_URI`, `MONGO_DATABASE`, `REDIS_URL`, `SECRETS_MASTER_KEYS` | Defaults: `31126`, `mongo:27017`, `prod`, `redis:6379`. Keys are required |
| realtime | `MONGODB_URI`, `REDIS_URL` | Required, no defaults |
| realtime | `MONGO_DATABASE`, `PORT` | Defaults `prod`, `3000` |
| frontend | `VITE_API_URL` | **Leave empty** (see below). Baked in at build time |
| frontend | `VITE_REALTIME_URL` | **Leave empty.** Optional socket.io origin override, falls back to `VITE_API_URL` then the page origin. Bare origin only |
| frontend | `BUILD_VERSION` | Optional SvelteKit version name |

Each folder has a `.env.example`. Real `.env*` files are git-ignored. On Vercel, set the per-service variables in the project settings (not the root compose ones). The docker frontend build ignores `frontend/.env*` (`frontend/.dockerignore`), so a stray value cannot be baked into the image.

## How the frontend finds the backend

`VITE_API_URL` and `VITE_REALTIME_URL` are empty on purpose. Two places read them:

- `$modules/api.svelte.ts`: `import.meta.env.VITE_API_URL || window.location.href`, then `new URL('/api' + path, base)`. An absolute-path reference keeps only the scheme, host and port of the base, so the page address decides the origin.
- `$modules/sockets.svelte.ts`: `io(VITE_REALTIME_URL || VITE_API_URL || window.location.origin, { path: '/ws' })`. socket.io swaps `http(s)` for `ws(s)` and uses the URL path as the **namespace**, which is why the value must be a **bare origin** (never `http://host/api`, that becomes namespace `/api` and API calls become `/api/api/v1/...`).

Same-origin means one build works on any host (localhost, LAN IP, every Vercel preview URL), needs no CORS, and follows the page's `https` automatically (`wss`). Set them only to reach a service on a different origin.

Over plain HTTP from a LAN address the page is **not a secure context**: `crypto.randomUUID()` is undefined there. Use `uuid()` from `$lib/utils` instead of calling it directly. WebRTC data channels (PeerJS) still work.

## Frontend commands (all run from `frontend/`)

| Command | Purpose |
|---|---|
| `pnpm dev` | Dev server |
| `pnpm build` | Vercel build → `.vercel/output` (`ADAPTER=static pnpm build` → static `build/`) |
| `pnpm check` | svelte-check + TypeScript |
| `pnpm lint` | Prettier + ESLint |
| `pnpm format` | Prettier write |
| `pnpm test` | Vitest (all projects, headless) |
| `pnpm test:unit` | Vitest watch mode |
| `pnpm storybook` | Storybook on port 6006 |

Vitest has three sub-projects: `client` (Playwright/Chromium, `*.svelte.spec.ts`), `server` (Node, `*.spec.ts`), `storybook`.

## Backend commands (run from `backend/`)

```bash
go build ./...
go test ./...                      # unit tests, no services needed
../run_backend_integration.sh      # or .ps1 — real Mongo + Redis + mock OAuth2 via docker compose
```

## Architecture: Backend

Layered Go hexagonal structure. All routes are registered under `/api/v1` (nginx and Vercel forward `/api` untouched, do not strip it).

```
main.go                           bootstrap, maintenance flags, Run
app.go                            newApp(): the whole service graph (shared with integration tests)
maintenance.go                    -rewrap-keys / -rotate-key / -rotate-all-keys
common/
  models/          domain types (Board, PostIts, Secret, SecretScope, SecretRef, Grant, Group, DataKey, OAuth2Material, SystemSecretName)
  infrastructure/  port interfaces (Database, Cache, Executer, SecretStore, KeyStore, GroupStore, ScopePolicy, SecretResolver, ...)
  ports/
    mongo/         Database, SecretStore, KeyStore and GroupStore impls (collections: boards, postit, secrets, data_keys, groups; uuid codec)
    redis/         Cache (post-it results), Locker, HandshakeStore and online peers
    crypto/        Sealer (master key ring = KEK) and Keyring (one wrapped data key per scope)
    oauth/         OAuth2 token endpoint client
    executer/      DewIt — HTTP fetch + gojq transform (json.go, html.go)
    safehttp/      SSRF-safe HTTP client
  services/
    boards/        board CRUD (+ purges the board's and its members' secrets and data keys on delete / on removing a collaborator)
    groups/        groups that own secrets: create, members, delete (+ purge)
    postits/       post-it CRUD + execution + caching + secret injection; well-known definitions
    secrets/       vault: put/list/delete per scope, policy (CanManage/CanView/CanUse), bindings, grants, usable listing, OAuth2 handshakes, platform providers, key rotation
    realtime/      online-peer bookkeeping in Redis
  controllers/
    boards/        Gin routes /v1/boards (+ strands, collaborators, online)
    postits/       Gin routes /v1/post-its
    groups/        Gin routes /v1/groups
    secrets/       Gin routes /v1/{boards,users,groups}/:id/{secrets,oauth2}, /v1/boards/:id/members/:user/*, /v1/boards/:id/secrets/usable, /v1/system/*, /v1/oauth2/*
```

**Post-it execution flow**: `ExecutePostIt` → check Redis cache → `prepare` (map every `$TOKEN` in params/headers/queries to a `SecretRef`: the post-it's `Bindings[TOKEN]` if present, else the board's own scope by name; resolve them as the post-it's `RunAs` principal through `ResolveAs`, which re-checks `CanUse`; then inject the well-known's **system secrets** last, into a clone) → `DewIt.Execute` (HTTP GET to `Resource`, apply query params/headers from `Params`, parse JSON, run `gojq` query) → store result in Redis for `Rate` seconds. A bound secret the principal may no longer use makes the card answer `403`; an unbound board token nobody stored is sent verbatim.

**Post-it writes carry a principal** (`cognito_id` in the body): the caller must be a board member; `bindings` are validated with `CanBind` as the caller; `RunAs` is stamped on create and whenever `params`/`bindings` change. Cards saved before `run_as` existed run as the board owner.

**Well-Knowns** (`services/postits/well-knowns.go`): `wellKnown{template, systemSecrets}` definitions (e.g. `temperature`, `dolar_oficial`, `nasa_apod`). Creating with `WellKnown` fills in resource/query/rate from the template; `Params` provides variable overrides. Placeholders are lowercase (`$credential`, `$api_key`); user secret names are uppercase (`$MY_KEY`). `systemSecrets` maps a placeholder to a `models.SystemSecretName` that is resolved from the system scope at execution time and never stored on the post-it.

**Secrets vault** (`services/secrets`): a secret is `(scope, name)` where `SecretScope{Kind: board|member|user|group|system, Owner}` (`member` owner is `<board>:<user>`: what one user keeps on one board for themselves, purged with the membership; `group` owner is the group id). Values are AES-GCM sealed with the scope's data key (`crypto.Keyring`), which is itself wrapped by the versioned master key ring in `SECRETS_MASTER_KEYS`. `ScopePolicy` (`policy.go`) is the only authorization seam: `CanManage`/`CanView` per scope and `CanUse(principal, board, secret)` = the scope admits the principal **or** one of the secret's `Grants` (`{to: user|group|board, board?}`, stored in the clear) reaches them on that board. `ListUsable(principal, board)` is what a picker offers. The controller's `scoping.principal` is the only place `cognito_id` is read. System-scope routes are open until authentication exists; system secrets are never bindable nor shareable. OAuth2 grants can be self-managed (user brings a client) or obtained through a platform **provider** (`models.OAuthProviders`; client id/secret stored as a system secret of kind `oauth2_client`, hydrated into the grant in memory only). Never return a secret value, a client secret or a token from any handler.

Adding a system secret, a provider or a well-known that needs one: see `.claude/tesis/operations.md` (local notes, not versioned).

## Architecture: Realtime

`realtime/` is a separate Node service, not part of the Go module.

- `mongo.ts` watches the `boards` collection with a change stream (`update` operations only, `fullDocument: updateLookup`). Needs a replica set.
- `socket.ts` runs socket.io at path `/ws`. A client connects with `?board=<id>&peer=<id>`, joins the room `<board>` and receives a `peers` event, then `update` events `{ board, ts }` on every board change.
- `redis.ts` keeps online peers in the set `board:<id>:online` (no TTL). The Go endpoints `PUT/DELETE /boards/:id/online` write the same key but the frontend uses the socket instead.
- Mouse cursors are peer-to-peer through PeerJS (`$modules/rtc.svelte.ts`), bootstrapped with the `peers` list from the socket.

## Architecture: Frontend

SvelteKit SPA (`ssr = false`, `prerender = false`). Svelte 5 **runes mode enforced** for all project files (`compilerOptions: { runes: true }`).

**Path aliases** (defined in `svelte.config.js`):

| Alias | Path |
|---|---|
| `$assets` | `src/lib/assets` |
| `$components` | `src/lib/components` |
| `$modules` | `src/lib/modules` |
| `$services` | `src/lib/services` |
| `$stores` | `src/lib/stores` |
| `$types` | `src/lib/types` |

**Key modules**:

- `$modules/api.svelte.ts` — HTTP helpers (`get`, `post`, `put`, `patch`, `del`). Resolves relative paths against the page origin (see above) and maps network errors to SvelteKit `error(503)`. There is no authentication yet: `cognito_id` is passed explicitly; `CURRENT_USER` (`Messi`) lives here.
- `$modules/realtime.svelte.ts` — `connect(board, onChange)`: opens the socket and the PeerJS mesh; used by `routes/board/[id]/Realtime.svelte`.
- `$modules/sockets.svelte.ts` / `rtc.svelte.ts` — socket.io client and PeerJS cursor sharing.
- `$modules/statefull.svelte.ts` — Preserves and restores arbitrary route state across navigation (`preserve` / `restore`), keyed by `[fromRoute][toRoute]`.
- `$services/{board,post-it,secrets,edge}.ts` — Typed API calls per resource. `secrets.ts` is scope-first (`boardScope`, `memberScope`, `pathOf(scope)`): list/put/delete, self-managed OAuth2 (`put_oauth2`, `authorize`), platform providers (`providers`, `connect`), and `usable(board)`. `post-it.ts` sends `cognito_id` and `bindings`.
- `$lib/secrets/{origin,binding}.ts` — Pure helpers behind the credential picker: where a usable secret comes from (`board | mine | profile | group | shared`) and what a card must send for the picked ones (`$NAME` + a binding for anything outside the board scope).
- `$stores/boards.ts`, `$stores/sidebar.ts`, `$stores/mouses.svelte.ts` — Board list, sidebar open/close, live cursors.
- `$types/api.ts` — API types (`Board`, `PostIt`, `Strand`, `SecretMeta`, `OAuth2Config`, `OAuthProvider`, ...).
- `$components/Nodes/node-map.ts` — Well-known key → Svelte node component, plus the parameter form each one needs (`type: "secret"` renders a picker over the board's credentials). A well-known whose credential is the platform's declares no parameter.
- `$components/Secrets/SecretsPanel.svelte` — Board credentials modal with two tabs, "Board" (shared with every member) and "Only mine" (the caller's member scope); same forms for API keys, self-managed OAuth2 and "Connect an account".
- Routes: `/` (board list), `/board/[id]` (canvas: `Flow`, `Dock`, `DnDProvider`, `Realtime`; the view is keyed by the board id so navigating between boards remounts it), `/oauth2/callback` (provider redirect target; forwards `state`/`code` to the backend).

After editing `messages/{en,es}.json` outside the Vite dev server, regenerate with `npx paraglide-js compile --project ./project.inlang --outdir ./src/lib/paraglide` before `pnpm check`.

**i18n**: Paraglide. Source messages in `frontend/messages/{en,es}.json`. Generated output in `src/lib/paraglide/`. Import messages as `import { m } from '$lib/paraglide/messages'`.

**UI library**: `@xyflow/svelte` for the board canvas. Component primitives in `$components/` (Button, Cursor, Divider, Drawer, Edges, Icon, Input, Modal, Nodes, Secrets, Switch, Toast). Styling via Tailwind CSS 4 + `clsx` + `tailwind-merge` (re-exported as `cn` from `$lib/utils.ts`).

## Svelte MCP (available in frontend/)

A Svelte MCP server is configured (`.mcp.json`). Use these tools when writing Svelte/SvelteKit code:

1. `list-sections` — discover available Svelte 5 / SvelteKit docs; call this first.
2. `get-documentation` — fetch full content for relevant sections.
3. `svelte-autofixer` — validate Svelte code; call until no issues remain before sending to user.
4. `playground-link` — generate playground link only on explicit user request, never when writing to project files.
