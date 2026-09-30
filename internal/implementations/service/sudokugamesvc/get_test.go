package sudokugamesvc

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokugame"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokuplayer"
)

// freezeNow pins the service clock for the test.
func freezeNow(t *testing.T, at time.Time) {
	t.Helper()
	prev := now
	now = func() time.Time { return at }
	t.Cleanup(func() { now = prev })
}

// storedGame creates a game through the service and returns it as the store would hold it.
func storedGame(t *testing.T, req sudokugame.CreateSessionRequest) (*service, *fakeGameRepo, *fakePlayerRepo) {
	t.Helper()
	svc, _, games, players := newService()
	if _, err := svc.CreateSession(context.Background(), caller, req); err != nil {
		t.Fatal(err)
	}
	games.lobby = games.inserted[0]
	games.lobby.CreatedAt = now()
	games.lobby.Version = 1
	players.seated = append(sudokuplayer.Players(nil), players.inserted...)
	return svc, games, players
}

var sameDeviceVersus = sudokugame.CreateSessionRequest{Mode: "versus", DifficultyRaw: "easy"}

func TestGetGame(t *testing.T) {
	t.Run("returns the state without the solution or any token", func(t *testing.T) {
		svc, games, _ := storedGame(t, sameDeviceVersus)
		resp, err := svc.GetGame(context.Background(), sudokugame.GetGameRequest{ID: games.lobby.ID})
		if err != nil {
			t.Fatal(err)
		}
		if resp.ID != games.lobby.ID || len(resp.Players) != 2 || resp.CurrentTurn == nil || resp.Version != 1 {
			t.Fatalf("unexpected game: %+v", resp)
		}

		body, _ := json.Marshal(resp)
		if strings.Contains(string(body), games.lobby.Solution) || strings.Contains(string(body), "solution") {
			t.Fatal("the solution must never be serialised")
		}
		for _, p := range svc.playerRepo.(*fakePlayerRepo).seated {
			if strings.Contains(string(body), p.TokenHash) || strings.Contains(string(body), "token") {
				t.Fatal("tokens must never be serialised")
			}
		}
		if len(games.updates) != 0 {
			t.Fatal("a turn that is not due must not be written")
		}
	})

	t.Run("unknown id is GAME_NOT_FOUND", func(t *testing.T) {
		svc, _, _ := storedGame(t, sameDeviceVersus)
		_, err := svc.GetGame(context.Background(), sudokugame.GetGameRequest{ID: "nope"})
		assertCode(t, err, 404, "GAME_NOT_FOUND")
	})

	t.Run("an expired lobby is gone", func(t *testing.T) {
		svc, games, _ := storedGame(t, sudokugame.CreateSessionRequest{Mode: "versus", DifficultyRaw: "easy", IsOnline: true})
		games.lobby.CreatedAt = now().Add(-sudokugame.LobbyTTL - time.Minute)
		_, err := svc.GetGame(context.Background(), sudokugame.GetGameRequest{ID: games.lobby.ID})
		assertCode(t, err, 404, "GAME_NOT_FOUND")
	})

	t.Run("a live lobby is returned with its join code", func(t *testing.T) {
		svc, games, _ := storedGame(t, sudokugame.CreateSessionRequest{Mode: "versus", DifficultyRaw: "easy", IsOnline: true})
		resp, err := svc.GetGame(context.Background(), sudokugame.GetGameRequest{ID: games.lobby.ID})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Status != sudokugame.StatusWaiting || resp.JoinCode == nil || resp.CurrentTurn != nil {
			t.Fatalf("unexpected lobby: %+v", resp)
		}
	})

	t.Run("a single game never expires a turn", func(t *testing.T) {
		svc, games, _ := storedGame(t, sudokugame.CreateSessionRequest{Mode: "single", DifficultyRaw: "easy"})
		freezeNow(t, now().Add(24*time.Hour))
		resp, err := svc.GetGame(context.Background(), sudokugame.GetGameRequest{ID: games.lobby.ID})
		if err != nil {
			t.Fatal(err)
		}
		if resp.CurrentTurn != nil || len(games.updates) != 0 {
			t.Fatal("single mode has no turn to expire")
		}
	})
}

