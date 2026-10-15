# CLAUDE.md (realtime)

## Architecture

`realtime/` is a separate Node service, not part of the Go module.

- `mongo.ts` watches the `boards` collection with a change stream (`update` operations only, `fullDocument: updateLookup`). Needs a replica set.
- `socket.ts` runs socket.io at path `/ws`. A client connects with `?board=<id>&peer=<id>` and `auth: { token }` (the session's access token). The handshake middleware asks the backend `GET /boards/:id` with that token (`auth.ts`, `roleOn`): no `200` with a role, an outage included, refuses it with `unauthorized`. Node never verifies a JWT; it only reads `sub`/`exp` from a token the backend just accepted. A member joins the room `<board>`, receives `peers`, then `update` events `{ board, ts }` on every board change. A minute before the token expires the server emits `token_expiring`; the client answers `auth { token }` (same user), the role is checked again and the timer restarts. No answer within 30 s after expiry, or no role any more, and the server disconnects the socket, so a removed member is dropped at the next renewal at the latest.
- `redis.ts` keeps online peers in the hash `board:<id>:online`, peer → user (one entry per tab), expiring 20 min after the last join or renewal. It is the only presence writer.
- Neither a Redis nor a Mongo restart ends the process: the Redis client reconnects, and a change stream that gives up is reopened from the last change seen.
- Mouse cursors are peer-to-peer through PeerJS (`$modules/rtc.svelte.ts`), bootstrapped with the `peers` list from the socket.
