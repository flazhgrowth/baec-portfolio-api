# Sudoku backend: design (draft for review)

Source: `../baec-sudoku-web/docs/API.md` (base URL `/ms/sudous/api/v1`). This doc covers the database, the
layering in this repo, the game engine, realtime, and the open questions I need you to decide.
Nothing here is implemented yet except what "Current state" lists.

## 1. Current state in this repo

| Exists | Notes |
| --- | --- |
| `accounts` table + `account` entity, `register`/`login`/`me` | Columns: `id (ULID), username, password, name, salt, created_at, updated_at`. |
| `entity/sudoku` | Board generator, puzzle carving, `Difficulty` (easy 40 / medium 32 / hard 26 clues). |
| `entity/sudokugame` | Only `CreateSessionRequest` and a stub response. |
| `entity/sudokuplayer` | Empty. |

Gaps between the existing account code and the API contract:

1. **Username uniqueness is case-insensitive**: needs a unique index on `lower(username)`.
2. **Username format** (3–20 of `[A-Za-z0-9_]`) and **password length** (6–72) are validation rules to enforce in `accountsvc`.
3. **Account tokens are stateless JWTs** (the existing approach). No token table. Consequence: `POST /auth/logout` cannot truly
   invalidate a token. It is a no-op that returns `204` (the contract already says it is idempotent and that old tokens keep
   working, so the client just discards its token). A stolen JWT stays valid until it expires, so use a sensible `exp`.
4. `POST /auth/logout` route (no-op 204) and `GET /auth/me` returning `{id, username, created_at}` (not `name`/`salt`/`password`).
5. The API has no separate `name`. `accounts.name` can stay (display name defaults to username) but the game uses `username`.

## 2. Decisions baked into the design

- **Postgres**, ULID text ids (matches `BaseULIDModel`). Game and player ids are ULIDs too.
- **One row per game holds the live state** (`board`, `current_turn`, `version`, …). Moves are also appended to a log table.
  The game row is what the rules engine loads, mutates and saves under a row lock.
- **Grids stored as an 81-char string** (`'530070000…'`, row-major, `0` = empty). Compact, trivially comparable, and cheap to
  diff. The API's `[][]int` is produced at the edge.
- **Solution stored on the game row, never serialized into any response.**
- **Seats** (`p1`, `p2`) are rows in `game_players`, keyed by `(game_id, seat)`. The API's `Player.id` is the seat string.
- **Account tokens are stateless JWTs**, nothing stored. **Player tokens** (`X-Player-Token`) are different: they are opaque
  random values stored hashed (SHA-256) in `sudoku_game_players.token_hash`, so a leaked DB dump can't hand out seats.
- **Rules snapshot per game** (`rules` JSONB) so changing defaults later doesn't alter games in flight, and the response can
  echo `Game.rules` exactly.
- **Timestamps are `timestamptz`** and serialized as UTC ISO-8601.

## 3. Schema

