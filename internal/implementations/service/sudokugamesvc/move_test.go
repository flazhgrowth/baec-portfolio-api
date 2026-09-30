package sudokugamesvc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokugame"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokuplayer"
)

type started struct {
	svc      *service
	tx       *fakeTx
	games    *fakeGameRepo
	players  *fakePlayerRepo
	moves    *fakeMoveRepo
	p1, p2   string // raw player tokens
	gameID   string
	turnFrom time.Time
}

// startedGame creates a real game through the service and freezes the clock at its first turn.
func startedGame(t *testing.T, req sudokugame.CreateSessionRequest) *started {
	t.Helper()
	svc, tx, games, players := newService()
	created, err := svc.CreateSession(context.Background(), caller, req)
	if err != nil {
		t.Fatal(err)
	}

	games.lobby = games.inserted[0]
	games.lobby.CreatedAt = now()
	players.seated = append(sudokuplayer.Players(nil), players.inserted...)
	*tx = fakeTx{} // count only what the test itself does
	s := &started{svc: svc, tx: tx, games: games, players: players, moves: svc.moveRepo.(*fakeMoveRepo), gameID: games.lobby.ID}
	s.p1 = created.Credentials[0].Token
	if len(created.Credentials) > 1 {
		s.p2 = created.Credentials[1].Token
	}
	if created.Game.CurrentTurn != nil {
		s.turnFrom = created.Game.CurrentTurn.StartedAt
		freezeNow(t, s.turnFrom)
	}
	return s
}

// cell returns the first empty cell of the stored game, its right value and a wrong one.
func (s *started) cell() (row, col, right, wrong int) {
	for i := 0; i < 81; i++ {
		if s.games.lobby.Board[i] == '0' {
			right = int(s.games.lobby.Solution[i] - '0')
			return i / 9, i % 9, right, right%9 + 1
		}
	}
	panic("no empty cell")
}

func num(n int) *float64 { f := float64(n); return &f }

func (s *started) move(token string, row, col, value int) (*sudokugame.MoveResponse, error) {
	return s.svc.SubmitMove(context.Background(), sudokugame.MoveRequest{
		GameID: s.gameID, Token: token, Row: num(row), Col: num(col), Value: num(value),
	})
}

func (s *started) play(token string, wrong bool) (*sudokugame.MoveResponse, error) {
	row, col, right, bad := s.cell()
	if wrong {
		return s.move(token, row, col, bad)
	}
	return s.move(token, row, col, right)
}

func conflict(t *testing.T, err error, code string) *sudokugame.ConflictError {
	t.Helper()
	var c *sudokugame.ConflictError
	if !errors.As(err, &c) || c.Code != code {
		t.Fatalf("want 409 %s, got %v", code, err)
	}
	return c
}

