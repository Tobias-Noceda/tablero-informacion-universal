# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this project is

A universal information dashboard ("tablero de información universal") where users create boards containing **post-its** — configurable cards that fetch data from external APIs, transform it via [gojq](https://github.com/itchyny/gojq) queries, and display it on an [XyFlow](https://svelteflow.dev/) canvas.

## Monorepo layout

```
backend/    Go API (Gin + MongoDB + Redis)
frontend/   SvelteKit SPA (Svelte 5 runes, Tailwind CSS 4, TypeScript)
nginx.conf  ALB: /api/ → backend:80, / → frontend:80
docker-compose.yaml
```

## Running locally

### Full stack (Docker)
```bash
cp .env.example .env   # fill MONGO credentials
docker compose up --build
# App at http://localhost:80
```

### Backend only
```bash
cd backend
go run main.go          # listens on 0.0.0.0:31126
```
Requires MongoDB and Redis reachable (`MONGODB_URI`, `MONGO_DATABASE`, `REDIS_URL`) and `SECRETS_MASTER_KEYS` (`1:<base64 of 32 random bytes>`; see `.env.example`).

Maintenance (one-shot, exits afterwards): `go run . -rewrap-keys`, `go run . -rotate-key <kind>:<owner>`, `go run . -rotate-all-keys`.

### Frontend only
```bash
cd frontend
pnpm install
pnpm dev                # Vite dev server
```
Set `PUBLIC_API_ORIGIN` env var to point at the backend.

## Frontend commands (all run from `frontend/`)

| Command | Purpose |
|---|---|
| `pnpm dev` | Dev server |
| `pnpm build` | Static build → `build/` |
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

Layered Go hexagonal structure:

```
main.go                           bootstrap, maintenance flags, Run
app.go                            newApp(): the whole service graph (shared with integration tests)
maintenance.go                    -rewrap-keys / -rotate-key / -rotate-all-keys
common/
  models/          domain types (Board, PostIts, Secret, SecretScope, DataKey, OAuth2Material, SystemSecretName)
  infrastructure/  port interfaces (Database, Cache, Executer, SecretStore, KeyStore, ScopePolicy, ...)
  ports/
    mongo/         Database, SecretStore and KeyStore impls (collections: boards, postit, secrets, data_keys)
    redis/         Cache, Locker and HandshakeStore impls
    crypto/        Sealer (master key ring = KEK) and Keyring (one wrapped data key per scope)
    oauth/         OAuth2 token endpoint client
    executer/      DewIt — HTTP fetch + gojq transform
    safehttp/      SSRF-safe HTTP client
  services/
    boards/        board CRUD (+ purges the board's secrets and data key on delete)
    postits/       post-it CRUD + execution + caching + secret injection; well-known definitions
    secrets/       vault: put/list/delete per scope, policy, OAuth2 handshakes, platform providers, key rotation
  controllers/
    boards/        Gin routes /v1/boards
    postits/       Gin routes /v1/post-its
    secrets/       Gin routes /v1/{boards,users}/:id/{secrets,oauth2}, /v1/system/*, /v1/oauth2/*
```

**Post-it execution flow**: `ExecutePostIt` → check Redis cache → `prepare` (resolve `$UPPER_NAME` references against the board's secrets, then inject the well-known's **system secrets** last, into a clone) → `DewIt.Execute` (HTTP GET to `Resource`, apply query params/headers from `Params`, parse JSON, run `gojq` query) → store result in Redis for `Rate` seconds.

**Well-Knowns** (`services/postits/well-knowns.go`): `wellKnown{template, systemSecrets}` definitions (e.g. `temperature`, `dolar_oficial`, `nasa_apod`). Creating with `WellKnown` fills in resource/query/rate from the template; `Params` provides variable overrides. Placeholders are lowercase (`$credential`, `$api_key`); user secret names are uppercase (`$MY_KEY`). `systemSecrets` maps a placeholder to a `models.SystemSecretName` that is resolved from the system scope at execution time and never stored on the post-it.

**Secrets vault** (`services/secrets`): a secret is `(scope, name)` where `SecretScope{Kind: board|user|system, Owner}`. Values are AES-GCM sealed with the scope's data key (`crypto.Keyring`), which is itself wrapped by the versioned master key ring in `SECRETS_MASTER_KEYS`. `ScopePolicy` (`policy.go`) is the only authorization seam; the controller's `scoping.principal` is the only place `cognito_id` is read. System-scope routes are open until authentication exists. OAuth2 grants can be self-managed (user brings a client) or obtained through a platform **provider** (`models.OAuthProviders`; client id/secret stored as a system secret of kind `oauth2_client`, hydrated into the grant in memory only). Never return a secret value, a client secret or a token from any handler.

Adding a system secret, a provider or a well-known that needs one: see `.claude/tesis/operations.md` (local notes, not versioned).

## Architecture: Frontend

SvelteKit SPA (static adapter, `index.html` fallback). Svelte 5 **runes mode enforced** for all project files (`compilerOptions: { runes: true }`).

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

- `$modules/api.svelte.ts` — HTTP helpers (`get`, `post`, `put`, `patch`, `del`) against `PUBLIC_API_ORIGIN`. There is no authentication yet: `cognito_id` is passed explicitly (the UI uses the placeholder `Messi`).
- `$modules/statefull.svelte.ts` — Preserves and restores arbitrary route state across navigation (`preserve` / `restore`), keyed by `[fromRoute][toRoute]`.
- `$services/{board,post-it,secrets,edge}.ts` — Typed API calls per resource. `secrets.ts` covers the vault: list/put/delete, self-managed OAuth2 (`put_oauth2`, `authorize`), platform providers (`providers`, `connect`).
- `$stores/boards.ts`, `$stores/sidebar.ts`, `$stores/mouses.svelte.ts` — Board list, sidebar open/close, live cursors.
- `$types/api.ts` — API types (`Board`, `PostIt`, `Strand`, `SecretMeta`, `OAuth2Config`, `OAuthProvider`, ...).
- `$components/Nodes/node-map.ts` — Well-known key → Svelte node component, plus the parameter form each one needs (`type: "secret"` renders a picker over the board's credentials). A well-known whose credential is the platform's declares no parameter.
- `$components/Secrets/SecretsPanel.svelte` — Board credentials modal (API keys, self-managed OAuth2, "Connect an account").
- `routes/oauth2/callback` — Provider redirect target; forwards `state`/`code` to the backend.

After editing `messages/{en,es}.json` outside the Vite dev server, regenerate with `npx paraglide-js compile --project ./project.inlang --outdir ./src/lib/paraglide` before `pnpm check`.

**i18n**: Paraglide. Source messages in `frontend/messages/{en,es}.json`. Generated output in `src/lib/paraglide/`. Import messages as `import { m } from '$lib/paraglide/messages'`.

**UI library**: `$xyflow/svelte` for the board canvas. Component primitives in `$components/` (Button, Drawer, Icon, Input, Switch, Toast, Divider). Styling via Tailwind CSS 4 + `clsx` + `tailwind-merge` (re-exported as `cn` from `$lib/utils.ts`).

## Svelte MCP (available in frontend/)

A Svelte MCP server is configured (`.mcp.json`). Use these tools when writing Svelte/SvelteKit code:

1. `list-sections` — discover available Svelte 5 / SvelteKit docs; call this first.
2. `get-documentation` — fetch full content for relevant sections.
3. `svelte-autofixer` — validate Svelte code; call until no issues remain before sending to user.
4. `playground-link` — generate playground link only on explicit user request, never when writing to project files.
