# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this project is

A universal information dashboard ("tablero de información universal") where users create boards containing **post-its** — configurable cards that fetch data from external APIs, transform it via [gojq](https://github.com/itchyny/gojq) queries, and display it on an [XyFlow](https://svelteflow.dev/) canvas. Boards update live for every connected user.

## Monorepo layout

`backend/` (Go API), `realtime/` (Node socket service) and `frontend/` (SvelteKit SPA). `frontend/CLAUDE.md` and `realtime/CLAUDE.md` hold the notes for those services; they load when working in that folder.

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
Needs MongoDB, Redis, `SECRETS_MASTER_KEYS` and `AUTH_SIGNING_KEYS` (see `backend/.env.example`). The main package spans `main.go`, `app.go` and `maintenance.go`, so build it as a package (`go run .`, `go build .`), never `go run main.go`.

Maintenance (one-shot, exits afterwards): `go run . -rewrap-keys`, `go run . -rotate-key <kind>:<owner>`, `go run . -rotate-all-keys`.

### Realtime only
```bash
cd realtime
pnpm install
pnpm build:dev && pnpm start:dev   # tsc → node dist/server.js, listens on :$PORT (default 3000)
```
Needs `MONGODB_URI` (replica set), `REDIS_URL` and `API_URL` (see `realtime/.env.example`). `pnpm test` compiles and runs `node --test` on `src/*.test.ts` (no services needed), `pnpm lint` runs Prettier and ESLint; CI runs both (`.github/workflows/realtime-tests.yml`). `package.json` pins pnpm via `devEngines` (`^12.3.4`, downloaded automatically).

### Frontend only
```bash
cd frontend
pnpm install
pnpm dev                # Vite dev server
```
`pnpm dev` is not behind the ALB, so `/api` and `/ws` do not exist on the dev origin. To use the Docker stack as the API, set `VITE_API_URL=http://localhost` in `frontend/.env` (the ALB on port 80 routes `/api` and `/ws`; leave `VITE_REALTIME_URL` empty). Docker does not publish the backend (31126) or realtime (3000) ports. If you run those services yourself instead, use `VITE_API_URL=http://localhost:31126` and `VITE_REALTIME_URL=http://localhost:3000` (websocket-only transport, so no CORS is needed on realtime). Use `pnpm dev --host` and your LAN IP to open the dev server from another machine. With `VITE_API_URL` pointing at another origin the refresh cookie is not sent (fetch only sends cookies same-origin), so signing in works but a reload signs you out.

## Environment variables

| Service | Variable | Notes |
|---|---|---|
| root (compose) | `MONGO_INITDB_ROOT_USERNAME`, `MONGO_INITDB_ROOT_PASSWORD` | Required. Compose builds `MONGODB_URI` from them |
| root (compose) | `MONGO_DATABASE` | Default `prod` |
| root (compose) | `REDIS_PASSWORD` | Required. Compose builds `REDIS_URL` from it |
| root (compose) | `SECRETS_MASTER_KEYS` | Required (backend). `<version>:<base64 of 32 bytes>`, comma-separated for rotation; the highest version encrypts |
| root (compose) | `AUTH_SIGNING_KEYS` | Required (backend). Ed25519 seeds `<kid>:<base64 of 32 bytes>`, comma-separated; the first signs, all verify |
| root (compose) | `PLATFORM_ADMINS`, `AUTH_COOKIE_SECURE`, `GOOGLE_SIGNIN_CLIENT_ID`, `GOOGLE_SIGNIN_CLIENT_SECRET`, `GOOGLE_ISSUER` | Passed to the backend (below) |
| backend | `PORT`, `MONGODB_URI`, `MONGO_DATABASE`, `REDIS_URL`, `SECRETS_MASTER_KEYS`, `AUTH_SIGNING_KEYS` | Defaults: `31126`, `mongo:27017`, `prod`, `redis:6379`. Keys are required |
| backend | `PLATFORM_ADMINS` | Emails that are platform admins when they sign in |
| backend | `AUTH_COOKIE_SECURE`, `ACCESS_TTL`, `REFRESH_TTL` | Defaults `true`, `15m`, `720h`. `false` only for plain-HTTP development; keep `true` on Vercel |
| backend | `GOOGLE_SIGNIN_CLIENT_ID`, `GOOGLE_SIGNIN_CLIENT_SECRET`, `GOOGLE_ISSUER` | Optional "Sign in with Google" (redirect URI `<origin>/auth/google/callback`); empty turns it off |
| realtime | `MONGODB_URI`, `REDIS_URL`, `API_URL` | Required, no defaults. `API_URL` is the backend with `/api/v1` (compose: `http://backend:31126/api/v1`) |
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

## Backend commands (run from `backend/`)

```bash
go build ./...
go test ./...                      # unit tests, no services needed
../run_backend_integration.sh      # or .ps1 — real Mongo + Redis + mock OAuth2 via docker compose
```

## Architecture: Backend

Layered Go hexagonal structure. All routes are registered under `/api/v1` (nginx and Vercel forward `/api` untouched, do not strip it). Only `/auth/*` is public: `app.go` mounts every other controller on a group behind `middleware.RequireAuth`, handlers read the caller with `middleware.Principal(c)` (never from the body or the query), and `/system/*` adds `middleware.RequireAdmin`. Anything the caller may not touch answers `404`.

**Authentication**: email/password (argon2id, email verified through a mailed link) or Google sign-in. A session is a 15-minute EdDSA access token, kept by the frontend in memory and sent as `Authorization: Bearer`, plus a rotating refresh token in Redis carried by the httpOnly cookie `tiu_refresh` (only `/auth/refresh` and `/auth/logout` read it, and refuse cross-site requests). The principal is `{ID: sub, Admin}`; `Admin` comes from `PLATFORM_ADMINS` at sign-in. Login, registration and reset mails are limited per client address and per account, adding people by email (boards and organizations share it) per user: `429 rate_limited`.

**Post-it execution flow**: `ExecutePostIt` → check Redis cache → `prepare` (map every `$TOKEN` in params/headers/queries to a `SecretRef`: the post-it's `Bindings[TOKEN]` if present, else the board's own scope by name; resolve them as the post-it's `RunAs` principal through `ResolveAs`, which re-checks `CanUse`; then inject the well-known's **system secrets** last, into a clone) → `DewIt.Execute` (HTTP GET to `Resource`, apply query params/headers from `Params`, parse JSON, run `gojq` query) → store result in Redis for `Rate` seconds. A bound secret the principal may no longer use makes the card answer `403`; an unbound board token nobody stored is sent verbatim.

**Board roles** (`models.BoardRole`, `owner > editor > viewer`): the owner is `Board.Owner`, everyone else is in `Board.Members` with `editor` or `viewer`. Every check asks `infrastructure.Access` (`services/access`, the one place a role is decided) and compares with `AtLeast`. Viewers read the board, its members and card results; editors also draw strands and create, edit, move and delete cards; the owner also renames, deletes and shares (by email, `PUT /boards/:id/members`). Anyone may leave. Below the role a route needs, the board answers `404`. Board responses carry the caller's `role`. **Organizations** (`models.Org`, roles `admin|member`) are a minimal tenant: a board or group created with `org` (by someone in it) reaches every member without being shared. On an org board, admins act as owners and members as viewers; the higher of that and the board's own role wins, so promoting a member is an explicit `PUT /boards/:id/members`. In an org group, admins act as owners and members only see it (`GroupViewer`): a group's secrets stay with the people it lists. An org keeps at least one admin (enforced in the Mongo filter) and is deleted only once no board or group names it (`409 org_not_empty`). **Post-it writes act as the caller**: they must be an editor; `bindings` are validated with `CanBind` as the caller; `RunAs` is stamped on create and whenever `params`/`bindings` change. Cards saved before `run_as` existed run as the board owner.

**Well-Knowns** (`services/postits/well-knowns.go`): `wellKnown{template, systemSecrets}` definitions (e.g. `temperature`, `dolar_oficial`, `nasa_apod`). Creating with `WellKnown` fills in resource/query/rate from the template; `Params` provides variable overrides. Placeholders are lowercase (`$credential`, `$api_key`); user secret names are uppercase (`$MY_KEY`). `systemSecrets` maps a placeholder to a `models.SystemSecretName` that is resolved from the system scope at execution time and never stored on the post-it.

**Secrets vault** (`services/secrets`): a secret is `(scope, name)` where `SecretScope{Kind: board|member|user|group|system, Owner}` (`member` owner is `<board>:<user>`: what one user keeps on one board for themselves, purged with the membership; `group` owner is the group id). Values are AES-GCM sealed with the scope's data key (`crypto.Keyring`), which is itself wrapped by the versioned master key ring in `SECRETS_MASTER_KEYS`. `ScopePolicy` (`policy.go`) is the only authorization seam: `CanManage`/`CanView` per scope (board scope: owner manages, editors view; member scope: that user, as editor) and `CanUse(principal, board, secret)` = the principal edits the board **and** the scope admits them **or** one of the secret's `Grants` (`{to: user|group|board, board?}`, stored in the clear) reaches them on that board. A viewer never binds, and demoting someone to viewer stops the cards that run as them (`403`). `ListUsable(principal, board)` is what a picker offers. The system scope is for platform admins only (`Principal.Admin`, seeded from `PLATFORM_ADMINS`), in the policy and on the routes; system secrets are never bindable nor shareable. An OAuth2 consent can only be finished (`/oauth2/callback`) by the user who started it. OAuth2 grants can be self-managed (user brings a client) or obtained through a platform **provider** (`models.OAuthProviders`; client id/secret stored as a system secret of kind `oauth2_client`, hydrated into the grant in memory only). Never return a secret value, a client secret or a token from any handler.

Adding a system secret, a provider or a well-known that needs one: see `.claude/tesis/operations.md` (local notes, not versioned).