func TestSubmitMove(t *testing.T) {
	t.Run("a correct move is scored, saved, logged and announced", func(t *testing.T) {
		s := startedGame(t, sameDeviceVersus)
		listener, _ := s.svc.hub.subscribe(s.gameID)
		row, col, right, _ := s.cell()
		freezeNow(t, s.turnFrom.Add(2500*time.Millisecond))

		resp, err := s.move(s.p1, row, col, right)
		if err != nil {
			t.Fatal(err)
		}
		if resp.Result != "correct" || resp.Points != 8 || resp.ElapsedMs == nil || *resp.ElapsedMs != 2500 {
			t.Fatalf("response: %+v", resp)
		}
		if resp.Game.Version != 2 || resp.Game.CurrentTurn.PlayerID != "p2" || resp.Game.Players[0].Score != 8 {
			t.Fatalf("game: version=%d turn=%s score=%d", resp.Game.Version, resp.Game.CurrentTurn.PlayerID, resp.Game.Players[0].Score)
		}
		if resp.Game.Board[row][col] != right || resp.Game.Puzzle[row][col] != 0 {
			t.Fatal("the board gets the fill, the puzzle never changes")
		}
		if len(resp.Events) != 1 || resp.Events[0].Type != sudokugame.EventMove || *resp.Events[0].Points != 8 {
			t.Fatalf("events: %+v", resp.Events)
		}

		f, guard := s.games.updates[0], s.games.filters[0]
		if !f.IncrementVersion || f.TurnPlayerSeat.String != "p2" || f.Board.String[row*9+col] != byte('0'+right) {
			t.Fatalf("saved fields: %+v", f)
		}
		if !guard.Version.Valid || guard.Version.V != 1 {
			t.Fatalf("the save must be guarded by the version it read: %+v", guard.Version)
		}
		if s.players.updates["p1"].Score != 8 || len(s.players.updates) != 2 {
			t.Fatalf("seats saved: %+v", s.players.updates)
		}
		if len(s.moves.moves) != 1 {
			t.Fatal("the move must be logged")
		}
		if m := s.moves.moves[0]; !m.Correct || m.Points != 8 || m.ElapsedMs.Int64 != 2500 || m.GameVersion != 2 || m.Seat != "p1" || m.Row != row || m.Col != col {
			t.Fatalf("log row: %+v", m)
		}
		if s.tx.begun != 1 || s.tx.lastErr != nil {
			t.Fatalf("one committed transaction expected: %+v", s.tx)
		}

		got := next(t, &sudokugame.Subscription{Updates: listener})
		if got.Game.Version != 2 || len(got.Events) != 1 {
			t.Fatalf("the open stream must hear about it: %+v", got)
		}
	})

	t.Run("a wrong move costs a fault and is not written to the board", func(t *testing.T) {
		s := startedGame(t, sameDeviceVersus)
		row, col, _, _ := s.cell()
		resp, err := s.play(s.p1, true)
		if err != nil {
			t.Fatal(err)
		}
		if resp.Result != "incorrect" || resp.Points != 0 || resp.Game.Board[row][col] != 0 {
			t.Fatalf("%+v", resp)
		}
		if u := s.players.updates["p1"]; u.Faults != 1 || u.Mistakes != 1 {
			t.Fatalf("p1: %+v", u)
		}
		if m := s.moves.moves[0]; m.Correct || m.Points != 0 {
			t.Fatalf("incorrect moves are logged too: %+v", m)
		}
	})

	t.Run("single mode: no turn, no points, elapsed is null", func(t *testing.T) {
		s := startedGame(t, sudokugame.CreateSessionRequest{Mode: "single", DifficultyRaw: "easy"})
		resp, err := s.play(s.p1, false)
		if err != nil {
			t.Fatal(err)
		}
		if resp.Points != 0 || resp.ElapsedMs != nil || resp.Game.CurrentTurn != nil {
			t.Fatalf("%+v", resp)
		}
		if f := s.games.updates[0]; !f.ClearTurn || s.moves.moves[0].ElapsedMs.Valid {
			t.Fatalf("single has no turn to save and no elapsed to log: %+v", f)
		}
	})

	t.Run("solving the board completes the game and names the winner", func(t *testing.T) {
		s := startedGame(t, sameDeviceVersus)
		board := []byte(s.games.lobby.Solution)
		board[0] = '0' // everything filled except the first cell
		s.games.lobby.Board = string(board)
		s.players.seated[0].Score = 20

		resp, err := s.move(s.p1, 0, 0, int(s.games.lobby.Solution[0]-'0'))
		if err != nil {
			t.Fatal(err)
		}
		last := resp.Events[len(resp.Events)-1]
		if last.Type != sudokugame.EventGameCompleted || resp.Game.Status != "completed" || *resp.Game.WinnerID != "p1" || resp.Game.CurrentTurn != nil {
			t.Fatalf("%+v", resp.Game)
		}
		f := s.games.updates[0]
		if f.Status.String != "completed" || !f.ClearTurn || !f.CompletedAt.Valid || f.EndReason.String != "solved" || f.WinnerSeat.String != "p1" {
			t.Fatalf("saved fields: %+v", f)
		}
	})
}

