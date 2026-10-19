You are able to use the Svelte MCP server, where you have access to comprehensive Svelte 5 and SvelteKit documentation. Here's how to use the available tools effectively:

## Available Svelte MCP Tools:

### 1. list-sections

Use this FIRST to discover all available documentation sections. Returns a structured list with titles, use_cases, and paths.
When asked about Svelte or SvelteKit topics, ALWAYS use this tool at the start of the chat to find relevant sections.

### 2. get-documentation

Retrieves full documentation content for specific sections. Accepts single or multiple sections.
After calling the list-sections tool, you MUST analyze the returned documentation sections (especially the use_cases field) and then use the get-documentation tool to fetch ALL documentation sections that are relevant for the user's task.

### 3. svelte-autofixer

Analyzes Svelte code and returns issues and suggestions.
You MUST use this tool whenever writing Svelte code before sending it to the user. Keep calling it until no issues or suggestions are returned.

### 4. playground-link

Generates a Svelte Playground link with the provided code.
After completing the code, ask the user if they want a playground link. Only call this tool after user confirmation and NEVER if code was written to files in their project.

## Architecture

SvelteKit SPA (`ssr = false`, `prerender = false`). Svelte 5 **runes mode enforced** for all project files (`compilerOptions: { runes: true }`).

**Key modules**:

- `$modules/api.svelte.ts` — HTTP helpers (`get`, `post`, `put`, `patch`, `del`) over one `request()`. Resolves relative paths against the page origin (see above), adds the session's `Authorization: Bearer`, renews once on a `401` (never for `/v1/auth/*`) and sends an expired session to `/login?next=`; `get` maps network errors to SvelteKit `error(503)`.
- `$modules/session.svelte.ts` — `session`: `user`, `status` (`unknown | anonymous | authenticated`), the access token in memory only. `bootstrap()` trades the httpOnly refresh cookie for a session once; `refresh()` is single-flight (Web Locks across tabs when available); `BroadcastChannel('tiu-session')` shares sign-in and sign-out between tabs.
- `$modules/realtime.svelte.ts` — `connect(board, user, onChange)`: opens the socket and the PeerJS mesh (`user` is only the cursor's name and picture; the socket knows the user from the token); used by `$layouts/BoardContent/Realtime.svelte`.
- `$modules/sockets.svelte.ts` — the board socket: sends `session.accessToken` on every handshake, answers `token_expiring` (renewing the session when its token is about to expire), reconnects when the server drops it and gives up after one refused handshake retried with a renewed session (the board keeps working without live updates).
- `$modules/rtc.svelte.ts` — PeerJS cursor sharing.
- `$modules/statefull.svelte.ts` — Preserves and restores arbitrary route state across navigation (`preserve` / `restore`), keyed by `[fromRoute][toRoute]`.
- `$services/{board,post-it,secrets,edge,auth,users,orgs}.ts` — Typed API calls per resource; `auth.ts` throws `AuthError` with the backend's `error` code (`$lib/auth/form.ts` turns it into a message). `secrets.ts` is scope-first (`boardScope`, `memberScope`, `pathOf(scope)`): list/put/delete, self-managed OAuth2 (`put_oauth2`, `authorize`), platform providers (`providers`, `connect`), and `usable(board)`. `post-it.ts` sends `bindings`. No service names the caller: the bearer token does.
- `$lib/secrets/{origin,binding}.ts` — Pure helpers behind the credential picker: where a usable secret comes from (`board | mine | profile | group | shared`) and what a card must send for the picked ones (`$NAME` + a binding for anything outside the board scope).
- `$stores/boards.ts`, `$stores/sidebar.ts`, `$stores/mouses.svelte.ts` — Board list, sidebar open/close, live cursors.
- User-scoped stores (orgs, boards, card results, route state, UI flags) reset through `session.onUserChange` when the signed-in user changes or signs out; keep any new per-user cache on it.
- `$stores/org.svelte.ts` — The user's organizations (`ensure()` loads them once, `refresh()` reloads). Which org is being looked at comes from the URL (`page.params.orgId`, none = Personal); `orgOf(board)` files a board of an org the user is not in as personal. The header `OrgSwitcher` navigates to the org's landing. Landing and board loads `remember()` their org (`localStorage` `tiu.org:<userId>`); `last` is where the logo goes from `/profile` and `/orgs`. Paths are built with `landingPath` / `boardPath` from `$lib/helpers/routes.ts`.
- `$types/api.ts` — API types (`Board`, `PostIt`, `Strand`, `SecretMeta`, `OAuth2Config`, `OAuthProvider`, ...).
- `$components/Nodes/node-map.ts` — Well-known key → Svelte node component, plus the parameter form each one needs (`type: "secret"` renders a picker over the board's credentials). A well-known whose credential is the platform's declares no parameter.
- `$components/Share/ShareDialog.svelte` — Who is on a board: the owner adds people by email as editor or viewer, changes roles and removes them; anyone else sees the list and can leave. `Flow` hides the dock, dragging, connecting, deleting and the credentials for viewers (`role` comes from `GET /boards/:id`).
- `$components/Secrets/SecretsPanel.svelte` — Board credentials modal with two tabs, "Board" (shared with every member) and "Only mine" (the caller's member scope); same forms for API keys, self-managed OAuth2 and "Connect an account".
- Routes: group `(app)` (its `+layout.ts` bootstraps the session and redirects anonymous visitors to `/login?next=`; header is `$layouts/TopBar`): `/` (Personal landing), `/boards/[id]` (a personal board), `/organizations/[orgId]` (an org's landing), `/organizations/[orgId]/boards/[id]` (an org board). Both board routes load through `loadBoard` (`$lib/helpers/board-load.ts`, `404` when the board is not filed under that org) and render `$layouts/BoardContent` (`Flow`, `SideBar`, `Realtime`, share dialog; keyed by the board id so navigating between boards remounts it); both landings render `$layouts/Landing`. Also `/profile`, `/orgs` (list, create), `/orgs/[id]` (members and roles, rename, leave, delete), `/oauth2/callback` (provider redirect target; forwards `state`/`code` to the backend). Group `(auth)`: `/login`, `/register`, `/verify?token=`, `/forgot`, `/reset?token=`, `/auth/google/callback` (Google's redirect target; posts `code`/`state` to the backend, then goes to `next`).

After editing `messages/{en,es}.json` outside the Vite dev server, regenerate with `npx paraglide-js compile --project ./project.inlang --outdir ./src/lib/paraglide` before `pnpm check`.

**i18n**: Paraglide. Source messages in `frontend/messages/{en,es}.json`. Generated output in `src/lib/paraglide/`. Import messages as `import { m } from '$lib/paraglide/messages'`.

**UI library**: `@xyflow/svelte` for the board canvas. Component primitives in `$components/` (Button, Cursor, Divider, Drawer, Edges, Icon, Input, Modal, Nodes, OrgSwitcher, Secrets, Share, Switch, Toast). Styling via Tailwind CSS 4 + `clsx` + `tailwind-merge` (re-exported as `cn` from `$lib/utils.ts`).

## Build and test

`pnpm build` is the Vercel build (`.vercel/output`); `ADAPTER=static pnpm build` produces the static `build/` that Docker serves.

Vitest has three sub-projects: `client` (Playwright/Chromium, `*.svelte.spec.ts`), `server` (Node, `*.spec.ts`), `storybook`.
