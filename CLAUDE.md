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
Requires MongoDB and Redis reachable (see `backend/common/models/envs.go` for env var names).

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
go test ./...
```

## Architecture: Backend

Layered Go hexagonal structure:

```
main.go                           wire-up only
common/
  models/          domain types (Board, PostIts, Position, Envs)
  infrastructure/  port interfaces (Database, Cache, Executer)
  ports/
    mongo/         Database impl
    redis/         Cache impl
    executer/      DewIt — HTTP fetch + gojq transform
  services/
    boards/        board CRUD
    postits/       post-it CRUD + execution + caching
  controllers/
    boards/        Gin routes /v1/boards
    postits/       Gin routes /v1/post-its
```

**Post-it execution flow**: `ExecutePostIt` → check Redis cache → `DewIt.Execute` (HTTP GET to `Resource`, apply query params/headers from `Params`, parse JSON, run `gojq` query) → store result in Redis for `Rate` seconds.

**Well-Knowns** (`services/postits/well-knowns.go`): pre-configured post-it templates (e.g. `temperature`, `dolar_oficial`, `dog_facts`). Creating with `WellKnown` key fills in resource/query/rate automatically; `Params` provides variable overrides.

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

- `$modules/api.svelte.ts` — All HTTP helpers (`get`, `getAuth`, `post`, `postAuth`, `put`, `patch`, `del`, `deleteAuth`, `fetchWithAuth`). Manages JWT access/refresh tokens in `localStorage` via `$state` rune. Token auto-selects access if not expired, falls back to refresh. `login()` / `logout()` / `refreshToken()` here.
- `$modules/statefull.svelte.ts` — Preserves and restores arbitrary route state across navigation (`preserve` / `restore`), keyed by `[fromRoute][toRoute]`.
- `$stores/user.ts` — `user` (JWT payload) and `userData` (full Doctor/Patient from API) writable stores.
- `$stores/sidebar.ts` — Sidebar open/close state.
- `$types/api.ts` — All API types (`Doctor`, `Patient`, `Appointment`, `Study`, `Paginated<T>`, `UriTemplate`, `Session`, etc.).

**i18n**: Paraglide. Source messages in `frontend/messages/{en,es}.json`. Generated output in `src/lib/paraglide/`. Import messages as `import { m } from '$lib/paraglide/messages'`.

**UI library**: `$xyflow/svelte` for the board canvas. Component primitives in `$components/` (Button, Drawer, Icon, Input, Switch, Toast, Divider). Styling via Tailwind CSS 4 + `clsx` + `tailwind-merge` (re-exported as `cn` from `$lib/utils.ts`).

## Svelte MCP (available in frontend/)

A Svelte MCP server is configured (`.mcp.json`). Use these tools when writing Svelte/SvelteKit code:

1. `list-sections` — discover available Svelte 5 / SvelteKit docs; call this first.
2. `get-documentation` — fetch full content for relevant sections.
3. `svelte-autofixer` — validate Svelte code; call until no issues remain before sending to user.
4. `playground-link` — generate playground link only on explicit user request, never when writing to project files.