func TestSubmitMoveRejections(t *testing.T) {
	t.Run("the wrong seat: 409 NOT_YOUR_TURN with the game, nothing saved", func(t *testing.T) {
		s := startedGame(t, sameDeviceVersus)
		_, err := s.play(s.p2, false)
		c := conflict(t, err, "NOT_YOUR_TURN")
		if c.Game.ID != s.gameID || c.Game.CurrentTurn.PlayerID != "p1" {
			t.Fatalf("the error must carry the current game: %+v", c.Game)
		}
		if len(s.games.updates) != 0 || len(s.moves.moves) != 0 || s.tx.begun != 0 {
			t.Fatal("a rejected move must write nothing")
		}
	})

	t.Run("an occupied cell", func(t *testing.T) {
		s := startedGame(t, sameDeviceVersus)
		var clue int
		for i := 0; i < 81; i++ {
			if s.games.lobby.Board[i] != '0' {
				clue = i
				break
			}
		}
		_, err := s.move(s.p1, clue/9, clue%9, int(s.games.lobby.Solution[clue]-'0'))
		conflict(t, err, "CELL_NOT_EMPTY")
		if len(s.games.updates) != 0 {
			t.Fatal("nothing to save")
		}
	})

	t.Run("a late move: the timeout is saved and announced, then the move is refused", func(t *testing.T) {
		s := startedGame(t, sameDeviceVersus)
		listener, _ := s.svc.hub.subscribe(s.gameID)
		row, col, right, _ := s.cell()
		freezeNow(t, s.turnFrom.Add(10*time.Second))

		_, err := s.move(s.p1, row, col, right)
		c := conflict(t, err, "TURN_EXPIRED")
		if c.Game.CurrentTurn.PlayerID != "p2" || c.Game.Version != 2 {
			t.Fatalf("the caller must see the new turn, not the old: %+v", c.Game.CurrentTurn)
		}
		if len(s.games.updates) != 1 || s.games.updates[0].TurnPlayerSeat.String != "p2" {
			t.Fatalf("the expiry must be committed even though the request failed: %+v", s.games.updates)
		}
		if len(s.moves.moves) != 0 || c.Game.Board[row][col] != 0 {
			t.Fatal("the late move itself must not be applied or logged")
		}
		got := next(t, &sudokugame.Subscription{Updates: listener})
		if len(got.Events) != 1 || got.Events[0].Type != sudokugame.EventTurnExpired {
			t.Fatalf("the expiry must reach the stream: %+v", got.Events)
		}
	})

	t.Run("a lobby, and a finished game", func(t *testing.T) {
		s := startedGame(t, sudokugame.CreateSessionRequest{Mode: "versus", DifficultyRaw: "easy", IsOnline: true})
		_, err := s.move(s.p1, 0, 0, 1)
		conflict(t, err, "GAME_NOT_STARTED")

		s = startedGame(t, sameDeviceVersus)
		s.games.lobby.Status = sudokugame.StatusCompleted
		_, err = s.move(s.p1, 0, 0, 1)
		conflict(t, err, "GAME_COMPLETED")
	})

	t.Run("token and game are checked before the input", func(t *testing.T) {
		s := startedGame(t, sameDeviceVersus)
		bad := sudokugame.MoveRequest{GameID: s.gameID, Token: "wrong", Row: num(99), Col: num(0), Value: num(1)}
		_, err := s.svc.SubmitMove(context.Background(), bad)
		assertCode(t, err, 401, "INVALID_TOKEN")

		bad.Token = ""
		_, err = s.svc.SubmitMove(context.Background(), bad)
		assertCode(t, err, 401, "INVALID_TOKEN")

		bad.GameID = "nope"
		_, err = s.svc.SubmitMove(context.Background(), bad)
		assertCode(t, err, 404, "GAME_NOT_FOUND")

		// p2's token must not work on a game it does not belong to: seats are per game
		s.players.seated[1].GameID = "elsewhere"
	})

	t.Run("bad input is VALIDATION_ERROR", func(t *testing.T) {
		s := startedGame(t, sameDeviceVersus)
		half := 1.5
		for name, req := range map[string]sudokugame.MoveRequest{
			"row too high": {Row: num(9), Col: num(0), Value: num(1)},
			"negative col": {Row: num(0), Col: num(-1), Value: num(1)},
			"value zero":   {Row: num(0), Col: num(0), Value: num(0)},
			"value ten":    {Row: num(0), Col: num(0), Value: num(10)},
			"missing row":  {Col: num(0), Value: num(1)},
			"missing col":  {Row: num(0), Value: num(1)},
			"missing val":  {Row: num(0), Col: num(0)},
			"fractional":   {Row: &half, Col: num(0), Value: num(1)},
		} {
			req.GameID, req.Token = s.gameID, s.p1
			_, err := s.svc.SubmitMove(context.Background(), req)
			assertCode(t, err, 422, "VALIDATION_ERROR")
			if len(s.games.updates) != 0 {
				t.Fatalf("%s: wrote state", name)
			}
		}
	})
}

