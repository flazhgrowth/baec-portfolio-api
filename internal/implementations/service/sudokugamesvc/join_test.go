package sudokugamesvc

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokugame"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokuplayer"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/entity"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/apierrors"
)

var guestCaller = entity.AccountInfo{ID: "acc-2", Username: "Sam"}

// waitingLobby returns a service whose store holds one open lobby, hosted by Alex.
func waitingLobby(t *testing.T) (*service, *fakeTx, *fakeGameRepo, *fakePlayerRepo) {
	svc, tx, games, players, _ := waitingLobbyWithHostToken(t)
	return svc, tx, games, players
}

// waitingLobbyWithHostToken also returns the host's raw player token.
func waitingLobbyWithHostToken(t *testing.T) (*service, *fakeTx, *fakeGameRepo, *fakePlayerRepo, string) {
	t.Helper()
	svc, tx, games, players := newService()
	created, err := svc.CreateSession(context.Background(), caller, sudokugame.CreateSessionRequest{Mode: "versus", DifficultyRaw: "hard", IsOnline: true})
	if err != nil {
		t.Fatal(err)
	}

	games.lobby = games.inserted[0]
	games.lobby.CreatedAt = now() // the database sets this; the fake does not
	players.seated = sudokuplayer.Players{players.inserted[0]}
	if created.Game.JoinCode == nil || *created.Game.JoinCode != games.lobby.JoinCode.String {
		t.Fatal("setup: lobby join code mismatch")
	}

	// reset the counters the setup call moved
	*tx = fakeTx{}
	players.inserted = nil
	return svc, tx, games, players, created.Credentials[0].Token
}

func TestJoinSession(t *testing.T) {
	t.Run("seats the guest, starts the game, hands p1 the turn", func(t *testing.T) {
		svc, tx, games, players := waitingLobby(t)
		code := games.lobby.JoinCode.String

		resp, err := svc.JoinSession(context.Background(), guestCaller, sudokugame.JoinSessionRequest{Code: "  " + lower(code) + " "})
		if err != nil {
			t.Fatalf("code must be case-insensitive and trimmed: %v", err)
		}

		g := resp.Game
		if g.Status != sudokugame.StatusInProgress || g.JoinCode != nil || g.Version != 2 {
			t.Fatalf("unexpected game: status=%s joinCode=%v version=%d", g.Status, g.JoinCode, g.Version)
		}
		if g.CurrentTurn == nil || g.CurrentTurn.PlayerID != "p1" || g.CurrentTurn.DeadlineAt.Sub(g.CurrentTurn.StartedAt) != 10*time.Second {
			t.Fatalf("p1 should get a 10s turn: %+v", g.CurrentTurn)
		}
		if len(g.Players) != 2 || g.Players[0].Name != "Alex" || g.Players[1].Name != "Sam" || g.Players[1].UserID == nil || *g.Players[1].UserID != guestCaller.ID {
			t.Fatalf("players: %+v", g.Players)
		}
		if len(resp.Credentials) != 1 || resp.Credentials[0].PlayerID != "p2" {
			t.Fatalf("guest must get only p2's credential: %+v", resp.Credentials)
		}
		if len(players.inserted) != 1 || players.inserted[0].TokenHash != sudokuplayer.HashToken(resp.Credentials[0].Token) {
			t.Fatal("guest seat must be stored with the token's hash only")
		}

		u := games.updates[0]
		if u.Status.String != sudokugame.StatusInProgress || !u.ClearJoinCode || !u.IncrementVersion || !u.TurnDeadlineAt.Valid {
			t.Fatalf("update must start the game and free the code: %+v", u)
		}
		if tx.begun != 1 || tx.finished != 1 || tx.lastErr != nil {
			t.Fatalf("want one committed transaction, got begun=%d finished=%d err=%v", tx.begun, tx.finished, tx.lastErr)
		}
	})

	t.Run("unknown code is JOIN_CODE_NOT_FOUND", func(t *testing.T) {
		svc, tx, _, _ := waitingLobby(t)
		_, err := svc.JoinSession(context.Background(), guestCaller, sudokugame.JoinSessionRequest{Code: "ZZZZZZ"})
		assertCode(t, err, 404, "JOIN_CODE_NOT_FOUND")
		if tx.begun != 0 {
			t.Fatal("must not open a transaction for an unknown code")
		}
	})

	t.Run("a lobby that already started looks like an unknown code", func(t *testing.T) {
		svc, _, games, _ := waitingLobby(t)
		code := games.lobby.JoinCode.String
		games.lobby.Status = sudokugame.StatusInProgress
		_, err := svc.JoinSession(context.Background(), guestCaller, sudokugame.JoinSessionRequest{Code: code})
		assertCode(t, err, 404, "JOIN_CODE_NOT_FOUND")
	})

	t.Run("an expired lobby is gone", func(t *testing.T) {
		svc, _, games, _ := waitingLobby(t)
		games.lobby.CreatedAt = now().Add(-sudokugame.LobbyTTL - time.Minute)
		_, err := svc.JoinSession(context.Background(), guestCaller, sudokugame.JoinSessionRequest{Code: games.lobby.JoinCode.String})
		assertCode(t, err, 404, "JOIN_CODE_NOT_FOUND")
	})

	t.Run("losing the race is GAME_FULL and rolls back", func(t *testing.T) {
		svc, tx, games, players := waitingLobby(t)
		games.updateErr = sql.ErrNoRows // another guest claimed the lobby between our read and our update
		_, err := svc.JoinSession(context.Background(), guestCaller, sudokugame.JoinSessionRequest{Code: games.lobby.JoinCode.String})
		assertCode(t, err, 409, "GAME_FULL")
		if len(players.inserted) != 0 || tx.lastErr == nil {
			t.Fatal("loser must not seat a guest, and the transaction must roll back")
		}
	})

	t.Run("seat insert failure rolls back the start", func(t *testing.T) {
		svc, tx, games, players := waitingLobby(t)
		players.err = errors.New("db down")
		_, err := svc.JoinSession(context.Background(), guestCaller, sudokugame.JoinSessionRequest{Code: games.lobby.JoinCode.String})
		assertCode(t, err, 500, "")
		if tx.lastErr == nil {
			t.Fatal("transaction must finish with the error")
		}
	})

	t.Run("malformed code is VALIDATION_ERROR", func(t *testing.T) {
		for _, code := range []string{"", "   ", "ABC", "ABCDEFG"} {
			svc, tx, _, _ := waitingLobby(t)
			_, err := svc.JoinSession(context.Background(), guestCaller, sudokugame.JoinSessionRequest{Code: code})
			assertCode(t, err, 422, "VALIDATION_ERROR")
			if tx.begun != 0 {
				t.Fatalf("code %q opened a transaction", code)
			}
		}
	})
}

func assertCode(t *testing.T, err error, status int, code string) {
	t.Helper()
	var httpErr apierrors.HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != status || (code != "" && httpErr.Code != code) {
		t.Fatalf("want HTTP %d %q, got %v", status, code, err)
	}
}

func lower(s string) string {
	out := []byte(s)
	for i, c := range out {
		if c >= 'A' && c <= 'Z' {
			out[i] = c + 32
		}
	}
	return string(out)
}
