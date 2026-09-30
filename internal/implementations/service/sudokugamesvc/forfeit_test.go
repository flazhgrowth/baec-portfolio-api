package sudokugamesvc

import (
	"context"
	"testing"
	"time"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokugame"
)

// startedOnline is a running online game: both seats, p1 on turn.
func startedOnline(t *testing.T) *started {
	t.Helper()
	s := startedGame(t, sameDeviceVersus)
	s.games.lobby.Online = true
	return s
}

func (s *started) forfeit(token string) (*sudokugame.GameUpdate, error) {
	return s.svc.Forfeit(context.Background(), sudokugame.ForfeitRequest{GameID: s.gameID, Token: token})
}

func TestForfeit(t *testing.T) {
	t.Run("the caller loses: saved, announced, winner is the opponent", func(t *testing.T) {
		s := startedOnline(t)
		listener, _ := s.svc.hub.subscribe(s.gameID)

		update, err := s.forfeit(s.p2) // p2 leaves while it is p1's turn
		if err != nil {
			t.Fatal(err)
		}
		g := update.Game
		if g.Status != "completed" || *g.EndReason != "forfeit" || *g.WinnerID != "p1" || g.CurrentTurn != nil || g.Version != 2 {
			t.Fatalf("game: %+v", g)
		}
		if len(update.Events) != 2 || update.Events[0].Type != sudokugame.EventPlayerForfeited || update.Events[0].PlayerID != "p2" || update.Events[1].Type != sudokugame.EventGameCompleted {
			t.Fatalf("events: %+v", update.Events)
		}

		f, guard := s.games.updates[0], s.games.filters[0]
		if f.Status.String != "completed" || !f.ClearTurn || f.EndReason.String != "forfeit" || f.WinnerSeat.String != "p1" || !f.CompletedAt.Valid {
			t.Fatalf("saved fields: %+v", f)
		}
		if guard.Version.V != 1 || len(s.moves.moves) != 0 {
			t.Fatal("guarded by the version read; a forfeit is not a move")
		}
		got := next(t, &sudokugame.Subscription{Updates: listener})
		if got.Game.Status != "completed" || len(got.Events) != 2 {
			t.Fatalf("the opponent's stream must hear about it: %+v", got)
		}
	})

	t.Run("then nothing more can happen", func(t *testing.T) {
		s := startedOnline(t)
		s.games.applyUpdates = true
		if _, err := s.forfeit(s.p1); err != nil {
			t.Fatal(err)
		}
		_, err := s.forfeit(s.p2)
		conflict(t, err, "GAME_COMPLETED")
		_, err = s.play(s.p2, false)
		conflict(t, err, "GAME_COMPLETED")
		_, err = s.svc.ExpireTurn(context.Background(), sudokugame.ExpireTurnRequest{GameID: s.gameID})
		conflict(t, err, "GAME_COMPLETED")
		if len(s.games.updates) != 1 {
			t.Fatalf("written exactly once: %d", len(s.games.updates))
		}
	})

	t.Run("refused with the game attached: not online, lobby", func(t *testing.T) {
		s := startedGame(t, sameDeviceVersus) // same device: Online is false
		_, err := s.forfeit(s.p1)
		c := conflict(t, err, "NOT_ONLINE")
		if c.Game.ID != s.gameID || c.Game.Status != "in_progress" {
			t.Fatalf("%+v", c.Game)
		}

		lobby := startedGame(t, sudokugame.CreateSessionRequest{Mode: "versus", DifficultyRaw: "easy", IsOnline: true})
		_, err = lobby.forfeit(lobby.p1)
		conflict(t, err, "GAME_NOT_STARTED")

		single := startedGame(t, sudokugame.CreateSessionRequest{Mode: "single", DifficultyRaw: "easy"})
		_, err = single.forfeit(single.p1)
		conflict(t, err, "NOT_ONLINE")

		for _, x := range []*started{s, lobby, single} {
			if len(x.games.updates) != 0 || x.tx.begun != 0 {
				t.Fatal("a refused forfeit must not write")
			}
		}
	})

	t.Run("401 before anything else; 404 for an unknown game", func(t *testing.T) {
		s := startedGame(t, sameDeviceVersus) // not online, so a valid token would get NOT_ONLINE
		for _, token := range []string{"", "wrong"} {
			_, err := s.forfeit(token)
			assertCode(t, err, 401, "INVALID_TOKEN")
		}
		_, err := s.svc.Forfeit(context.Background(), sudokugame.ForfeitRequest{GameID: "nope", Token: s.p1})
		assertCode(t, err, 404, "GAME_NOT_FOUND")
	})

	t.Run("works even when the turn is overdue", func(t *testing.T) {
		s := startedOnline(t)
		freezeNow(t, s.turnFrom.Add(time.Minute))
		update, err := s.forfeit(s.p1)
		if err != nil || *update.Game.WinnerID != "p2" {
			t.Fatalf("%v", err)
		}
	})

	t.Run("a lost race is retried from a fresh read", func(t *testing.T) {
		s := startedOnline(t)
		s.games.updateQueue = []error{errNoRows, nil}
		if _, err := s.forfeit(s.p1); err != nil {
			t.Fatal(err)
		}
		if len(s.games.updates) != 1 {
			t.Fatalf("saved once: %d", len(s.games.updates))
		}
	})

	t.Run("if the rival forfeited first, the retry sees a finished game", func(t *testing.T) {
		s := startedOnline(t)
		s.games.updateQueue = []error{errNoRows}
		s.games.onUpdate = func(stored *sudokugame.Game) { stored.Status = sudokugame.StatusCompleted; stored.Version = 2 }
		_, err := s.forfeit(s.p1)
		conflict(t, err, "GAME_COMPLETED")
	})
}