func TestGetGameAppliesDueTurn(t *testing.T) {
	t.Run("hands the turn over and starts the next one now", func(t *testing.T) {
		svc, games, _ := storedGame(t, sameDeviceVersus)
		late := games.lobby.TurnDeadlineAt.Time.Add(3 * time.Second)
		freezeNow(t, late)

		resp, err := svc.GetGame(context.Background(), sudokugame.GetGameRequest{ID: games.lobby.ID})
		if err != nil {
			t.Fatal(err)
		}
		turn := resp.CurrentTurn
		if turn.PlayerID != "p2" || !turn.StartedAt.Equal(late) || turn.DeadlineAt.Sub(turn.StartedAt) != 10*time.Second {
			t.Fatalf("next turn must go to p2 and start at processing time, not the old deadline: %+v", turn)
		}
		if resp.Version != 2 || resp.ServerTime != late {
			t.Fatalf("version=%d serverTime=%s", resp.Version, resp.ServerTime)
		}

		u, f := games.updates[0], games.filters[0]
		if !u.IncrementVersion || u.TurnPlayerSeat.String != "p2" {
			t.Fatalf("bad update: %+v", u)
		}
		if !f.Version.Valid || f.Version.V != 1 {
			t.Fatalf("update must be guarded by the version it read: %+v", f)
		}
	})

	t.Run("the deadline itself counts as due", func(t *testing.T) {
		svc, games, _ := storedGame(t, sameDeviceVersus)
		freezeNow(t, games.lobby.TurnDeadlineAt.Time)
		resp, err := svc.GetGame(context.Background(), sudokugame.GetGameRequest{ID: games.lobby.ID})
		if err != nil || resp.CurrentTurn.PlayerID != "p2" {
			t.Fatalf("err=%v turn=%+v", err, resp.CurrentTurn)
		}
	})

	t.Run("just before the deadline nothing changes", func(t *testing.T) {
		svc, games, _ := storedGame(t, sameDeviceVersus)
		freezeNow(t, games.lobby.TurnDeadlineAt.Time.Add(-time.Millisecond))
		resp, err := svc.GetGame(context.Background(), sudokugame.GetGameRequest{ID: games.lobby.ID})
		if err != nil || resp.CurrentTurn.PlayerID != "p1" || len(games.updates) != 0 {
			t.Fatalf("err=%v turn=%+v updates=%d", err, resp.CurrentTurn, len(games.updates))
		}
	})

	t.Run("p2's expired turn goes back to p1", func(t *testing.T) {
		svc, games, _ := storedGame(t, sameDeviceVersus)
		games.lobby.TurnPlayerSeat = sql.NullString{Valid: true, String: "p2"}
		freezeNow(t, games.lobby.TurnDeadlineAt.Time.Add(time.Second))
		resp, err := svc.GetGame(context.Background(), sudokugame.GetGameRequest{ID: games.lobby.ID})
		if err != nil || resp.CurrentTurn.PlayerID != "p1" {
			t.Fatalf("err=%v turn=%+v", err, resp.CurrentTurn)
		}
	})

	t.Run("losing the race serves the winner's state instead of applying twice", func(t *testing.T) {
		svc, games, _ := storedGame(t, sameDeviceVersus)
		late := games.lobby.TurnDeadlineAt.Time.Add(time.Second)
		freezeNow(t, late)

		// Another request expires the turn between our read and our guarded update.
		games.updateErr = sql.ErrNoRows
		games.onUpdate = func(stored *sudokugame.Game) {
			stored.TurnPlayerSeat = sql.NullString{Valid: true, String: "p2"}
			stored.TurnStartedAt = sql.NullTime{Valid: true, Time: late}
			stored.TurnDeadlineAt = sql.NullTime{Valid: true, Time: late.Add(10 * time.Second)}
			stored.Version = 2
		}

		resp, err := svc.GetGame(context.Background(), sudokugame.GetGameRequest{ID: games.lobby.ID})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Version != 2 || resp.CurrentTurn.PlayerID != "p2" || games.gets != 2 {
			t.Fatalf("want the other writer's state after one re-read: version=%d turn=%+v reads=%d", resp.Version, resp.CurrentTurn, games.gets)
		}
	})
}
