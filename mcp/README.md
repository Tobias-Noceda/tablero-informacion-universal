# MCP server

An [xmcp](https://xmcp.dev) server (Streamable HTTP, `/mcp`) that lets an agent keep its memories on a user's boards ("PKGs", personal knowledge graphs):

| Tool | What it returns |
|---|---|
| `list_pkgs` | Every board the user can see (`id - organization - name`) |
| `create_pkg` / `delete_pkg` | Create or delete a board |
| `memories` | The post-its of one board (`id - (x; y) - title`) |
| `retrieve` | One post-it: its title, its data and the post-its connected to it |
| `consolidate` | Save a text memory (a `static_card`) at a position, connected to related ones |
| `reconsolidate` | Change a memory's title or text |
| `forget` | Delete a memory |

Every request needs `Authorization: Bearer <token>`. The server forwards it unchanged to the backend, which decides what the caller may see. Nothing is stored here. For how this could grow into OAuth for non-technical users, see `AUTH.md`.

## Tokens (developers)

1. Sign in to the app and open **Profile → Developer access tokens**.
2. Create a token (7, 15 or 30 days; 15 by default). It is shown **once**.
3. Add the server to your client, e.g. Claude Code:
   ```bash
   claude mcp add --transport http tiu http://localhost/mcp --header "Authorization: Bearer tiu_pat_…"
   ```
   or any client that takes JSON:
   ```json
   { "mcpServers": { "tiu": { "url": "http://localhost/mcp", "headers": { "Authorization": "Bearer tiu_pat_…" } } } }
   ```

Tokens reach only boards and cards (`middleware.RestrictAccessTokens`): the reads, plus creating, editing and deleting boards, cards and strands, with the user's board roles. Secrets, sharing, organizations, tokens and system routes answer `403 token_not_allowed`. Revoking one in the profile stops it on its next request.

## Running

- **Docker**: `docker compose up --build` from the repo root; the ALB routes `/mcp` here.
- **Alone**: `pnpm install`, then `pnpm dev` (or `pnpm build && pnpm start`). Listens on `0.0.0.0:3001/mcp`. `API_URL` is the backend's **origin**: the server adds `/api` (`http://localhost` for the Docker stack, `http://localhost:31126` for a backend run by hand).

To try it without an agent: `npx @modelcontextprotocol/inspector`, transport "Streamable HTTP", URL `http://localhost/mcp`, header `Authorization: Bearer tiu_pat_…`.
