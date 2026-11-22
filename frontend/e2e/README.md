# End-to-end tests

Playwright drives the whole app (frontend, backend, realtime, Mongo, Redis) behind the Docker ALB, as a person would.

## Running

Start the stack with perf's overlay (a fake data source the card tests fetch, on an address the SSRF guard lets through), mails going to the log, a database of its own and plain-HTTP cookies (overriding the root `.env` on purpose: a `RESEND_API_KEY` there would mail every test account):

```bash
RESEND_API_KEY= MAIL_FROM= MONGO_DATABASE=e2e_ui AUTH_COOKIE_SECURE=false   docker compose -f docker-compose.yaml -f perf/docker-compose.perf.yaml up -d --build
cd frontend
pnpm exec playwright install chromium   # once
pnpm test:e2e
```

After changing the frontend, rebuild only it, so the backend keeps the overrides: `docker compose -f docker-compose.yaml -f perf/docker-compose.perf.yaml up -d --build --no-deps frontend`. Drop the test database afterwards with `db.getSiblingDB('e2e_ui').dropDatabase()` in `mongosh`.

`E2E_BASE_URL` (default `http://localhost`) and `E2E_MONGO_DATABASE` (default `e2e_ui`) point the tests elsewhere. CI runs them in `.github/workflows/e2e-tests.yml` on a fresh stack.

## How the helpers work (`support/stack.ts`)

- `account(name, plan)` registers through the API, reads the verification link from `docker compose logs backend` and confirms it; a plan other than free is set in Mongo through `docker compose exec mongo mongosh`.
- Sign-up and sign-in are limited per client address. Behind the ALB the backend trusts the private network's `X-Forwarded-For` (`TRUSTED_PROXIES`), so each test account sends an address of its own.
- `signIn(page, account)` signs in from the page's own context, so the refresh cookie lands in that browser as after the form.
- `t('group.key')` reads `messages/en.json`, so a copy change does not break a selector.

## What they cover

| Spec | Flow |
|---|---|
| `auth` | Sign up, verify, land signed in; sign out, protected pages redirect |
| `starters` | A board made from the Trip example has its 4 cards and 2 strands; picking an example keeps a name already typed |
| `boards` | Create a board, drop a note from the sidebar, reload; create another from inside a board |
| `cards` | A card whose source is down says what to check, the rest keeps working; a refused save stays in the form |
| `save-as-type` | A custom card is saved as a card type from its drawer; the form opens filled in, and the saved type is offered in the cards sidebar |
| `duplicate` | A card is duplicated from its toolbar with its title and colour, next to it, and stays after a reload |
| `collaboration` | A second member sees moves and new cards live (logs the propagation time); a viewer gets no editing tools |
| `credentials` | A credential value never shows on the page nor in any response |
| `catalog-search` | The cards sidebar narrows its card types to what is typed (case and accents aside), says when nothing matches, and Escape clears it |
| `curl` | A pasted cURL command fills in a custom card (its credential is not kept as text) and can be tried at once; a paste that is not cURL says so |
| `preview` | "Try request" shows the source's answer, a click on a value fills the output, the card shows it; a failed try says why |
| `connect` | A strand drawn from a card's edge can be dropped anywhere on the other card |
| `strand-hint` | The first-strand hint shows with two cards and no strand, and stays gone after drawing one or dismissing it |
| `dates` | A card's date parameters are picked from the date picker and saved as YYYY-MM-DD |
| `i18n` | The sign-in page switches language; the public pages have an address per language and work without scripts |
