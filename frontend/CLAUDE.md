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
- `$services/{board,post-it,secrets,edge,auth,users,orgs,well-knowns,postit-templates}.ts` — Typed API calls per resource; `auth.ts` throws `AuthError` with the backend's `error` code (`$lib/auth/form.ts` turns it into a message). `secrets.ts` is scope-first (`boardScope`, `memberScope`, `pathOf(scope)`): list/put/delete, self-managed OAuth2 (`put_oauth2`, `authorize`), platform providers (`providers`, `connect`), and `usable(board)`. `post-it.ts` sends `bindings`. No service names the caller: the bearer token does.
- `$lib/secrets/{origin,binding}.ts` — Pure helpers behind the credential picker: where a usable secret comes from (`board | mine | profile | group | shared`) and what a card must send for the picked ones (`$NAME` + a binding for anything outside the board scope).
- `$stores/boards.ts`, `$stores/sidebar.ts`, `$stores/mouses.svelte.ts` — Board list, sidebar open/close, live cursors.
- User-scoped stores (orgs, boards, card results, route state, UI flags) reset through `session.onUserChange` when the signed-in user changes or signs out; keep any new per-user cache on it.
- `$stores/org.svelte.ts` — The user's organizations (`ensure()` loads them once, `refresh()` reloads). Which org is being looked at comes from the URL (`page.params.orgId`, none = Personal); `orgOf(board)` files a board of an org the user is not in as personal. The header `OrgSwitcher` navigates to the org's landing. Landing and board loads `remember()` their org (`localStorage` `tiu.org:<userId>`); `last` is where the logo goes from `/profile` and `/orgs`. Paths are built with `landingPath` / `boardPath` from `$lib/helpers/routes.ts`.
- `$types/api.ts` — API types (`Board`, `PostIt`, `Strand`, `SecretMeta`, `OAuth2Config`, `OAuthProvider`, ...).
- `$components/Nodes/node-map.ts` — Well-known key → hand-made Svelte node component; `nodeTypesFor()` falls back to `Nodes/Generic` for any other well-known. What each well-known is (names, parameter form, outputs) comes from `$stores/well-knowns.svelte.ts` (`GET /well-knowns`, loaded by `loadBoard`; `type: "secret"` parameters render a picker over the board's credentials). A well-known whose credential is the platform's declares no parameter. The store also holds the open board's post-it templates (`templates`, `loadBoard(id)` → `GET /boards/:id/postit-templates/usable`, normalised to the `WellKnown` shape by `$lib/well-knowns/normalize.ts`): `all` is everything the board renders, `pickable` what the cards sidebar offers (catalogue + usable templates). `$layouts/PostitTemplates/PostitTemplatesPanel.svelte` (opened by the sidebar's "create" button, tabs Mine / This board / Organization; the Organization tab picks any org the user is in, defaulting to the board's; a new template being written survives tab switches and is created in the tab shown on save, as is a credential in `SecretsPanel`) lists, creates, edits and deletes them; the form's rules and body live in `$lib/well-knowns/draft.ts`.
- One-off cards (own request, no card type; board `type: ""`) are the canvas type `ONE_OFF` (`$lib/well-knowns/one-off.ts`: `nodeType()` maps them in `board-load.ts` and on live updates, plus the form's draft/body/problems). The cards sidebar's first element drops one; `$layouts/OneOffCard/OneOffCardPanel.svelte` creates and edits it (query string and header values are text or a picked credential, sent as `$NAME` + binding); `Nodes/Generic` renders it and `NodeDetail` lists its request.
- Strands (board edges): `$lib/strands/style.ts` is the pure side (palette/arrow/width lists, `strandToEdge` used by `board-load.ts` and live updates, `applyPatch`, `normalizeLabel`); the colours are CSS tokens `--strand-<color>` in `BoardContent/index.css` (one light block, two equal dark ones). `$components/Edges/FloatingEdge.svelte` draws the line border to border, its own `<marker>` arrowheads (the path is trimmed so the tip lands on the card), the label (double-click edits it in place) and, for editors, `StrandToolbar` in an `EdgeToolbar` over the selected strand. It reaches the board through `$lib/strands/editor.ts` (context set by `Flow`: `canEdit`, `select`, `update`, `remove`); `Flow` applies a restyle at once and reverts it if `PATCH /boards/:id/strands/:strand` fails. Double clicks inside the canvas are stopped by d3-zoom before Svelte's delegated handlers, so the label listens natively.
- `$stores/postit-results.svelte.ts` — what each card answered, shared by its node and detail drawer. `watch(id)` (from an `$effect` in `NodeWrapper`, hand-made nodes and `NodeDetail`; ref-counted) keeps it fresh: after the card's `rate` (read once from `GET /post-its/:id/settings`; cards without a request never refetch) plus a random 0–60 s (`$lib/helpers/auto-refresh.ts`, floor 30 s) it `revalidate`s, keeping the old answer on screen until the new one arrives. A hidden tab skips its turn and catches up when shown.
- `$stores/theme.svelte.ts` — the app's own dark/light choice over the system's, in the cookie `tiu-theme` (`light|dark`, absent = follow the system; helpers in `$lib/helpers/theme.ts`). `src/app.html` sets `<html data-theme>` from it before the first paint; the CSS dark blocks (`routes/layout.css`, `BoardContent/index.css`) apply on `data-theme='dark'` or a dark system without `data-theme='light'` (keep both copies equal); `Flow` passes `theme.colorMode` to SvelteFlow. The user menu's `Switch` sets it; choosing what the system already shows deletes the cookie.
- `$components/Share/ShareDialog.svelte` — Who is on a board: the owner adds people by email as editor or viewer, changes roles and removes them; anyone else sees the list and can leave. `Flow` hides the dock, dragging, connecting, deleting and the credentials for viewers (`role` comes from `GET /boards/:id`).
- `$components/Secrets/SecretsPanel.svelte` — Board credentials modal with two tabs, "Board" (shared with every member) and "Only mine" (the caller's member scope); same forms for API keys, self-managed OAuth2 and "Connect an account".
- Routes: group `(app)` (its `+layout.ts` bootstraps the session and redirects anonymous visitors to `/login?next=`; header is `$layouts/TopBar`): `/` (Personal landing), `/boards/[id]` (a personal board), `/organizations/[orgId]` (an org's landing), `/organizations/[orgId]/boards/[id]` (an org board). Both board routes load through `loadBoard` (`$lib/helpers/board-load.ts`, `404` when the board is not filed under that org) and render `$layouts/BoardContent` (`Flow`, `SideBar`, `Realtime`, share dialog; keyed by the board id so navigating between boards remounts it); both landings render `$layouts/Landing`. Also `/profile`, `/orgs` (list, create), `/orgs/[id]` (members and roles, rename, leave, delete), `/oauth2/callback` (provider redirect target; forwards `state`/`code` to the backend). Group `(auth)`: `/login`, `/register`, `/verify?token=`, `/forgot`, `/reset?token=`, `/auth/google/callback` (Google's redirect target; posts `code`/`state` to the backend, then goes to `next`).

After editing `messages/{en,es}.json` outside the Vite dev server, regenerate with `npx paraglide-js compile --project ./project.inlang --outdir ./src/lib/paraglide` before `pnpm check`.

**i18n**: Paraglide. Source messages in `frontend/messages/{en,es}.json`. Generated output in `src/lib/paraglide/`. Import messages as `import { m } from '$lib/paraglide/messages'`.

**UI library**: `@xyflow/svelte` for the board canvas. Component primitives in `$components/` (Button, Cursor, Divider, Drawer, Edges, Icon, Input, Modal, Nodes, OrgSwitcher, Secrets, Share, Switch, Toast). Styling via Tailwind CSS 4 + `clsx` + `tailwind-merge` (re-exported as `cn` from `$lib/utils.ts`).

## Build and test

`pnpm build` is the Vercel build (`.vercel/output`); `ADAPTER=static pnpm build` produces the static `build/` that Docker serves.

Vitest has three sub-projects: `client` (Playwright/Chromium, `*.svelte.spec.ts`), `server` (Node, `*.spec.ts`), `storybook`.