```sql
-- 001: accounts (extend what exists)
CREATE UNIQUE INDEX accounts_username_lower_uq ON accounts (lower(username));


-- 002: games
CREATE TYPE sudoku_mode        AS ENUM ('single', 'versus');
CREATE TYPE sudoku_difficulty  AS ENUM ('easy', 'medium', 'hard');
CREATE TYPE sudoku_status      AS ENUM ('waiting', 'in_progress', 'completed');
CREATE TYPE sudoku_end_reason  AS ENUM ('solved', 'forfeit');

CREATE TABLE sudoku_games (
    id                 TEXT PRIMARY KEY,           -- ULID
    mode               sudoku_mode       NOT NULL,
    online             BOOLEAN           NOT NULL DEFAULT FALSE,
    difficulty         sudoku_difficulty NOT NULL,
    status             sudoku_status     NOT NULL,
    join_code          TEXT,                       -- set only while waiting
    puzzle             CHAR(81)          NOT NULL, -- initial clues, immutable
    solution           CHAR(81)          NOT NULL, -- NEVER returned to clients
    board              CHAR(81)          NOT NULL, -- clues + correct fills
    rules              JSONB             NOT NULL, -- snapshot of Rules (see API "Rules")
    -- current turn (all NULL in single, while waiting, and once completed)
    turn_player_seat   TEXT,                       -- 'p1' | 'p2'
    turn_started_at    TIMESTAMPTZ,
    turn_deadline_at   TIMESTAMPTZ,
    started_at         TIMESTAMPTZ       NOT NULL,
    completed_at       TIMESTAMPTZ,
    end_reason         sudoku_end_reason,
    winner_seat        TEXT,                       -- NULL on draw / single / not finished
    version            INTEGER           NOT NULL DEFAULT 1,
    created_by         TEXT              NOT NULL REFERENCES accounts(id),
    created_at         TIMESTAMPTZ       NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ       NOT NULL DEFAULT now(),

    CONSTRAINT online_only_versus  CHECK (NOT online OR mode = 'versus'),
    CONSTRAINT waiting_only_online CHECK (status <> 'waiting' OR online),
    CONSTRAINT join_code_iff_waiting CHECK ((join_code IS NOT NULL) = (status = 'waiting')),
    CONSTRAINT turn_iff_running CHECK (
        (status = 'in_progress' AND mode = 'versus')
        = (turn_player_seat IS NOT NULL AND turn_started_at IS NOT NULL AND turn_deadline_at IS NOT NULL)
    ),
    CONSTRAINT completed_fields CHECK ((status = 'completed') = (completed_at IS NOT NULL AND end_reason IS NOT NULL))
);

-- join code unique among *open* lobbies only (codes get reused after a game starts)
CREATE UNIQUE INDEX sudoku_games_join_code_uq ON sudoku_games (upper(join_code)) WHERE status = 'waiting';
-- sweeper: stale lobbies, due turns
CREATE INDEX sudoku_games_waiting_idx  ON sudoku_games (created_at)       WHERE status = 'waiting';
CREATE INDEX sudoku_games_deadline_idx ON sudoku_games (turn_deadline_at) WHERE status = 'in_progress';
CREATE INDEX sudoku_games_creator_idx  ON sudoku_games (created_by, created_at DESC);

-- 003: seats
CREATE TABLE sudoku_game_players (
    game_id               TEXT        NOT NULL REFERENCES sudoku_games(id) ON DELETE CASCADE,
    seat                  TEXT        NOT NULL CHECK (seat IN ('p1', 'p2')),   -- == API Player.id
    user_id               TEXT        REFERENCES accounts(id),                 -- NULL = same-device guest
    name                  TEXT        NOT NULL,
    token_hash            TEXT        NOT NULL UNIQUE,                         -- X-Player-Token, hashed
    score                 INTEGER     NOT NULL DEFAULT 0,
    faults                INTEGER     NOT NULL DEFAULT 0,
    mistakes              INTEGER     NOT NULL DEFAULT 0,
    skip_turns_remaining  INTEGER     NOT NULL DEFAULT 0,
    connected             BOOLEAN     NOT NULL DEFAULT TRUE,
    last_seen_at          TIMESTAMPTZ NOT NULL DEFAULT now(),                  -- presence clock
    forfeit_at            TIMESTAMPTZ,
    PRIMARY KEY (game_id, seat)
);
CREATE INDEX sudoku_game_players_user_idx ON sudoku_game_players (user_id, game_id) WHERE user_id IS NOT NULL;
CREATE INDEX sudoku_game_players_forfeit_idx ON sudoku_game_players (forfeit_at) WHERE forfeit_at IS NOT NULL;

ALTER TABLE sudoku_games
    ADD CONSTRAINT sudoku_games_turn_fk   FOREIGN KEY (id, turn_player_seat) REFERENCES sudoku_game_players (game_id, seat) DEFERRABLE INITIALLY DEFERRED,
    ADD CONSTRAINT sudoku_games_winner_fk FOREIGN KEY (id, winner_seat)      REFERENCES sudoku_game_players (game_id, seat) DEFERRABLE INITIALLY DEFERRED;

-- 004: move + event log (append-only)
CREATE TABLE sudoku_moves (
    id           BIGSERIAL PRIMARY KEY,
    game_id      TEXT        NOT NULL REFERENCES sudoku_games(id) ON DELETE CASCADE,
    seat         TEXT        NOT NULL,
    row          SMALLINT    NOT NULL CHECK (row BETWEEN 0 AND 8),
    col          SMALLINT    NOT NULL CHECK (col BETWEEN 0 AND 8),
    value        SMALLINT    NOT NULL CHECK (value BETWEEN 1 AND 9),
    correct      BOOLEAN     NOT NULL,
    points       INTEGER     NOT NULL DEFAULT 0,
    elapsed_ms   INTEGER,                               -- NULL in single mode
    game_version INTEGER     NOT NULL,                  -- version after this move
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX sudoku_moves_game_idx ON sudoku_moves (game_id, id);
```

### Why these shapes

- **`turn_iff_running`**: the contract says `current_turn` is null in `single`, while `waiting`, and when completed. The CHECK
  makes an impossible state unrepresentable.
- **`join_code` partial unique index**: "unique among open lobbies". Plain `UNIQUE` would leak codes forever and eventually
  collide. Lookup is case-insensitive via `upper()`; generate codes already uppercase from `ABCDEFGHJKLMNPQRSTUVWXYZ23456789`.
  On a unique violation, regenerate and retry.
- **Seat FKs are deferrable** because `sudoku_games` and `sudoku_game_players` reference each other; both rows are inserted in
  one transaction.
