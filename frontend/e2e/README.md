# End-to-end tests

Playwright drives the whole app (frontend, backend, realtime, Mongo, Redis) behind the Docker ALB, as a person would.

## Running

Start the stack with mails going to the log, a database of its own and plain-HTTP cookies (overriding the root `.env` on purpose: a `RESEND_API_KEY` there would mail every test account):

```bash
RESEND_API_KEY= MAIL_FROM= MONGO_DATABASE=e2e_ui AUTH_COOKIE_SECURE=false docker compose up -d --build
cd frontend
pnpm exec playwright install chromium   # once
pnpm test:e2e
```

After changing the frontend, rebuild only it, so the backend keeps the overrides: `docker compose up -d --build --no-deps frontend`. Drop the test database afterwards with `db.getSiblingDB('e2e_ui').dropDatabase()` in `mongosh`.

`E2E_BASE_URL` (default `http://localhost`) and `E2E_MONGO_DATABASE` (default `e2e_ui`) point the tests elsewhere. CI runs them in `.github/workflows/e2e-tests.yml` on a fresh stack.

## How the helpers work (`support/stack.ts`)

- `account(name, plan)` registers through the API, reads the verification link from `docker compose logs backend` and confirms it; a plan other than free is set in Mongo through `docker compose exec mongo mongosh`.
- Sign-up and sign-in are limited per client address. Behind the ALB the backend trusts the private network's `X-Forwarded-For` (`TRUSTED_PROXIES`), so each test account sends an address of its own.
- `signIn(page, account)` signs in from the page's own context, so the refresh cookie lands in that browser as after the form.
- `t('group.key')` reads `messages/en.json`, so a copy change does not break a selector.

## What they cover

| Spec | Flow |
|---|---|
| `auth` | E2E-01 sign up, verify, land signed in; E2E-02 sign out, protected pages redirect |
| `boards` | E2E-03 create a board, drop a note from the sidebar, reload; create another from inside a board |
| `cards` | E2E-10 a card whose source is down says what to check, the rest keeps working; a refused save stays in the form |
| `collaboration` | E2E-07 a second member sees moves and new cards live (logs the propagation time); E2E-08 a viewer gets no editing tools |
| `credentials` | E2E-06 a credential value never shows on the page nor in any response |
| `i18n` | E2E-09 the sign-in page switches language; the public pages have an address per language and work without scripts |
