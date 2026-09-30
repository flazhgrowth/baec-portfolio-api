# Sudoku API

**This document is the source of truth for the Sudoku backend.** It describes the API exactly as it is
implemented and deployed, and it wins over `baec-sudoku-web/docs/API.md` and `openapi.yaml` wherever they differ.
Every example below was captured from the running server, not written by hand.

- Production: `https://api.baeclatant.com/ms/sudous/api/v1`
- Local: `http://localhost:12000/ms/sudous/api/v1`
- Code: `internal/implementations/{api,service,repository}` and the rules engine in
  `internal/entity/sudokugame/engine.go`. Game design notes: [`sudoku-design.md`](./sudoku-design.md).

The server is authoritative for the solution, move validation, scoring, turn timing and game state.
**The solution is never sent to the client.**

> **Read §3 first.** This API does not use one response shape: there are three, and which one you get depends on
> the endpoint and the status. A client that assumes a single shape will misread some responses.
>
> **Then read §11.** It lists bugs and gaps that exist *today* (for example: `register` does no validation, and the
> account token cannot be refreshed). The rest of this document describes the intended behavior; §11 describes where the
> running server falls short of it.

## Contents

1. [Conventions](#1-conventions)
2. [Endpoints at a glance](#2-endpoints-at-a-glance)
3. [Response shapes](#3-response-shapes)
4. [Authentication and identity](#4-authentication-and-identity)
5. [How a game works](#5-how-a-game-works)
6. [Endpoints: accounts](#6-endpoints-accounts)
7. [Endpoints: games](#7-endpoints-games)
8. [Live updates (SSE)](#8-live-updates-sse)
9. [Data models](#9-data-models)
10. [Error codes](#10-error-codes)
11. [Known issues and limitations](#11-known-issues-and-limitations)
12. [Differences from the original contract](#12-differences-from-the-original-contract)
13. [Client checklist](#13-client-checklist)

---

## 1. Conventions

- **Ids** are ULIDs (26 characters, e.g. `01M3SDZKPH9SPF9BE0SXWE88W2`): users and games. A seat's id (`Player.id`) is
  `"p1"` or `"p2"` and is only unique within one game.
- **Time.** Timestamps inside game and user objects are ISO-8601 UTC with microseconds and a `Z`
  (`2026-09-30T15:13:30.880313Z`). The response envelope's `servertime` is a different thing: Unix seconds.
  Every `Game` carries `server_time`; use it to correct clock skew before rendering a countdown.
- **Coordinates.** `row` and `col` are 0-based, `(0, 0)` is top-left, grids are indexed `[row][col]`, and `0` means an
  empty cell.
- **Content type.** Send `Content-Type: application/json` with JSON bodies. Responses are JSON, except `204`
  (empty) and the SSE stream (`text/event-stream`).
- **Paths.** `POST /games` also accepts a trailing slash (`/games/`). Use no trailing slash elsewhere.
- **Unknown route** → `404` with the plain-text body `404 page not found`. **Wrong method** on a known route →
  `405` with an empty body. Neither is JSON.
- **CORS.** All origins (`*`) are allowed, methods `GET, POST, PUT, DELETE, OPTIONS`. Allowed request headers: `Accept`,
  `Authorization`, `Content-Type`, `Content-Length`, `X-CSRF-Token`, `Accept-Encoding`, `X-Callback-Token`, `X-API-Key`,
  `X-API-Version`, `Idempotency-Key`, `X-Player-Token`. Preflights are cached for 300 s.
- **Examples** were captured from the server and trimmed in three ways: `puzzle` and `board` are shown as
  `"<9x9 grid>"` (the real value is a 9×9 array of integers), a `game` that just repeats the full `Game` shape is shown as
  `"<Game>"` (the complete object appears in the `POST /games`, `POST /games/join` and `GET /games/{id}` examples), and tokens
  are masked as `<account-jwt>` / `<player-token>`. Everything else is verbatim.
- **No rate limiting** exists yet.

## 2. Endpoints at a glance

| Method & path | Auth | Purpose | Success shape |
| --- | --- | --- | --- |
| `POST /auth/register` | none | Create an account and log in | envelope `201` |
| `POST /auth/login` | none | Log in | envelope `200` |
| `GET /auth/me` | account JWT | Who does this token belong to? | **bare** `200` |
| `POST /auth/logout` | none (token ignored) | Log out (a no-op) | empty `204` |
| `POST /games` | account JWT | Start a single / same-device game, or open an online lobby | envelope `201` |
| `POST /games/join` | account JWT | Join an online lobby by code | envelope `200` |
| `GET /games/{gameId}` | none | Current state of a game | envelope `200` |
| `GET /games/{gameId}/events?token=` | player token (query) | Live updates (SSE) | `text/event-stream` |
| `POST /games/{gameId}/moves` | `X-Player-Token` | Fill one cell | envelope `200` |
| `POST /games/{gameId}/turn/expire` | none | Report that the current turn ran out | envelope `200` |
| `POST /games/{gameId}/forfeit` | `X-Player-Token` | Leave a running online game (you lose) | envelope `200` |

## 3. Response shapes

There are three shapes. To decode any response: look at the HTTP status first, then the body.

### Shape A: the envelope (most endpoints)

Success, the payload is under `data`:

```json
{ "code": "success", "message": "Success", "data": { "...": "payload" }, "servertime": 1790781154 }
```

Error, `data` is `null` and `code` is the error code (see [§10](#10-error-codes)):

```json
{ "code": "GAME_NOT_FOUND", "message": "game not found", "data": null, "servertime": 1790781154 }
```

**Game 409 errors carry the current game** so you can resync without another call. It is under `data.game`:

```json
{ "code": "NOT_YOUR_TURN", "message": "It is not your turn", "data": { "game": { "...": "Game" } }, "servertime": 1790781154 }
```

The HTTP status is the real status (`201`, `404`, `409`, ...). `servertime` is Unix seconds.

### Shape B: bare

`GET /auth/me` returns the resource itself, with no envelope: `{ "id": "...", "username": "...", "created_at": "..." }`.
`POST /auth/logout` returns `204` and no body. Each SSE `update` message's data is also bare:
`{ "game": {...}, "events": [...] }`.

### Shape C: error object

Used **only** for account-token failures, which is `401` on `GET /auth/me`, `POST /games` and `POST /games/join`:

```json
{ "error": { "code": "INVALID_TOKEN", "message": "Missing or invalid account token" } }
```

(`GET /auth/me` also uses it for `500`: `{"error":{"code":"INTERNAL_ERROR","message":"internal server error"}}`.)

### Which endpoint uses which

| Endpoint | Success | Errors |
| --- | --- | --- |
| `POST /auth/register`, `POST /auth/login` | A | A |
| `GET /auth/me` | B | **C** |
| `POST /auth/logout` | `204` empty | (never fails) |
| `POST /games`, `POST /games/join` | A | `401` is **C**; everything else A |
| `GET /games/{id}` | A | A |
| `POST .../moves`, `.../turn/expire`, `.../forfeit` | A | A (`401 INVALID_TOKEN` here is the **player** token and is shape A) |
| `GET .../events` | SSE (bare messages) | A, sent as a normal JSON response *before* any stream starts |

Note that `INVALID_TOKEN` appears in two shapes: shape C for a bad **account** token, shape A for a bad **player** token.

### A decoder that handles all of it

```ts
async function decode(res: Response) {
  if (res.status === 204) return { ok: true, data: undefined };
  const body = await res.json().catch(() => undefined);       // 404/405 on unknown routes are not JSON
  if (body && typeof body === 'object' && 'error' in body)     // shape C
    return { ok: false, status: res.status, code: body.error.code, message: body.error.message };
  if (body && typeof body === 'object' && 'servertime' in body) {  // shape A
    return res.ok
      ? { ok: true, data: body.data }
      : { ok: false, status: res.status, code: body.code, message: body.message, game: body.data?.game };
  }
  return res.ok ? { ok: true, data: body } : { ok: false, status: res.status, code: 'UNKNOWN', message: res.statusText }; // shape B
}
```

## 4. Authentication and identity

Two separate tokens answer two different questions.

| | **Account token** | **Player token** |
| --- | --- | --- |
| Answers | *Who is calling?* | *Which seat, in this one game?* |
| Kind | JWT (HS256), stateless | Opaque random string (43 chars), stored hashed |
| Comes from | `token` in `register` / `login` | `credentials[].token` in the response to `POST /games` and `POST /games/join` |
| Sent as | `Authorization: Bearer <token>` | `X-Player-Token: <token>` header, or `?token=` on the event stream |
| Needed by | `GET /auth/me`, `POST /games`, `POST /games/join` | `moves`, `forfeit`, `events` |
| Lifetime | **30 minutes** | until the game is deleted |

**The account token expires after 30 minutes and there is no way to refresh it.** `register` and `login` also return a
`refresh_token` (7-day JWT) but **no endpoint accepts it**. When the account token expires, `GET /auth/me` /
`POST /games` / `POST /games/join` answer `401 INVALID_TOKEN` (shape C) and the user has to log in again. Already-running
games are unaffected: they use player tokens.

**Player tokens are shown exactly once**, in the `credentials` of the create/join response. The server stores only a
hash, `GET /games/{id}` never returns them, and they cannot be recovered. **The client must persist them** (for
example in `localStorage`, keyed by game id) or a page reload loses the seat.

**Who gets which seat.**

- The caller always takes the first seat (`p1`), named after their account `username`. `player_names[0]` is ignored.
- Online: the host is `p1`; the guest who joins is `p2`, named and identified by *their own* account.
- Same-device `versus`: one call returns **two** credentials (`p1`, `p2`). Only `p1` belongs to the account
  (`user_id` set). `p2` is a guest with `user_id: null` and the name from `player_names[1]`: trimmed, cut to 20
  characters, `"Player 2"` if missing or blank. Extra `player_names` entries are ignored.
- Single: one credential.

Treat both kinds of token as secrets; anyone holding one can act as that account or seat.

**Logout does nothing on the server.** Tokens are stateless, so the client logs out by discarding its token; the token
keeps verifying until it expires.

## 5. How a game works

### Game kinds

| `mode` | `online` | Meaning |
| --- | --- | --- |
| `single` | `false` | One player. No turns, no timer, no points. `mistakes` are counted. |
| `versus` | `false` | Two players sharing one device, turn based. Both seats exist from the start. |
| `versus` | `true` | Two players on their own devices. Starts as a lobby (`status: "waiting"`) that a second player joins with a code. |

`online: true` is only valid with `mode: "versus"`.

### Online lifecycle

1. The host calls `POST /games` with `online: true`. The response is a `waiting` game with a 6-character `join_code`
   (alphabet `ABCDEFGHJKLMNPQRSTUVWXYZ23456789`), one player, and no turn.
2. The host opens the event stream and waits.
3. The guest calls `POST /games/join` with the code and **their own** account token. The game becomes `in_progress`,
   `join_code` becomes `null`, `started_at` resets to now, and `p1` gets the first turn. The host hears about it on
   the stream as a `player_joined` event.
4. Both play. The game ends when the board is solved, or when someone forfeits (`end_reason` says which).

A lobby nobody joins stays joinable for **30 minutes**; after that, `join`, `GET` and the stream answer `404`. This is
enforced when someone asks, not by a background job.

### Versus rules

All constants are on `Game.rules`; read them from there instead of hardcoding.

1. A turn is one attempt to fill one cell. `p1` moves first.
2. A turn lasts `turn_limit_ms` (10 000 ms). At or after `current_turn.deadline_at` it is over: no fault, no points, and
   the turn passes to the other player. The next turn starts at the moment the expiry is *processed*, not at the old
   deadline.
3. A correct fill scores `max(min_points, max_points - floor(elapsed_ms / 1000))` where
   `elapsed_ms = now - current_turn.started_at`: 10 points in the first second, 9 in the second, ... 1 point from the
   tenth second on. A correct fill ends the turn.
4. A wrong fill scores nothing, is **not** written to the board, adds one fault (`faults` and `mistakes` both go up),
   and ends the turn.
5. On reaching `fault_limit` (3) faults, the player owes `skip_turns_on_fault_limit` (2) turns
   (`skip_turns_remaining = 2`). Each time the turn would come to them it is skipped instead
   (`skip_turns_remaining` goes down, a `turn_skipped` event is sent) and stays with the opponent. When the last
   skipped turn is consumed, their `faults` resets to 0. Net effect: after the third fault the opponent plays three
   turns in a row.
6. The game completes when every cell matches the solution. `winner_id` is the player with the higher score, or `null`
   on a draw. In `single` mode `winner_id` is always `null`.

A `single` game has no turns at all: `current_turn` is `null`, `points` are always `0`, `elapsed_ms` is `null`, and a
wrong fill only increments `mistakes` and `faults` (there is no fault limit).

### Turn expiry: the client must drive it

**There is no server-side timer.** The server applies a timeout only when something asks it to: a `GET /games/{id}`, a
`moves` or `turn/expire` call, or a new event-stream connection notices an overdue turn and expires it first.
(`forfeit` does not bother: the game ends either way.) Nothing expires a turn in the background.

So **when the client's own countdown (corrected with `server_time`) reaches zero, it must call
`POST /games/{id}/turn/expire`**, in every versus game (same-device and online). It is safe to call from both devices
at once: exactly one call applies the expiry and the rest get `409 TURN_NOT_EXPIRED`, which is harmless (the state
they carry is the new one). If neither device calls it, the turn stays overdue until someone makes another request.

A move that arrives after the deadline is refused with `409 TURN_EXPIRED`, but **the expiry is applied and saved
first**, so the error's `data.game` already shows the next turn.

### Presence and disconnects: not implemented

`Player.connected` is always `true` and `Player.forfeit_at` is always `null`. There is no `player_connection` event and
**no disconnect forfeit**: a player who closes their tab or loses connection is never marked gone, and the game waits
for them (their turns keep expiring normally). The only way out of a running online game is the explicit
`POST /games/{id}/forfeit`. See [§11](#11-known-issues-and-limitations).

### Versions

`Game.version` starts at `1` and goes up by exactly one on every state change (a join, an accepted move, a turn expiry,
a forfeit). Every `update` carries the full game, so a client can just drop any state whose `version` is lower than the
one it holds. A change reaches the player who caused it twice (in the HTTP response and on the stream); announce its
`events` only once.

### Concurrency

Writes are safe under races: two requests can never both apply. The loser simply gets the normal error for the state it
now finds (for example `NOT_YOUR_TURN`, `GAME_COMPLETED`, `TURN_NOT_EXPIRED`, `GAME_FULL`). Requests that keep losing
the race are retried internally; only after five losses in a row does a request fail, with a `500`.

---

## 6. Endpoints: accounts

### `POST /auth/register`: create an account

No verification of any kind; the account exists as soon as the call returns, and you are logged in.

Body: `{ "username": string, "password": string }`.

**The username is lowercased before it is stored and returned** (`"Alex"` becomes `"alex"`). Show the `username` from the
response, not what the user typed. Because of this, usernames are unique regardless of case: registering `Alex` and then
`alex` (or `ALEX`) conflicts.

**The server does not validate either field today** (see [§11](#11-known-issues-and-limitations)); the client should
enforce the intended rules: username 3–20 letters, digits or underscores; password 6–72 characters.

```http
POST /auth/register
Content-Type: application/json

{"username": "alice", "password": "hunter2222"}
```

Response `201`

```json
{
  "code": "success",
  "message": "Success",
  "data": {
    "id": "01M3SE1DE9CQXG9SCWTJP83YZG",
    "username": "alice",
    "token": "<account-jwt>",
    "refresh_token": "<account-jwt>"
  },
  "servertime": 1790781208
}
```

| Status | `code` | When |
| --- | --- | --- |
| `201` | `success` | Created. `data`: `{ id, username, token, refresh_token }`. |
| `409` | `conflict` | That username is taken (compared case-insensitively). `message` is empty. |
| `400` | `bad_request` | Body is not valid JSON (`invalid request body`). |

```http
POST /auth/register
Content-Type: application/json

{"username": "alice", "password": "hunter2222"}
```

Response `409`

```json
{
  "code": "conflict",
  "message": "",
  "data": null,
  "servertime": 1790781208
}
```

### `POST /auth/login`

Body: `{ "username": string, "password": string }`. The username is case-insensitive (`Alex`, `ALEX` and `alex` all find the
account `alex`). Returns the same `data` as `register`, and issues a fresh token; older tokens for the account keep working
until they expire.

```http
POST /auth/login
Content-Type: application/json

{"username": "alice", "password": "hunter2222"}
```

Response `200`

```json
{
  "code": "success",
  "message": "Success",
  "data": {
    "id": "01M3SE1DE9CQXG9SCWTJP83YZG",
    "username": "alice",
    "token": "<account-jwt>",
    "refresh_token": "<account-jwt>"
  },
  "servertime": 1790781208
}
```

| Status | `code` | When |
| --- | --- | --- |
| `200` | `success` | Logged in. |
| `401` | `invalid_credentials` | Unknown username **or** wrong password (deliberately the same, so usernames cannot be probed). The `message` differs (`invalid credentials` / `unauthorized`); key off the `code`. |
| `400` | `bad_request` | Body is not valid JSON. |

```http
POST /auth/login
Content-Type: application/json

{"username": "alice", "password": "wrong"}
```

Response `401`

```json
{
  "code": "invalid_credentials",
  "message": "invalid credentials",
  "data": null,
  "servertime": 1790781209
}
```

### `GET /auth/me`: resolve a token

Header: `Authorization: Bearer <account token>`. Returns the **bare** `User` (shape B), used to restore a session after a
reload and to check that a stored token still works.

```http
GET /auth/me
Authorization: Bearer <account-jwt>
```

Response `200`

```json
{
  "id": "01M3SE1DE9CQXG9SCWTJP83YZG",
  "username": "alice",
  "created_at": "2026-09-30T15:13:28.009991Z"
}
```

| Status | Body | When |
| --- | --- | --- |
| `200` | `{ id, username, created_at }` | Token is valid and the account exists. |
| `401` | `{ "error": { "code": "INVALID_TOKEN", ... } }` (shape C) | Missing, malformed, forged or expired token, **or** a valid token whose account no longer exists. |
| `500` | `{ "error": { "code": "INTERNAL_ERROR", ... } }` (shape C) | Server or database failure (does not mean the token is bad; do not log the user out). |

```http
GET /auth/me
```

Response `401`

```json
{
  "error": {
    "code": "INVALID_TOKEN",
    "message": "Missing or invalid account token"
  }
}
```

### `POST /auth/logout`

Always `204` with an empty body, whatever token (or none) you send, including an expired or invalid one. It revokes
nothing ([§4](#4-authentication-and-identity)); the client discards its token.

```http
POST /auth/logout
Authorization: Bearer <account-jwt>
```

Response `204` (empty body)

---

## 7. Endpoints: games

Game errors use shape A. The `409` errors below all carry the current game under `data.game`.

### `POST /games`: start a game

Header: `Authorization: Bearer <account token>`.

| Field | Type | Notes |
| --- | --- | --- |
| `mode` | `"single"` \| `"versus"` | required |
| `difficulty` | `"easy"` \| `"medium"` \| `"hard"` | required. Puzzles have exactly one solution: 40, 32 and 26 clues. |
| `online` | boolean | optional, default `false`; only valid with `mode: "versus"` |
| `player_names` | string[] | optional; only `player_names[1]` is used (see [§4](#4-authentication-and-identity)) |

Returns a **Session**: the game plus one credential per seat the caller controls.

- `single`: `in_progress`, no turn, one credential.
- same-device `versus`: `in_progress`, `p1` already has the turn (10 s), **two** credentials.
- online `versus`: `waiting`, a `join_code`, no turn, one credential (the host's).

Generating the puzzle happens inside the request: the median is about 80 ms for `hard` and a few ms for `easy`/`medium`, but a slow `hard` puzzle can take 1-3 seconds, so show a spinner and allow a generous timeout.

```http
POST /games
Authorization: Bearer <account-jwt>
Content-Type: application/json

{"mode": "versus", "difficulty": "medium", "player_names": ["ignored", "  Guest  "]}
```

Response `201`

```json
{
  "code": "success",
  "message": "Success",
  "data": {
    "game": {
      "id": "01M3SE1G80N7Y1QAWR6NKYP3JA",
      "mode": "versus",
      "online": false,
      "difficulty": "medium",
      "status": "in_progress",
      "join_code": null,
      "puzzle": "<9x9 grid>",
      "board": "<9x9 grid>",
      "players": [
        {
          "id": "p1",
          "name": "alice",
          "user_id": "01M3SE1DE9CQXG9SCWTJP83YZG",
          "score": 0,
          "faults": 0,
          "mistakes": 0,
          "skip_turns_remaining": 0,
          "connected": true,
          "forfeit_at": null
        },
        {
          "id": "p2",
          "name": "Guest",
          "user_id": null,
          "score": 0,
          "faults": 0,
          "mistakes": 0,
          "skip_turns_remaining": 0,
          "connected": true,
          "forfeit_at": null
        }
      ],
      "current_turn": {
        "player_id": "p1",
        "started_at": "2026-09-30T15:13:30.880313Z",
        "deadline_at": "2026-09-30T15:13:40.880313Z"
      },
      "rules": {
        "turn_limit_ms": 10000,
        "max_points": 10,
        "min_points": 1,
        "fault_limit": 3,
        "skip_turns_on_fault_limit": 2,
        "disconnect_forfeit_ms": 60000
      },
      "started_at": "2026-09-30T15:13:30.880313Z",
      "completed_at": null,
      "end_reason": null,
      "winner_id": null,
      "version": 1,
      "server_time": "2026-09-30T15:13:30.880313Z"
    },
    "credentials": [
      {
        "player_id": "p1",
        "token": "<player-token>"
      },
      {
        "player_id": "p2",
        "token": "<player-token>"
      }
    ]
  },
  "servertime": 1790781210
}
```

The same call for an online lobby (the game is shown in full only once; grids elided):

```http
POST /games
Authorization: Bearer <account-jwt>
Content-Type: application/json

{"mode": "versus", "difficulty": "hard", "online": true}
```

Response `201`

```json
{
  "code": "success",
  "message": "Success",
  "data": {
    "game": {
      "id": "01M3SE1GA185KDKDVVVX830JV5",
      "mode": "versus",
      "online": true,
      "difficulty": "hard",
      "status": "waiting",
      "join_code": "GWH8GX",
      "puzzle": "<9x9 grid>",
      "board": "<9x9 grid>",
      "players": [
        {
          "id": "p1",
          "name": "alice",
          "user_id": "01M3SE1DE9CQXG9SCWTJP83YZG",
          "score": 0,
          "faults": 0,
          "mistakes": 0,
          "skip_turns_remaining": 0,
          "connected": true,
          "forfeit_at": null
        }
      ],
      "current_turn": null,
      "rules": {
        "turn_limit_ms": 10000,
        "max_points": 10,
        "min_points": 1,
        "fault_limit": 3,
        "skip_turns_on_fault_limit": 2,
        "disconnect_forfeit_ms": 60000
      },
      "started_at": "2026-09-30T15:13:30.945714Z",
      "completed_at": null,
      "end_reason": null,
      "winner_id": null,
      "version": 1,
      "server_time": "2026-09-30T15:13:30.945714Z"
    },
    "credentials": [
      {
        "player_id": "p1",
        "token": "<player-token>"
      }
    ]
  },
  "servertime": 1790781210
}
```

| Status | `code` | When |
| --- | --- | --- |
| `201` | `success` | Created. |
| `401` | `INVALID_TOKEN` (**shape C**) | Missing/invalid/expired account token. |
| `422` | `VALIDATION_ERROR` | `mode must be 'single' or 'versus'` · `difficulty must be 'easy', 'medium' or 'hard'` · `online is only valid with mode 'versus'` · `request body is not valid` (malformed JSON). |

```http
POST /games
Authorization: Bearer <account-jwt>
Content-Type: application/json

{"mode": "coop", "difficulty": "easy"}
```

Response `422`

```json
{
  "code": "VALIDATION_ERROR",
  "message": "mode must be 'single' or 'versus'",
  "data": null,
  "servertime": 1790781210
}
```

### `POST /games/join`: join an online lobby

Header: `Authorization: Bearer <account token>`. Body: `{ "code": "ABC234" }`. The code is case-insensitive and
surrounding whitespace is ignored.

Seats the caller as `p2`, using **their** account's username and id (nothing from the request), starts the game and
`p1`'s first turn, and publishes `player_joined` to the host's stream. The response is a Session with **only `p2`'s
credential**.

```http
POST /games/join
Authorization: Bearer <account-jwt>
Content-Type: application/json

{"code": "gwh8gx"}
```

Response `200`

```json
{
  "code": "success",
  "message": "Success",
  "data": {
    "game": {
      "id": "01M3SE1GA185KDKDVVVX830JV5",
      "mode": "versus",
      "online": true,
      "difficulty": "hard",
      "status": "in_progress",
      "join_code": null,
      "puzzle": "<9x9 grid>",
      "board": "<9x9 grid>",
      "players": [
        {
          "id": "p1",
          "name": "alice",
          "user_id": "01M3SE1DE9CQXG9SCWTJP83YZG",
          "score": 0,
          "faults": 0,
          "mistakes": 0,
          "skip_turns_remaining": 0,
          "connected": true,
          "forfeit_at": null
        },
        {
          "id": "p2",
          "name": "bob",
          "user_id": "01M3SE1G7G8HSY6Q97QMYBH2RJ",
          "score": 0,
          "faults": 0,
          "mistakes": 0,
          "skip_turns_remaining": 0,
          "connected": true,
          "forfeit_at": null
        }
      ],
      "current_turn": {
        "player_id": "p1",
        "started_at": "2026-09-30T15:13:31.967058Z",
        "deadline_at": "2026-09-30T15:13:41.967058Z"
      },
      "rules": {
        "turn_limit_ms": 10000,
        "max_points": 10,
        "min_points": 1,
        "fault_limit": 3,
        "skip_turns_on_fault_limit": 2,
        "disconnect_forfeit_ms": 60000
      },
      "started_at": "2026-09-30T15:13:31.967058Z",
      "completed_at": null,
      "end_reason": null,
      "winner_id": null,
      "version": 2,
      "server_time": "2026-09-30T15:13:31.967058Z"
    },
    "credentials": [
      {
        "player_id": "p2",
        "token": "<player-token>"
      }
    ]
  },
  "servertime": 1790781211
}
```

| Status | `code` | When |
| --- | --- | --- |
| `200` | `success` | Joined; the game is now `in_progress`. |
| `401` | `INVALID_TOKEN` (**shape C**) | Bad account token. |
| `404` | `JOIN_CODE_NOT_FOUND` | No open lobby with that code. Also returned for a lobby that already started and one older than 30 minutes, so a stale code reveals nothing. |
| `409` | `GAME_FULL` | Another guest took the seat in the same instant (a race). `message` is just `conflict`. No `data.game`. |
| `422` | `VALIDATION_ERROR` | `code must be 6 characters` · `request body is not valid`. |

```http
POST /games/join
Authorization: Bearer <account-jwt>
Content-Type: application/json

{"code": "ZZZZZZ"}
```

Response `404`

```json
{
  "code": "JOIN_CODE_NOT_FOUND",
  "message": "no open lobby with that code",
  "data": null,
  "servertime": 1790781211
}
```

```http
POST /games/join
Authorization: Bearer <account-jwt>
Content-Type: application/json

{"code": "J52QCA"}
```

Response `409`

```json
{
  "code": "GAME_FULL",
  "message": "conflict",
  "data": null,
  "servertime": 1790781212
}
```

Nothing stops the host from joining their own lobby with their own account.

### `GET /games/{gameId}`: current state

No token. Applies anything already due (an overdue turn) before answering, then returns the `Game`. Use it to resume a
game after a reload (you still need the player token you saved). It exposes the board, player names and `user_id`s, and a
waiting lobby's `join_code`, to anyone who knows the id; it never exposes the solution or any token.

```http
GET /games/01M3SE1G80N7Y1QAWR6NKYP3JA
```

Response `200`

```json
{
  "code": "success",
  "message": "Success",
  "data": {
    "id": "01M3SE1G80N7Y1QAWR6NKYP3JA",
    "mode": "versus",
    "online": false,
    "difficulty": "medium",
    "status": "in_progress",
    "join_code": null,
    "puzzle": "<9x9 grid>",
    "board": "<9x9 grid>",
    "players": [
      {
        "id": "p1",
        "name": "alice",
        "user_id": "01M3SE1DE9CQXG9SCWTJP83YZG",
        "score": 37,
        "faults": 0,
        "mistakes": 0,
        "skip_turns_remaining": 0,
        "connected": true,
        "forfeit_at": null
      },
      {
        "id": "p2",
        "name": "Guest",
        "user_id": null,
        "score": 0,
        "faults": 0,
        "mistakes": 3,
        "skip_turns_remaining": 0,
        "connected": true,
        "forfeit_at": null
      }
    ],
    "current_turn": {
      "player_id": "p2",
      "started_at": "2026-09-30T15:13:34.412331Z",
      "deadline_at": "2026-09-30T15:13:44.412331Z"
    },
    "rules": {
      "turn_limit_ms": 10000,
      "max_points": 10,
      "min_points": 1,
      "fault_limit": 3,
      "skip_turns_on_fault_limit": 2,
      "disconnect_forfeit_ms": 60000
    },
    "started_at": "2026-09-30T15:13:30.880313Z",
    "completed_at": null,
    "end_reason": null,
    "winner_id": null,
    "version": 10,
    "server_time": "2026-09-30T15:13:34.420372Z"
  },
  "servertime": 1790781214
}
```

| Status | `code` | When |
| --- | --- | --- |
| `200` | `success` | `data` is the `Game`. |
| `404` | `GAME_NOT_FOUND` | No such game, or an online lobby nobody joined within 30 minutes. |

### `POST /games/{gameId}/moves`: fill a cell

Header: `X-Player-Token: <token>`. Body: `{ "row": 0-8, "col": 0-8, "value": 1-9 }`, all JSON numbers with whole values
(`"4"` or `4.5` are rejected).

**A wrong number is a normal `200` with `"result": "incorrect"`, not an error.** Errors are only for requests that cannot
be processed at all.

```http
POST /games/01M3SE1GA185KDKDVVVX830JV5/moves
X-Player-Token: <player-token>
Content-Type: application/json

{"row": 0, "col": 0, "value": 1}
```

Response `200`

```json
{
  "code": "success",
  "message": "Success",
  "data": {
    "result": "correct",
    "points": 10,
    "elapsed_ms": 216,
    "game": "<Game>",
    "events": [
      {
        "type": "move",
        "player_id": "p1",
        "row": 0,
        "col": 0,
        "value": 1,
        "result": "correct",
        "points": 10
      }
    ]
  },
  "servertime": 1790781212
}
```

```http
POST /games/01M3SE1GA185KDKDVVVX830JV5/moves
X-Player-Token: <player-token>
Content-Type: application/json

{"row": 0, "col": 3, "value": 1}
```

Response `200`

```json
{
  "code": "success",
  "message": "Success",
  "data": {
    "result": "incorrect",
    "points": 0,
    "elapsed_ms": 59,
    "game": "<Game>",
    "events": [
      {
        "type": "move",
        "player_id": "p2",
        "row": 0,
        "col": 3,
        "value": 1,
        "result": "incorrect",
        "points": 0
      }
    ]
  },
  "servertime": 1790781212
}
```

**Fault limit and skipped turns:** the events tell you what happened after the move.

```http
POST /games/01M3SE1G80N7Y1QAWR6NKYP3JA/moves
X-Player-Token: <player-token>
Content-Type: application/json

{"row": 0, "col": 6, "value": 6}
```

Response `200`

```json
{
  "code": "success",
  "message": "Success",
  "data": {
    "result": "incorrect",
    "points": 0,
    "elapsed_ms": 59,
    "game": "<Game>",
    "events": [
      {
        "type": "move",
        "player_id": "p2",
        "row": 0,
        "col": 6,
        "value": 6,
        "result": "incorrect",
        "points": 0
      },
      {
        "type": "fault_limit_reached",
        "player_id": "p2",
        "skip_turns": 2
      }
    ]
  },
  "servertime": 1790781214
}
```

```http
POST /games/01M3SE1G80N7Y1QAWR6NKYP3JA/moves
X-Player-Token: <player-token>
Content-Type: application/json

{"row": 0, "col": 6, "value": 5}
```

Response `200`

```json
{
  "code": "success",
  "message": "Success",
  "data": {
    "result": "correct",
    "points": 10,
    "elapsed_ms": 59,
    "game": "<Game>",
    "events": [
      {
        "type": "move",
        "player_id": "p1",
        "row": 0,
        "col": 6,
        "value": 5,
        "result": "correct",
        "points": 10
      },
      {
        "type": "turn_skipped",
        "player_id": "p2"
      }
    ]
  },
  "servertime": 1790781214
}
```

**Completing the game:** the last events are `game_completed`; the game has `status: "completed"`, `end_reason: "solved"`,
`current_turn: null`, and `winner_id` (or `null` on a draw).

```http
POST /games/01M3SE1G80N7Y1QAWR6NKYP3JA/moves
X-Player-Token: <player-token>
Content-Type: application/json

{"row": 8, "col": 6, "value": 4}
```

Response `200`

```json
{
  "code": "success",
  "message": "Success",
  "data": {
    "result": "correct",
    "points": 10,
    "elapsed_ms": 90,
    "game": "<Game>",
    "events": [
      {
        "type": "move",
        "player_id": "p2",
        "row": 8,
        "col": 6,
        "value": 4,
        "result": "correct",
        "points": 10
      },
      {
        "type": "game_completed"
      }
    ]
  },
  "servertime": 1790781218
}
```

In `single` mode `points` is `0` and `elapsed_ms` is `null`:

```http
POST /games/01M3SE1G7QT8M25ARDHYFVVGTQ/moves
X-Player-Token: <player-token>
Content-Type: application/json

{"row": 0, "col": 0, "value": 7}
```

Response `200`

```json
{
  "code": "success",
  "message": "Success",
  "data": {
    "result": "correct",
    "points": 0,
    "elapsed_ms": null,
    "game": "<Game>",
    "events": [
      {
        "type": "move",
        "player_id": "p1",
        "row": 0,
        "col": 0,
        "value": 7,
        "result": "correct",
        "points": 0
      }
    ]
  },
  "servertime": 1790781218
}
```

**Checks happen in this order**, and the first failure wins: game exists (`404`) → token belongs to a seat in this game
(`401`) → input (`422`) → game state (`409`s below).

| Status | `code` | When |
| --- | --- | --- |
| `200` | `success` | Move accepted (`result` is `correct` or `incorrect`). |
| `404` | `GAME_NOT_FOUND` | Unknown game. |
| `401` | `INVALID_TOKEN` | Missing, unknown, or another game's player token. **Shape A**, not shape C. |
| `422` | `VALIDATION_ERROR` | `row and col must be integers between 0 and 8` · `value must be an integer between 1 and 9` (also for a missing field) · `request body is not valid`. |
| `409` | `GAME_NOT_STARTED` | An online lobby still waiting for a second player. |
| `409` | `GAME_COMPLETED` | The game is over (including by forfeit). |
| `409` | `TURN_EXPIRED` | The deadline passed before the move arrived. **The expiry is already applied and saved**; `data.game.current_turn` is the next turn. The move itself was not applied. |
| `409` | `NOT_YOUR_TURN` | The token's seat is not `current_turn.player_id`. |
| `409` | `CELL_NOT_EMPTY` | The cell already holds a clue or a previous correct fill. |

```http
POST /games/01M3SE1G80N7Y1QAWR6NKYP3JA/moves
X-Player-Token: <player-token>
Content-Type: application/json

{"row": 0, "col": 7, "value": 1}
```

Response `409`

```json
{
  "code": "TURN_EXPIRED",
  "message": "Your turn has expired",
  "data": {
    "game": "<Game>"
  },
  "servertime": 1790781214
}
```

```http
POST /games/01M3SE1GA185KDKDVVVX830JV5/moves
X-Player-Token: <player-token>
Content-Type: application/json

{"row": 0, "col": 0, "value": 1}
```

Response `409`

```json
{
  "code": "NOT_YOUR_TURN",
  "message": "It is not your turn",
  "data": {
    "game": "<Game>"
  },
  "servertime": 1790781212
}
```

### `POST /games/{gameId}/turn/expire`: report a timeout

No token. **This is how a turn times out in practice** ([§5](#turn-expiry-the-client-must-drive-it)): call it when your
countdown reaches zero. It acts only once `now >= deadline_at`, so calling early or twice cannot cut a turn short.

Returns a `GameUpdate`: the new game plus the events that happened (`turn_expired`, and a `turn_skipped` if the
other player owed a skip).

```http
POST /games/01M3SE1G80N7Y1QAWR6NKYP3JA/turn/expire
```

Response `200`

```json
{
  "code": "success",
  "message": "Success",
  "data": {
    "game": "<Game>",
    "events": [
      {
        "type": "turn_expired",
        "player_id": "p1"
      }
    ]
  },
  "servertime": 1790781214
}
```

| Status | `code` | When |
| --- | --- | --- |
| `200` | `success` | The turn expired; `data` is `{ game, events }`. |
| `404` | `GAME_NOT_FOUND` | Unknown game. |
| `409` | `TURN_NOT_EXPIRED` | The deadline has not passed yet, or someone (another device, another call) already expired it. `data.game` is the current state. |
| `409` | `NO_ACTIVE_TURN` | `single` game. |
| `409` | `GAME_NOT_STARTED` | Online lobby still waiting. |
| `409` | `GAME_COMPLETED` | The game is over. |

```http
POST /games/01M3SE1G80N7Y1QAWR6NKYP3JA/turn/expire
```

Response `409`

```json
{
  "code": "TURN_NOT_EXPIRED",
  "message": "The current turn has not expired yet",
  "data": {
    "game": "<Game>"
  },
  "servertime": 1790781214
}
```

### `POST /games/{gameId}/forfeit`: leave a running online game

Header: `X-Player-Token: <token>`. No body. Online games only, and only while `in_progress`. The caller loses and the
opponent wins with `end_reason: "forfeit"`. It does not have to be the caller's turn, and the score does not matter.
Leaving a `waiting` lobby needs no call; it simply expires unjoined.

Returns a `GameUpdate` with `player_forfeited` (the caller) then `game_completed`.

```http
POST /games/01M3SE1GA185KDKDVVVX830JV5/forfeit
X-Player-Token: <player-token>
```

Response `200`

```json
{
  "code": "success",
  "message": "Success",
  "data": {
    "game": "<Game>",
    "events": [
      {
        "type": "player_forfeited",
        "player_id": "p2"
      },
      {
        "type": "game_completed"
      }
    ]
  },
  "servertime": 1790781218
}
```

| Status | `code` | When |
| --- | --- | --- |
| `200` | `success` | Forfeited; `data` is `{ game, events }`. |
| `404` | `GAME_NOT_FOUND` | Unknown game. |
| `401` | `INVALID_TOKEN` | Missing/unknown/other game's player token (checked before anything else). |
| `409` | `NOT_ONLINE` | Single or same-device game. |
| `409` | `GAME_NOT_STARTED` | Online lobby still waiting. |
| `409` | `GAME_COMPLETED` | Already over. |

```http
POST /games/01M3SE1G80N7Y1QAWR6NKYP3JA/forfeit
X-Player-Token: <player-token>
```

Response `409`

```json
{
  "code": "NOT_ONLINE",
  "message": "Only online games can be forfeited",
  "data": {
    "game": "<Game>"
  },
  "servertime": 1790781214
}
```

---

## 8. Live updates (SSE)

```
GET /games/{gameId}/events?token=<player token>
```

Streams `text/event-stream`. The token goes in the query string because browsers cannot set headers on `EventSource`.
This is the **player** token ([§4](#4-authentication-and-identity)), not the account token.

- On connect the server immediately sends one `update` with the full current state and `events: []`. A reconnect
  therefore fully resyncs the client; nothing is replayed, and `Last-Event-ID` is not used.
- After every state change it sends another `update` with the new `game` and the `events` describing what just happened.
  Connecting also applies an overdue turn first, which the other device is told about.
- Every ~10 seconds it sends a `: ping` comment line to stop proxies closing an idle stream.
- Only `event: update` messages exist. The `data:` is bare JSON (shape B): `{ "game": Game, "events": GameEvent[] }`.
- Connections are not limited per game. The stream stays open after the game completes.
- A client that falls too far behind (32 unsent updates) is disconnected; its `EventSource` reconnects and is resynced
  by the initial `update`.
- **Errors arrive as a normal JSON response before any stream starts** (shape A): `401 INVALID_TOKEN` (missing or wrong
  token, or a token from another game) and `404 GAME_NOT_FOUND`. `EventSource` treats any non-`200` as fatal and closes
  without retrying, so handle this in `onerror` by checking `readyState === EventSource.CLOSED`.
- The server runs as a single instance, so the stream always reflects every change.

What goes over the wire (this is the host of an online game, from lobby to forfeit):

```
# host connects to the lobby  ->  full state, events: []
event: update
data: {"game": {"id": "01M3SE1GA185KDKDVVVX830JV5", "status": "waiting", "version": 1, "current_turn": null, "...": "the full Game"}, "events": []}

# the guest joins
event: update
data: {"game": {"id": "01M3SE1GA185KDKDVVVX830JV5", "status": "in_progress", "version": 2, "current_turn": {"player_id": "p1", "started_at": "2026-09-30T15:13:31.967058Z", "deadline_at": "2026-09-30T15:13:41.967058Z"}, "...": "the full Game"}, "events": [{"type": "player_joined", "player_id": "p2"}]}

# host plays a correct move
event: update
data: {"game": {"id": "01M3SE1GA185KDKDVVVX830JV5", "status": "in_progress", "version": 3, "current_turn": {"player_id": "p2", "started_at": "2026-09-30T15:13:32.184002Z", "deadline_at": "2026-09-30T15:13:42.184002Z"}, "...": "the full Game"}, "events": [{"type": "move", "player_id": "p1", "row": 0, "col": 0, "value": 1, "result": "correct", "points": 10}]}

# ...every ~10 s while idle, a comment line keeps the connection open:
: ping

# guest plays a wrong move
event: update
data: {"game": {"id": "01M3SE1GA185KDKDVVVX830JV5", "status": "in_progress", "version": 4, "current_turn": {"player_id": "p1", "started_at": "2026-09-30T15:13:32.2439Z", "deadline_at": "2026-09-30T15:13:42.2439Z"}, "...": "the full Game"}, "events": [{"type": "move", "player_id": "p2", "row": 0, "col": 3, "value": 1, "result": "incorrect", "points": 0}]}

# the guest forfeits
event: update
data: {"game": {"id": "01M3SE1GA185KDKDVVVX830JV5", "status": "completed", "version": 5, "current_turn": null, "...": "the full Game"}, "events": [{"type": "player_forfeited", "player_id": "p2"}, {"type": "game_completed"}]}
```

(`game` is abbreviated here to keep the example short; in the real message it is the complete `Game`.)

### Events

A tagged union on `type`. Events inside one message are in chronological order; a single move can produce several.

| `type` | Extra fields | Sent when |
| --- | --- | --- |
| `move` | `player_id, row, col, value, result, points` | Every accepted move, correct or not. `result` is `correct` / `incorrect`. |
| `turn_expired` | `player_id` | That player's turn ran out. |
| `turn_skipped` | `player_id` | That player had a skip penalty consumed instead of getting the turn. |
| `fault_limit_reached` | `player_id, skip_turns` | That player just hit the fault limit. |
| `player_joined` | `player_id` | The guest joined an online lobby (sent to the host). |
| `player_forfeited` | `player_id` | That player forfeited. |
| `game_completed` | none | The game just ended, for any reason. Always the last event. |

`player_connection` (presence) is **not** emitted ([§11](#11-known-issues-and-limitations)).

---

## 9. Data models

### `User` (`GET /auth/me`)

| Field | Type | Notes |
| --- | --- | --- |
| `id` | string | ULID |
| `username` | string | always lowercase (lowercased when registered) |
| `created_at` | string | ISO-8601 UTC |

### Account session (`register`, `login`; in the envelope's `data`)

| Field | Type | Notes |
| --- | --- | --- |
| `id` | string | the account id |
| `username` | string | lowercase |
| `token` | string | account JWT, 30 minutes |
| `refresh_token` | string | 7-day JWT that **nothing accepts**; ignore it |

There is no nested `user` object.

### `Session` (`POST /games`, `POST /games/join`)

```ts
Session    = { game: Game, credentials: Credential[] }
Credential = { player_id: "p1" | "p2", token: string }   // token: send as X-Player-Token
```

`Credential.player_id` is the **seat** within this game; it equals `Player.id`, and is what `current_turn.player_id`,
`winner_id` and every event's `player_id` refer to. It is not the account's id.

| Field | Identifies | Scope |
| --- | --- | --- |
| `Player.user_id` | the account playing the seat, or `null` for a guest | global, permanent |
| `Player.name` | the account's username, or the guest name | display only |
| `Player.id` = `Credential.player_id` | the seat in this game | this game only |

### `Game`

| Field | Type | Notes |
| --- | --- | --- |
| `id` | string | ULID |
| `mode` | `"single"` \| `"versus"` | |
| `online` | boolean | |
| `difficulty` | `"easy"` \| `"medium"` \| `"hard"` | |
| `status` | `"waiting"` \| `"in_progress"` \| `"completed"` | `waiting` only for online lobbies |
| `join_code` | string \| null | set only while `waiting` |
| `puzzle` | `Grid` | initial clues; never changes |
| `board` | `Grid` | clues plus every correct fill; wrong fills are never stored |
| `players` | `Player[]` | turn order = array order. One entry for `single` and for a waiting lobby (host only); two otherwise. |
| `current_turn` | `Turn` \| null | `null` in `single`, while `waiting`, and once completed |
| `rules` | `Rules` | |
| `started_at` | string | creation time; reset to the join moment for online games |
| `completed_at` | string \| null | |
| `end_reason` | `"solved"` \| `"forfeit"` \| null | |
| `winner_id` | string \| null | seat id; `null` on a draw, in `single`, and while unfinished |
| `version` | integer | starts at 1, +1 per state change |
| `server_time` | string | for clock-skew correction |

`Grid` is `number[][]`, 9 rows of 9, `0` = empty.

### `Player`

| Field | Type | Notes |
| --- | --- | --- |
| `id` | string | `"p1"` / `"p2"` |
| `name` | string | |
| `user_id` | string \| null | account id; `null` for a same-device guest |
| `score` | integer | |
| `faults` | integer | toward the fault limit; resets to 0 once the skip penalty is served |
| `mistakes` | integer | total wrong fills, never resets |
| `skip_turns_remaining` | integer | |
| `connected` | boolean | **always `true`** (presence is not implemented) |
| `forfeit_at` | string \| null | **always `null`** |

### `Turn`

`{ player_id: string, started_at: string, deadline_at: string }`

### `Rules`

| Field | Value | Meaning |
| --- | --- | --- |
| `turn_limit_ms` | 10000 | turn length |
| `max_points` | 10 | points for an instant correct fill |
| `min_points` | 1 | floor for a correct fill |
| `fault_limit` | 3 | faults before the skip penalty |
| `skip_turns_on_fault_limit` | 2 | turns skipped once the limit is hit |
| `disconnect_forfeit_ms` | 60000 | present for compatibility; **no disconnect forfeit exists yet** |

### `MoveResponse` (`POST .../moves`, in `data`)

```ts
MoveResponse = {
  result: "correct" | "incorrect",
  points: number,             // 0 if incorrect, always 0 in single mode
  elapsed_ms: number | null,  // null in single mode
  game: Game,
  events: GameEvent[],
}
```

### `GameUpdate` (`turn/expire`, `forfeit` in `data`; every SSE `update`)

```ts
GameUpdate = { game: Game, events: GameEvent[] }
```

---

## 10. Error codes

Always read the HTTP status and `code`; `message` is for humans and some messages are inconsistent.
"Shape" is from [§3](#3-response-shapes).

**Game errors**

| `code` | HTTP | Shape | Meaning |
| --- | --- | --- | --- |
| `VALIDATION_ERROR` | 422 | A | Bad input, or a malformed body on the game endpoints. |
| `INVALID_TOKEN` | 401 | **C** for the account token, **A** for the player token | Missing/unknown/expired token. |
| `GAME_NOT_FOUND` | 404 | A | No such game (or an expired unjoined lobby). |
| `JOIN_CODE_NOT_FOUND` | 404 | A | No open lobby with that code. |
| `GAME_FULL` | 409 | A | Lost the race to join. No `data.game`. |
| `GAME_NOT_STARTED` | 409 | A | Online lobby still waiting. |
| `GAME_COMPLETED` | 409 | A | The game is over. |
| `NOT_YOUR_TURN` | 409 | A | Wrong seat for the current turn. |
| `TURN_EXPIRED` | 409 | A | A move arrived after the deadline (expiry already applied). |
| `TURN_NOT_EXPIRED` | 409 | A | `turn/expire` too early, or already done. |
| `NO_ACTIVE_TURN` | 409 | A | `turn/expire` on a single game. |
| `NOT_ONLINE` | 409 | A | `forfeit` on a non-online game. |
| `CELL_NOT_EMPTY` | 409 | A | The cell is already filled. |

Every game-endpoint `409` except `GAME_FULL` carries `data.game` (the account `409 conflict` on `register` does not).

**Account and general errors** (note: lowercase)

| `code` | HTTP | Shape | Meaning |
| --- | --- | --- | --- |
| `bad_request` | 400 | A | `register` / `login` body is not valid JSON. |
| `conflict` | 409 | A | `register`: that exact username already exists. |
| `invalid_credentials` | 401 | A | `login`: unknown user or wrong password. |
| `internal_server_error` | 500 | A | Unexpected server or database failure. |
| `INTERNAL_ERROR` | 500 | C | Same, from `GET /auth/me`. |

---

## 11. Known issues and limitations

These describe the server **as deployed today**. Items marked **bug** are defects to be fixed; items marked
**not built** are missing features. Update this section as they are resolved.

### Bugs in the account endpoints

1. **bug: no input validation on `register`.** A one-character username, names with spaces and punctuation, and even an
   empty body (empty username, empty password) all create an account. Enforce the intended rules client-side
   (username 3–20 letters/digits/underscore; password 6–72 characters).
2. **`register` conflict and some login messages are inconsistent.** `409 conflict` has an empty `message`; the login
   `401` message is `invalid credentials` for a wrong password but `unauthorized` for an unknown user. Key off `code`.

### Sessions

3. **The account token lasts 30 minutes and cannot be refreshed** ([§4](#4-authentication-and-identity)). The client has
   to handle `401 INVALID_TOKEN` by sending the user back to login. The `refresh_token` returned by `register`/`login`
   is unusable.

### Not built

4. **Presence and disconnect forfeit.** `connected` is always `true`, `forfeit_at` is always `null`, there is no
   `player_connection` event, and a silent disconnect never forfeits the game. Only an explicit `forfeit` ends a running
   online game early.
5. **No background timer.** Turns expire only when a request touches the game. The client must call `turn/expire`
   itself ([§5](#turn-expiry-the-client-must-drive-it)).
6. **Unjoined lobbies are never deleted.** After 30 minutes they answer `404`, but the rows stay in the database.
7. **Single instance only.** The event-stream fan-out is in memory; running more than one API replica would make
   streams miss each other's updates.
8. **No leaderboard, no "my games" listing, no rate limiting.**
9. **Malformed bodies answer differently per endpoint group**: `400 bad_request` on `register`/`login`,
    `422 VALIDATION_ERROR` on the game endpoints.

## 12. Differences from the original contract

For whoever adapts `baec-sudoku-web` (`docs/API.md` / `openapi.yaml` are the older contract):

| Area | Original contract | This API |
| --- | --- | --- |
| Success body | bare resource | wrapped in `{ code, message, data, servertime }` **except** `GET /auth/me` |
| Error body | `{ "error": { code, message, game? } }` | that shape **only** for account-token `401`s; otherwise `{ code, message, data, servertime }`; the game is at `data.game`, not `error.game` |
| `register` / `login` response | `{ user: {id, username, created_at}, token }` | `data: { id, username, token, refresh_token }`; no `user` object, no `created_at` |
| Username rules | 3–20 `[A-Za-z0-9_]`, unique case-insensitively | lowercased on register, so unique case-insensitively; format and length **not** enforced ([§11](#11-known-issues-and-limitations)) |
| Password rules | 6–72 chars | not enforced |
| Session length | no limit | account token expires after 30 minutes, no refresh |
| `refresh_token` | none | returned but unusable |
| Account error codes | `INVALID_CREDENTIALS`, `USERNAME_TAKEN` | `invalid_credentials`, `conflict` (lowercase; different names) |
| Turn timeouts | server timer fires at `deadline_at`; `turn/expire` is a fallback | **no timer**; `turn/expire` is the mechanism ([§5](#turn-expiry-the-client-must-drive-it)) |
| Presence | `connected` flips, `forfeit_at`, `player_connection`, forfeit after 60 s | none of it |
| Malformed body | `422 VALIDATION_ERROR` | `422` on game endpoints, `400 bad_request` on `register`/`login` |
| `GAME_FULL` | `409` | `409`, message `conflict`, no `data.game` |
| `POST /auth/logout` | invalidates the token | no-op `204` |
| SSE, endpoints, event types, game rules, scoring | as described | **same** (minus `player_connection`) |

## 13. Client checklist

1. Decode every response with the three-shape decoder ([§3](#3-response-shapes)).
2. After `POST /games` / `POST /games/join`, **persist `credentials`** (per game id). They are never shown again.
3. Validate usernames and passwords client-side (the server does not, [§11](#11-known-issues-and-limitations)), and show the
   `username` the server returns: it is lowercased.
4. Handle `401 INVALID_TOKEN` on account calls by returning to the login screen (the token lasts 30 minutes).
5. Run your own turn countdown from `current_turn.deadline_at`, corrected by `server_time`, and call
   `POST /games/{id}/turn/expire` when it hits zero. Treat `409 TURN_NOT_EXPIRED` as success (resync from `data.game`).
6. Open the event stream for online games. Keep only states with a higher `version`; announce each change's `events` once.
   On `EventSource` error with `readyState === CLOSED`, stop (the JSON error came from `401` / `404`).
7. On any game `409`, replace your local game with `data.game` instead of refetching (except `GAME_FULL`, which has none).
8. A `200` with `result: "incorrect"` is a normal outcome, not an error.
9. Do not show "opponent disconnected" UI: presence does not exist yet.