- **`sudoku_moves` stores only the log**, not events. `GameEvent`s (`turn_expired`, `turn_skipped`, `fault_limit_reached`,
  `player_joined`, …) are derived while processing and pushed over SSE; they aren't replayed on reconnect (the contract resyncs
  from full state), so persisting them buys nothing. If you want a leaderboard or replay later, `sudoku_moves` + `sudoku_game_players`
  is sufficient.
- **`last_seen_at` on the seat** implements the 15 s presence rule without a separate table.
- **Leaderboard** (API says "later") needs no schema now: aggregate `sudoku_game_players.user_id` joined to
  `sudoku_games.winner_seat`/`status = 'completed'`. Add a materialized view when it's actually built.

## 4. API → storage mapping

| Endpoint | Reads / writes |
| --- | --- |
| `POST /auth/register` | insert `accounts`, sign a JWT → `AuthSession`. 409 `USERNAME_TAKEN` on the `lower(username)` unique violation. |
| `POST /auth/login` | lookup by `lower(username)`, verify hash (existing `ValidatePassword`), sign a new JWT. Same 401 for unknown user and bad password. |
| `GET /auth/me` | verify the JWT, load the account by `sub`. |
| `POST /auth/logout` | no-op, always 204 (stateless JWT). |
| `POST /games` | generate puzzle+solution; insert game + seat(s). `single`: 1 seat, `in_progress`, no turn. Same-device versus: 2 seats (p2 `user_id` NULL, name from `player_names[1]`, default "Player 2"), `in_progress`, turn = p1. Online: 1 seat, `waiting`, `join_code`. |
| `POST /games/join` | lock the `waiting` game by `upper(join_code)`; insert p2; set `in_progress`, `join_code = NULL`, `started_at = now()`, turn = p1. Lost race → `GAME_FULL`; no match → `JOIN_CODE_NOT_FOUND`. |
| `GET /games/{id}` | load, **apply due state** (§5), return `Game` (minus `solution`). |
| `GET /games/{id}/events` | resolve seat by `token_hash`; register SSE subscriber; send full `update`. |
| `POST /games/{id}/moves` | lock game, apply due state, validate, update board/seat/turn/version, insert `sudoku_moves`. |
| `POST /games/{id}/turn/expire` | lock game, apply due state; 409 `TURN_NOT_EXPIRED` if nothing was due. |
| `POST /games/{id}/forfeit` | lock game, mark completed with `end_reason = forfeit`, winner = other seat. |

`Game` response assembly: `puzzle`/`board` are expanded from the 81-char strings; `players` are ordered `seat ASC`;
`current_turn` built from the three `turn_*` columns; `version`, `server_time = now()`.

## 5. Concurrency and the rules engine

Every game-touching request runs in **one transaction**:

```
BEGIN
  SELECT … FROM sudoku_games WHERE id = $1 FOR UPDATE        -- serialises moves/expiry/forfeit per game
  SELECT … FROM sudoku_game_players WHERE game_id = $1
  engine.ApplyDue(game, now)       -- turn timeout, presence flips, disconnect forfeit
  COMMIT-point A: persist ApplyDue changes
  engine.Action(game, …)           -- move / forfeit / nothing
  persist + version++
COMMIT
```

The contract requires that lazily-applied state **commits even if the action then fails** (late move → expire the turn, save,
then reply `409 TURN_EXPIRED` with the new game). So the handler must persist `ApplyDue` results, then return the domain error
as a value rather than rolling back. Design: the engine returns `(newState, events, err)`; the service always saves
`newState` and only then surfaces `err`. No rollback path for domain errors, only for DB errors.

**Engine is pure Go** (`internal/entity/sudokugame` or a new `sudokuengine` package): `ApplyDue`, `Move`, `Forfeit`, `Join`,
with `now` injected so it is unit-testable, mirroring `src/mock/engine.ts` and `engine.test.ts` in the web repo. Port those tests.

Scoring: `max(min_points, max_points - floor(elapsed_ms/1000))`. A wrong fill isn't written to `board`; faults/mistakes +1;
turn ends. At `fault_limit` set `skip_turns_remaining = skip_turns_on_fault_limit`; consume on each return of the turn; reset
`faults = 0` when the last skip is consumed. Completion: `board == solution`, winner = higher score or NULL on draw.

**Turn timer.** A background goroutine in the API process:
a sweeper ticking about every 500 ms that runs
`SELECT id FROM sudoku_games WHERE status='in_progress' AND turn_deadline_at <= now()` (uses `sudoku_games_deadline_idx`), then
applies `ApplyDue` per game under the row lock, publishes the events. Same sweeper handles presence flips, disconnect forfeits
(`forfeit_at <= now()`) and discards stale lobbies (`status='waiting' AND created_at < now() - 30 min` → delete; later requests
get 404 `GAME_NOT_FOUND`). Lazy application on request remains the correctness guarantee; the sweeper is for clients that are gone.

