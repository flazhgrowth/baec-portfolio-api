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

func next(t *testing.T, sub *sudokugame.Subscription) sudokugame.GameUpdate {
	t.Helper()
	select {
	case u, open := <-sub.Updates:
		if !open {
			t.Fatal("stream closed")
		}
		return u
	case <-time.After(time.Second):
		t.Fatal("no update arrived")
	}
	return sudokugame.GameUpdate{}
}

func quiet(t *testing.T, sub *sudokugame.Subscription) {
	t.Helper()
	select {
	case u := <-sub.Updates:
		t.Fatalf("unexpected update: %+v", u.Events)
	case <-time.After(30 * time.Millisecond):
	}
}

func TestSubscribe(t *testing.T) {
	t.Run("sends the full state with no events, then live updates", func(t *testing.T) {
		svc, _, games, _, hostToken := waitingLobbyWithHostToken(t)
		sub, err := svc.Subscribe(context.Background(), sudokugame.StreamEventsRequest{GameID: games.lobby.ID, Token: hostToken})
		if err != nil {
			t.Fatal(err)
		}
		defer sub.Close()

		if sub.Initial.Game.ID != games.lobby.ID || sub.Initial.Game.Status != sudokugame.StatusWaiting {
			t.Fatalf("unexpected snapshot: %+v", sub.Initial.Game)
		}
		body, _ := json.Marshal(sub.Initial)
		if !strings.Contains(string(body), `"events":[]`) {
			t.Fatalf("initial events must serialise as [], got %s", body)
		}
		if strings.Contains(string(body), "solution") || strings.Contains(string(body), hostToken) {
			t.Fatal("neither the solution nor a token may be streamed")
		}

		svc.hub.publish(games.lobby.ID, update(9))
		if next(t, sub).Game.Version != 9 {
			t.Fatal("live update not delivered")
		}
	})

	t.Run("rejects a bad token with INVALID_TOKEN and leaves nothing subscribed", func(t *testing.T) {
		svc, _, games, _, _ := waitingLobbyWithHostToken(t)
		for _, token := range []string{"", "wrong"} {
			_, err := svc.Subscribe(context.Background(), sudokugame.StreamEventsRequest{GameID: games.lobby.ID, Token: token})
			assertCode(t, err, 401, "INVALID_TOKEN")
		}
		if svc.hub.count(games.lobby.ID) != 0 {
			t.Fatal("a rejected caller must not stay subscribed")
		}
	})

	t.Run("a token from another game is invalid here", func(t *testing.T) {
		svc, _, games, players, hostToken := waitingLobbyWithHostToken(t)
		players.seated[0].GameID = "some-other-game" // the token belongs to a different game
		_, err := svc.Subscribe(context.Background(), sudokugame.StreamEventsRequest{GameID: games.lobby.ID, Token: hostToken})
		assertCode(t, err, 401, "INVALID_TOKEN")
	})

	t.Run("unknown game is GAME_NOT_FOUND even with a bad token", func(t *testing.T) {
		svc, _, _, _, _ := waitingLobbyWithHostToken(t)
		_, err := svc.Subscribe(context.Background(), sudokugame.StreamEventsRequest{GameID: "nope", Token: "whatever"})
		assertCode(t, err, 404, "GAME_NOT_FOUND")
	})
}

func TestEventsArePublished(t *testing.T) {
	t.Run("join tells the host, with the guest seated", func(t *testing.T) {
		svc, _, games, _, hostToken := waitingLobbyWithHostToken(t)
		host, err := svc.Subscribe(context.Background(), sudokugame.StreamEventsRequest{GameID: games.lobby.ID, Token: hostToken})
		if err != nil {
			t.Fatal(err)
		}
		defer host.Close()

		if _, err = svc.JoinSession(context.Background(), guestCaller, sudokugame.JoinSessionRequest{Code: games.lobby.JoinCode.String}); err != nil {
			t.Fatal(err)
		}

		got := next(t, host)
		if len(got.Events) != 1 || got.Events[0].Type != sudokugame.EventPlayerJoined || got.Events[0].PlayerID != "p2" {
			t.Fatalf("want player_joined p2, got %+v", got.Events)
		}
		if got.Game.Status != sudokugame.StatusInProgress || len(got.Game.Players) != 2 || got.Game.Version != 2 {
			t.Fatalf("update must carry the started game: %+v", got.Game)
		}
		body, _ := json.Marshal(got)
		if strings.Contains(string(body), "token") {
			t.Fatal("the update must not carry any token")
		}
	})

	t.Run("a join that loses the race announces nothing", func(t *testing.T) {
		svc, _, games, _, hostToken := waitingLobbyWithHostToken(t)
		host, _ := svc.Subscribe(context.Background(), sudokugame.StreamEventsRequest{GameID: games.lobby.ID, Token: hostToken})
		defer host.Close()

		games.updateErr = errNoRows
		_, err := svc.JoinSession(context.Background(), guestCaller, sudokugame.JoinSessionRequest{Code: games.lobby.JoinCode.String})
		assertCode(t, err, 409, "GAME_FULL")
		quiet(t, host)
	})

	t.Run("a turn expiry noticed by a GET reaches the open streams", func(t *testing.T) {
		svc, games, players := storedGame(t, sameDeviceVersus)
		// hand p1 a known token so it can subscribe
		players.seated[0].TokenHash = hashOf("p1-token")
		sub, err := svc.Subscribe(context.Background(), sudokugame.StreamEventsRequest{GameID: games.lobby.ID, Token: "p1-token"})
		if err != nil {
			t.Fatal(err)
		}
		defer sub.Close()

		freezeNow(t, games.lobby.TurnDeadlineAt.Time.Add(time.Second))
		if _, err = svc.GetGame(context.Background(), sudokugame.GetGameRequest{ID: games.lobby.ID}); err != nil {
			t.Fatal(err)
		}

		got := next(t, sub)
		if len(got.Events) != 1 || got.Events[0].Type != sudokugame.EventTurnExpired || got.Events[0].PlayerID != "p1" {
			t.Fatalf("want turn_expired for p1, got %+v", got.Events)
		}
		if got.Game.CurrentTurn.PlayerID != "p2" || got.Game.Version != 2 {
			t.Fatalf("update must carry the new turn: %+v", got.Game.CurrentTurn)
		}
	})

	t.Run("a GET with nothing due publishes nothing", func(t *testing.T) {
		svc, games, players := storedGame(t, sameDeviceVersus)
		players.seated[0].TokenHash = hashOf("p1-token")
		sub, _ := svc.Subscribe(context.Background(), sudokugame.StreamEventsRequest{GameID: games.lobby.ID, Token: "p1-token"})
		defer sub.Close()

		if _, err := svc.GetGame(context.Background(), sudokugame.GetGameRequest{ID: games.lobby.ID}); err != nil {
			t.Fatal(err)
		}
		quiet(t, sub)
	})
}

var errNoRows = sql.ErrNoRows

func hashOf(token string) string { return sudokuplayer.HashToken(token) }