func TestSubmitMoveConcurrency(t *testing.T) {
	t.Run("losing a write race restarts from a fresh read and can then succeed", func(t *testing.T) {
		s := startedGame(t, sameDeviceVersus)
		s.games.updateQueue = []error{errVersionConflictSQL(), nil}

		resp, err := s.play(s.p1, false)
		if err != nil {
			t.Fatal(err)
		}
		if resp.Result != "correct" || len(s.moves.moves) != 1 || len(s.games.updates) != 1 {
			t.Fatalf("exactly one move may be applied: moves=%d saves=%d", len(s.moves.moves), len(s.games.updates))
		}
		if s.tx.begun != 2 || s.tx.lastErr != nil {
			t.Fatalf("two transactions, the first rolled back: %+v", s.tx)
		}
	})

	t.Run("after losing, the fresh read decides: the rival moved first, so it is not our turn", func(t *testing.T) {
		s := startedGame(t, sameDeviceVersus)
		s.games.updateQueue = []error{errVersionConflictSQL()}
		s.games.onUpdate = func(stored *sudokugame.Game) { // the rival's move lands in between
			stored.TurnPlayerSeat.String = "p2"
			stored.Version = 2
		}
		_, err := s.play(s.p1, false)
		conflict(t, err, "NOT_YOUR_TURN")
		if len(s.moves.moves) != 0 {
			t.Fatal("the loser must not log a move")
		}
	})

	t.Run("gives up after repeated losses", func(t *testing.T) {
		s := startedGame(t, sameDeviceVersus)
		s.games.updateErr = errVersionConflictSQL()
		_, err := s.play(s.p1, false)
		assertCode(t, err, 500, "")
		if len(s.moves.moves) != 0 {
			t.Fatal("nothing may be logged")
		}
	})

	t.Run("a failure saving a seat rolls everything back and announces nothing", func(t *testing.T) {
		s := startedGame(t, sameDeviceVersus)
		listener, _ := s.svc.hub.subscribe(s.gameID)
		s.players.updateErr = errors.New("db down")
		_, err := s.play(s.p1, false)
		assertCode(t, err, 500, "")
		if s.tx.lastErr == nil || len(s.moves.moves) != 0 {
			t.Fatal("the transaction must roll back with the error, before the move is logged")
		}
		quiet(t, &sudokugame.Subscription{Updates: listener})
	})

	t.Run("the move log failing rolls the move back", func(t *testing.T) {
		s := startedGame(t, sameDeviceVersus)
		s.moves.err = errors.New("db down")
		_, err := s.play(s.p1, false)
		assertCode(t, err, 500, "")
		if s.tx.lastErr == nil {
			t.Fatal("transaction must roll back")
		}
	})
}

func errVersionConflictSQL() error { return errNoRows }
