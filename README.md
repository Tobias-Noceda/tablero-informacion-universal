# tablero-informacion-universal

A universal information dashboard. Users create **boards** containing **post-its**: configurable cards that fetch data from external APIs, transform it with [gojq](https://github.com/itchyny/gojq) queries and display the result on a [SvelteFlow](https://svelteflow.dev/) canvas. Boards update live for everyone on them, can be shared by role or through an organization, and cards can use credentials kept in an encrypted vault.

## Repo layout

```
backend/     Go API (Gin + MongoDB + Redis), every route under /api/v1
realtime/    Node service (socket.io + Mongo change streams + Redis presence), path /ws
frontend/    SvelteKit SPA (Svelte 5 runes, Tailwind 4, TypeScript)
nginx.conf   Docker router: /api/ → backend, /ws/ → realtime, / → frontend
vercel.json  The same routing on Vercel
docker-compose.yaml
```

The three services live behind one origin and are routed by path, so the frontend never needs to know where the API is. [CLAUDE.md](CLAUDE.md) has the architecture in depth.

## Quick start (Docker)

### 1. Prerequisites

Docker with Compose v2 and a free port `80`. Only the router publishes a port; Mongo, Redis, the backend and realtime stay on private networks.

### 2. Create `.env`

```bash
cp .env.example .env
```

Then fill in:

| Variable | Value |
|---|---|
| `MONGO_INITDB_ROOT_PASSWORD`, `REDIS_PASSWORD` | Any URL-safe passwords (letters, digits, `- _ .`) |
| `SECRETS_MASTER_KEYS` | `echo "1:$(openssl rand -base64 32)"`, the key ring that encrypts the vault |
| `AUTH_SIGNING_KEYS` | `echo "k1:$(openssl rand -base64 32)"`, the key ring that signs sessions |
| `PLATFORM_ADMINS` | Your email: whoever signs in with it is a platform admin |
| `AUTH_COOKIE_SECURE` | `false` over plain HTTP (`http://localhost`, a LAN IP); leave it `true` behind HTTPS |

Both take a comma-separated list so keys can rotate (see [Key rotation](#key-rotation)).

### 3. Bring it up

```bash
docker compose up --build
```

Open <http://localhost> (or `http://<your LAN IP>` from another machine).

Mongo runs as a single-node replica set because the realtime service needs change streams. A `mongo_data` volume left over from an older standalone setup has to go first: `docker compose down -v`.

### 4. Create your account

1. **Register** at `/register`. There is no mail server in development: the verification link is written to the backend log.

   ```bash
   docker compose logs backend | grep "mail to"
   ```

2. Open the link and you are signed in. With your email in `PLATFORM_ADMINS` the account is a platform admin.
3. **Forgot password** works the same way (link in the log). It is also how an account created with Google gets a password; the profile page links to it.

### 5. Use it

- **Create a board** from the home page and drag cards from the dock onto the canvas.
- **Share** it from the board's *Share* button by email, as **editor** (works on cards) or **viewer** (only looks). The owner renames, deletes and shares; anyone may leave.
- **Organizations** (*Organizations* in the user menu): boards and groups created inside one reach every member without sharing. Org admins act as owners of its boards and members as viewers. The header switcher picks which organization's boards the sidebar shows and where new boards go.
- **Credentials** (*Board credentials*, editors only): API keys and OAuth2 accounts for the cards, shared with the board or kept for yourself.

### 6. Tear down

```bash
docker compose down        # stop, keep data
docker compose down -v     # also drop the Mongo volume
```

## Platform setup (admins)

Platform secrets are provisioned through the API with an admin's access token (there is no admin UI yet):

```bash
TOKEN=$(curl -s -X POST http://localhost/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"you@example.com","password":"your password"}' | jq -r .access_token)

# what the code expects, and which ones are configured
curl -s http://localhost/api/v1/system/secrets -H "Authorization: Bearer $TOKEN"

# the NASA key behind the "NASA picture of the day" card
curl -s -X PUT http://localhost/api/v1/system/secrets -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"NASA_API_KEY","kind":"api_key","value":"<your key from api.nasa.gov>"}'
```

`/api/v1/system/*` answers `403` to anyone who is not a platform admin. Secret values are write-only: no response ever carries one.

### Google

Two independent Google OAuth clients (type *Web application* in Google Cloud Console), each with its own redirect URI:

| Purpose | Redirect URI | Configured with |
|---|---|---|
| **Sign in with Google** | `<origin>/auth/google/callback` | `GOOGLE_SIGNIN_CLIENT_ID` / `GOOGLE_SIGNIN_CLIENT_SECRET` in `.env` (leave empty to hide the button) |
| **Connect a Google account** to the vault (Gmail, Calendar cards) | `<origin>/oauth2/callback` | `PUT /api/v1/system/oauth2/clients` `{"provider":"google","client_id":"…","client_secret":"…"}` as an admin |

`<origin>` is the address in the browser, e.g. `http://localhost`. Google accepts plain `http` only for `localhost`. Discord works like the second row with `"provider":"discord"`.

## Running pieces individually

```bash
docker compose up mongo redis      # bind them to 127.0.0.1 in compose to reach them from the host
```

| Service | Commands | Needs |
|---|---|---|
| backend | `cd backend && go run .` (listens on `:31126`) | `backend/.env.example`: `MONGODB_URI`, `REDIS_URL`, `SECRETS_MASTER_KEYS`, `AUTH_SIGNING_KEYS` |
| realtime | `cd realtime && pnpm install && pnpm build:dev && pnpm start:dev` (`:3000`) | `realtime/.env.example`: `MONGODB_URI` (replica set), `REDIS_URL`, `API_URL` |
| frontend | `cd frontend && pnpm install && pnpm dev` | `VITE_API_URL=http://localhost` in `frontend/.env` to use the Docker stack as the API |

Build the backend as a package (`go run .`, `go build .`), never `go run main.go`. From a host, add `&directConnection=true` to the Mongo URI.

## Tests

```bash
cd backend && go test ./...              # unit tests, no services
./run_backend_integration.sh             # or .ps1: real Mongo, Redis and a mock OAuth2 provider from compose
cd realtime && pnpm lint && pnpm test    # node --test, no services
cd frontend && pnpm check && pnpm test   # svelte-check, Vitest (Node, browser and Storybook projects)
```

The integration suite drives the HTTP API with the real wiring (`backend/app.go`): sign-up, login, refresh rotation and logout; rate limits; the refresh cookie's attributes and the same-origin check; board roles and organizations; the vault from provisioning to execution, with no response ever carrying a secret; Google sign-in against the mock provider. It uses a throwaway database and Redis db `1`.

## Key rotation

```bash
cd backend
go run . -rewrap-keys                 # after adding a higher version to SECRETS_MASTER_KEYS
go run . -rotate-key board:<uuid>     # new data key for one scope (<kind>:<owner>, or system)
go run . -rotate-all-keys
```

The highest `SECRETS_MASTER_KEYS` version encrypts and the older ones still decrypt until `-rewrap-keys` has run. `AUTH_SIGNING_KEYS` signs with its first key and verifies with all of them: put a new `kid` first and drop the old one once its tokens (15 minutes) have expired.

## API

Everything is under `/api/v1`. Only `/auth/*` is public; every other route needs `Authorization: Bearer <access token>` and answers `404` for anything the caller may not see.

| Area | Routes |
|---|---|
| Auth | `POST /auth/register`, `/auth/verify-email`, `/auth/login`, `/auth/refresh`, `/auth/logout`, `/auth/password/forgot`, `/auth/password/reset`; `GET /auth/google/start`, `POST /auth/google/callback` |
| Users | `GET`, `PATCH /users/:id` (yourself) |
| Boards | `GET /boards`, `POST /boards` (`{name, org?}`), `GET`, `DELETE /boards/:id`, `PATCH /boards/:id/name`, `GET /boards/:id/post-its` |
| Members | `GET /boards/:id/members`, `PUT /boards/:id/members` (`{email, role}`), `DELETE /boards/:id/members/:user` |
| Strands | `POST /boards/:id/strands`, `DELETE /boards/:id/strands/:strand` |
| Post-its | `POST /post-its`, `GET /post-its/:id` (execute), `DELETE /post-its/:id`, `GET`, `PATCH /post-its/:id/settings`, `PATCH /post-its/:id/position` |
| Organizations | `POST`, `GET /orgs`, `GET`, `PATCH`, `DELETE /orgs/:id`, `PUT /orgs/:id/members` (`{email, role}`), `DELETE /orgs/:id/members/:user` |
| Groups | `POST`, `GET /groups`, `GET`, `DELETE /groups/:id`, `POST`, `DELETE /groups/:id/members` |
| Vault | `{GET,PUT} …/secrets`, `DELETE …/secrets/:name`, `PUT …/secrets/:name/grants`, `PUT …/oauth2`, `GET …/oauth2/authorize`, `POST …/oauth2/connect` under `/boards/:id`, `/boards/:id/members/:user`, `/users/:id`, `/groups/:id`; `GET /boards/:id/secrets/usable` |
| OAuth2 | `GET /oauth2/providers`, `GET /oauth2/callback` |
| System (admins) | `GET`, `PUT /system/secrets`, `DELETE /system/secrets/:name`, `PUT /system/oauth2`, `PUT /system/oauth2/clients`, `GET /system/keys` |

Login, registration and password resets are rate limited per address and per account; adding people by email to boards and organizations, per user (`429 rate_limited`).

## Troubleshooting

- **Compose refuses to start**: a required variable is missing in `.env` (`SECRETS_MASTER_KEYS`, `AUTH_SIGNING_KEYS`, the passwords).
- **Signed out on every reload over HTTP**: `AUTH_COOKIE_SECURE=false` is missing, so the browser drops the refresh cookie on `http://`.
- **Mongo keeps restarting**: an old standalone `mongo_data` volume; `docker compose down -v`.
- **No live updates**: the realtime service needs a replica set and `API_URL`; `docker compose logs realtime`.
- **`429 rate_limited` while testing**: the counters expire on their own (a minute for logins, an hour for registration mails).