## 6. Realtime (SSE)

- In-process hub: `map[gameID]map[subscriber]chan GameUpdate`. After a committing transaction, the service publishes
  `GameUpdate{game, events}` to the hub.
- Handler: on connect send `update` with current state and `events: []`; `: ping` comment every 10 s; subscriber close marks
  the seat's `last_seen_at` and starts the disconnect clock.
- Presence: open stream counts as connected; otherwise connected if `last_seen_at > now - 15 s`. Any authenticated request
  touches `last_seen_at`. A flip writes `connected`, bumps `version`, emits `player_connection`.
- **Single instance assumption.** The hub is in memory, so the API must run as one replica (the Dockerfile suggests it does).
  If it ever scales out, swap the hub's publish for Postgres `LISTEN/NOTIFY` (payload = game id + version) and have each
  instance re-read and fan out. The schema needs no change for that.
- Note: check that the HTTP server's write timeout (`etc/config/config.yaml` has `write: 30`) does not kill long-lived SSE
  responses. Needs `http.ResponseController.SetWriteDeadline(time.Time{})` on the events handler.

## 7. Code layout (follows the repo's existing pattern)

```
internal/entity/sudokugame/        Game, Rules, Turn, GameEvent, request/response DTOs, GameRepository interface
internal/entity/sudokuplayer/      Player, Credential, PlayerRepository interface
internal/entity/sudoku/            (existing) generator; add Grid <-> string codec + clue-count / uniqueness check
internal/implementations/repository/db/{sudokugamerepo,sudokuplayerrepo}/
internal/implementations/service/sudokusvc/   create.go join.go move.go expire.go forfeit.go get.go due.go
internal/implementations/service/sudokusvc/hub.go, sweeper.go
internal/implementations/api/sudokuapi/       handlers incl. events.go (SSE)
internal/implementations/routes/sudokuroutes/ mounted at /ms/sudous/api/v1 (auth + games)
migrations/                                   001…004 above (the repo has none yet, see Q1)
```

Errors use `apierrors` with the codes from the API error table. Note the API envelope is `{"error":{"code","message","game?"}}`;
the existing `apierrors` output may differ, so a thin adapter for the sudoku routes may be needed (Q5).

## 8. Puzzle generation

The generator carves cells and keeps a removal only if the puzzle still has exactly one solution (capped backtracking
count). Verified by test for all three difficulties: exact clue count, unique solution, clues match the solution. Hard takes
roughly 30-300 ms, done synchronously in `POST /games`.

Bug fixed while implementing `POST /games`: `Board.ToPuzzle` used `maps.Clone`, a shallow copy of a map of slices, so carving
the puzzle also blanked the stored solution. It now deep-copies.

## 9. Open questions (please decide)

1. **Migrations:** this repo has no migration files or tool. Options: plain `.sql` files in `migrations/` applied by
   `golang-migrate`, or reuse `../gramedia-migrator`. I'd pick golang-migrate behind a `make migrate` target.
2. **JWT expiry:** the contract has no session limit. What `exp` does the current JWT use? A long TTL (e.g. 30 days) is
   the closest fit, since logout can't revoke anything.
3. **Retention:** keep completed games and moves forever (leaderboard/replay), or prune after N days? I assume keep.
4. **Response envelope:** confirmed by running the server: every response, success and error, is wrapped as
   `{"code","message","data","servertime"}` and error codes are lowercase (`unauthorized`, `bad_request`). The contract wants bare
   bodies (`Session`, `{"error":{code,message,game}}`) and uppercase codes. `POST /games` currently returns the repo's envelope
   with the contract's `VALIDATION_ERROR` code for 422s. Decide: adapt the sudoku routes to the contract, or have the web client
   read `data`?
5. **Single replica** acceptable for SSE (in-memory hub)? If not, I'll design it on `LISTEN/NOTIFY` from the start.
6. **Same-device versus and the turn timer:** the server sweeper will expire turns for these too (simplest, consistent). The
   web client's `turn/expire` call then mostly returns `409 TURN_NOT_EXPIRED`. Fine?
7. **`accounts.name`:** keep (defaulting to username) or drop from the sudoku flow entirely?
8. **Does `GET /games/{id}` stay unauthenticated?** The contract says yes, with a ULID game id as the only protection. It
   returns the board but never the solution or tokens, so I left it as specified.

## 10. Suggested build order

1. Migrations + account hardening (case-insensitive username, logout no-op, validation).
2. Grid codec, unique-solution generator, pure engine with ported tests.
3. Repos + create/join/get.
4. Moves, expire, forfeit with the commit-then-error pattern.
5. SSE hub, presence, sweeper.
6. Contract tests against `openapi.yaml`.
