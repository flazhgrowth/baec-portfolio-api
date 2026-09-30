package sudokugamesvc

import (
	"context"
	"testing"
	"time"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokugame"
)

func (s *started) expire() (*sudokugame.GameUpdate, error) {
	return s.svc.ExpireTurn(context.Background(), sudokugame.ExpireTurnRequest{GameID: s.gameID})
}

func TestExpireTurn(t *testing.T) {
	t.Run("after the deadline: hands the turn over, saves it, announces it", func(t *testing.T) {
		s := startedGame(t, sameDeviceVersus)
		listener, _ := s.svc.hub.subscribe(s.gameID)
		late := s.turnFrom.Add(10 * time.Second)
		freezeNow(t, late)

		update, err := s.expire()
		if err != nil {
			t.Fatal(err)
		}
		if len(update.Events) != 1 || update.Events[0].Type != sudokugame.EventTurnExpired || update.Events[0].PlayerID != "p1" {
			t.Fatalf("events: %+v", update.Events)
		}
		turn := update.Game.CurrentTurn
		if turn.PlayerID != "p2" || !turn.StartedAt.Equal(late) || update.Game.Version != 2 {
			t.Fatalf("the next turn starts now and goes to p2: %+v version %d", turn, update.Game.Version)
		}
		if len(s.games.updates) != 1 || s.games.filters[0].Version.V != 1 {
			t.Fatalf("saved once, guarded by the version read: %+v", s.games.filters)
		}
		if len(s.moves.moves) != 0 {
			t.Fatal("an expiry is not a move")
		}
		if got := next(t, &sudokugame.Subscription{Updates: listener}); len(got.Events) != 1 || got.Game.Version != 2 {
			t.Fatalf("the open stream must hear about it: %+v", got)
		}
	})

	t.Run("the deadline instant itself counts", func(t *testing.T) {
		s := startedGame(t, sameDeviceVersus)
		freezeNow(t, s.turnFrom.Add(10*time.Second))
		if _, err := s.expire(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("before the deadline: 409 TURN_NOT_EXPIRED with the current game, nothing written", func(t *testing.T) {
		s := startedGame(t, sameDeviceVersus)
		listener, _ := s.svc.hub.subscribe(s.gameID)
		freezeNow(t, s.turnFrom.Add(9999*time.Millisecond))

		_, err := s.expire()
		c := conflict(t, err, "TURN_NOT_EXPIRED")
		if c.Game.CurrentTurn.PlayerID != "p1" || c.Game.Version != 1 {
			t.Fatalf("%+v", c.Game.CurrentTurn)
		}
		if len(s.games.updates) != 0 || s.tx.begun != 0 {
			t.Fatal("an early call must not write")
		}
		quiet(t, &sudokugame.Subscription{Updates: listener})
	})

	t.Run("calling twice: the second finds nothing to expire", func(t *testing.T) {
		s := startedGame(t, sameDeviceVersus)
		s.games.applyUpdates = true
		freezeNow(t, s.turnFrom.Add(10*time.Second))

		if _, err := s.expire(); err != nil {
			t.Fatal(err)
		}
		_, err := s.expire()
		c := conflict(t, err, "TURN_NOT_EXPIRED")
		if c.Game.CurrentTurn.PlayerID != "p2" || c.Game.Version != 2 {
			t.Fatalf("the second caller sees the state the first one produced: %+v v%d", c.Game.CurrentTurn, c.Game.Version)
		}
		if len(s.games.updates) != 1 {
			t.Fatalf("expired once, not twice: %d saves", len(s.games.updates))
		}
	})

	t.Run("an owed skip is consumed, so the turn can come straight back", func(t *testing.T) {
		s := startedGame(t, sameDeviceVersus)
		s.players.seated[1].SkipTurnsRemaining = 1 // p2 owes a turn
		s.players.seated[1].Faults = 3
		freezeNow(t, s.turnFrom.Add(10*time.Second))

		update, err := s.expire()
		if err != nil {
			t.Fatal(err)
		}
		kinds := []string{update.Events[0].Type, update.Events[1].Type}
		if len(update.Events) != 2 || kinds[0] != sudokugame.EventTurnExpired || kinds[1] != sudokugame.EventTurnSkipped || update.Events[1].PlayerID != "p2" {
			t.Fatalf("events: %+v", update.Events)
		}
		if update.Game.CurrentTurn.PlayerID != "p1" {
			t.Fatal("p1 plays again because p2's turn was skipped")
		}
		if u := s.players.updates["p2"]; u.SkipTurnsRemaining != 0 || u.Faults != 0 {
			t.Fatalf("the consumed skip must be saved, faults reset: %+v", u)
		}
	})

	t.Run("nothing to expire: lobby, single game, finished game", func(t *testing.T) {
		lobby := startedGame(t, sudokugame.CreateSessionRequest{Mode: "versus", DifficultyRaw: "easy", IsOnline: true})
		_, err := lobby.expire()
		conflict(t, err, "GAME_NOT_STARTED")

		single := startedGame(t, sudokugame.CreateSessionRequest{Mode: "single", DifficultyRaw: "easy"})
		freezeNow(t, now().Add(24*time.Hour))
		_, err = single.expire()
		conflict(t, err, "NO_ACTIVE_TURN")

		done := startedGame(t, sameDeviceVersus)
		done.games.lobby.Status = sudokugame.StatusCompleted
		_, err = done.expire()
		conflict(t, err, "GAME_COMPLETED")

		for _, s := range []*started{lobby, single, done} {
			if len(s.games.updates) != 0 {
				t.Fatal("must not write")
			}
		}
	})

	t.Run("unknown game is 404", func(t *testing.T) {
		s := startedGame(t, sameDeviceVersus)
		_, err := s.svc.ExpireTurn(context.Background(), sudokugame.ExpireTurnRequest{GameID: "nope"})
		assertCode(t, err, 404, "GAME_NOT_FOUND")
	})

	t.Run("losing a race to someone who already expired the turn answers TURN_NOT_EXPIRED", func(t *testing.T) {
		s := startedGame(t, sameDeviceVersus)
		late := s.turnFrom.Add(10 * time.Second)
		freezeNow(t, late)
		s.games.updateQueue = []error{errNoRows}
		s.games.onUpdate = func(stored *sudokugame.Game) { // the server's own timer wins
			stored.TurnPlayerSeat.String = "p2"
			stored.TurnStartedAt.Time = late
			stored.TurnDeadlineAt.Time = late.Add(10 * time.Second)
			stored.Version = 2
		}

		_, err := s.expire()
		c := conflict(t, err, "TURN_NOT_EXPIRED")
		if c.Game.CurrentTurn.PlayerID != "p2" || c.Game.Version != 2 {
			t.Fatalf("%+v", c.Game.CurrentTurn)
		}
	})

	t.Run("a lost race where the turn is still due is retried and succeeds", func(t *testing.T) {
		s := startedGame(t, sameDeviceVersus)
		freezeNow(t, s.turnFrom.Add(10*time.Second))
		s.games.updateQueue = []error{errNoRows, nil}
		if _, err := s.expire(); err != nil {
			t.Fatal(err)
		}
		if len(s.games.updates) != 1 {
			t.Fatalf("expired exactly once: %d", len(s.games.updates))
		}
	})
}
