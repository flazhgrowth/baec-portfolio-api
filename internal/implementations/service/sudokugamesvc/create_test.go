package sudokugamesvc

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokugame"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokumove"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokuplayer"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/entity"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/apierrors"
)

type fakeTx struct {
	begun, finished int
	lastErr         error
}

func (tx *fakeTx) Begin(ctx context.Context) (context.Context, error) {
	tx.begun++
	return ctx, nil
}

func (tx *fakeTx) Finish(_ context.Context, err *error) {
	tx.finished++
	tx.lastErr = *err
}

type fakeGameRepo struct {
	collisions int // how many inserts fail with ErrJoinCodeTaken before one succeeds
	inserted   []*sudokugame.Game

	lobby     *sudokugame.Game // returned by Get when the filter asks for its code while waiting
	updateErr error
	// updateQueue scripts the next Update results, one per call; nil means success.
	updateQueue []error
	// applyUpdates makes Update change the stored game, as a database would.
	applyUpdates bool
	updates      []sudokugame.GameUpdateFields
	filters      []sudokugame.GameFilter
	gets         int // reads by id
	// onUpdate lets a test change the stored game as a side effect, to simulate a concurrent writer.
	onUpdate func(*sudokugame.Game)
}

func (repo *fakeGameRepo) Get(_ context.Context, filter sudokugame.GameFilter) (*sudokugame.Game, error) {
	if filter.ID.Valid {
		if repo.lobby == nil || filter.ID.V != repo.lobby.ID {
			return nil, sql.ErrNoRows
		}
		game := *repo.lobby
		repo.gets++
		return &game, nil
	}
	if repo.lobby == nil || !filter.JoinCode.Valid || filter.JoinCode.V != repo.lobby.JoinCode.String ||
		!filter.Status.Valid || filter.Status.V != repo.lobby.Status {
		return nil, sql.ErrNoRows
	}
	lobby := *repo.lobby
	return &lobby, nil
}

func (repo *fakeGameRepo) Update(_ context.Context, fields sudokugame.GameUpdateFields, filter sudokugame.GameFilter) error {
	if len(repo.updateQueue) > 0 {
		scripted := repo.updateQueue[0]
		repo.updateQueue = repo.updateQueue[1:]
		if scripted != nil {
			if repo.onUpdate != nil {
				repo.onUpdate(repo.lobby)
			}
			return scripted
		}
	} else if repo.updateErr != nil {
		if repo.onUpdate != nil {
			repo.onUpdate(repo.lobby)
		}
		return repo.updateErr
	}
	repo.updates = append(repo.updates, fields)
	repo.filters = append(repo.filters, filter)
	if repo.applyUpdates && repo.lobby != nil {
		applyFields(repo.lobby, fields)
	}
	return nil
}

// applyFields does to a stored game what GameUpdateFields.UpdateSetQuery does in SQL.
func applyFields(game *sudokugame.Game, f sudokugame.GameUpdateFields) {
	if f.Status.Valid {
		game.Status = f.Status.String
	}
	if f.Board.Valid {
		game.Board = f.Board.String
	}
	if f.ClearTurn {
		game.TurnPlayerSeat, game.TurnStartedAt, game.TurnDeadlineAt = sql.NullString{}, sql.NullTime{}, sql.NullTime{}
	}
	if f.TurnPlayerSeat.Valid {
		game.TurnPlayerSeat, game.TurnStartedAt, game.TurnDeadlineAt = f.TurnPlayerSeat, f.TurnStartedAt, f.TurnDeadlineAt
	}
	if f.CompletedAt.Valid {
		game.CompletedAt, game.EndReason, game.WinnerSeat = f.CompletedAt, f.EndReason, f.WinnerSeat
	}
	if f.IncrementVersion {
		game.Version++
	}
}

func (repo *fakeGameRepo) Insert(_ context.Context, datum *sudokugame.Game) error {
	if repo.collisions > 0 {
		repo.collisions--
		return sudokugame.ErrJoinCodeTaken
	}
	repo.inserted = append(repo.inserted, datum)
	return nil
}

type fakePlayerRepo struct {
	err       error
	inserted  []sudokuplayer.Player
	seated    sudokuplayer.Players // returned by Find
	updates   map[string]sudokuplayer.PlayerUpdateFields
	updateErr error
}

func (repo *fakePlayerRepo) Update(_ context.Context, fields sudokuplayer.PlayerUpdateFields, filter sudokuplayer.PlayerFilter) error {
	if repo.updateErr != nil {
		return repo.updateErr
	}
	if repo.updates == nil {
		repo.updates = map[string]sudokuplayer.PlayerUpdateFields{}
	}
	repo.updates[filter.Seat.V] = fields
	return nil
}

type fakeMoveRepo struct {
	err   error
	moves []sudokumove.Move
}

func (repo *fakeMoveRepo) Insert(_ context.Context, datum *sudokumove.Move) error {
	if repo.err != nil {
		return repo.err
	}
	repo.moves = append(repo.moves, *datum)
	return nil
}

func (repo *fakePlayerRepo) Get(_ context.Context, filter sudokuplayer.PlayerFilter) (*sudokuplayer.Player, error) {
	for _, p := range repo.seated {
		if filter.GameID.Valid && p.GameID != filter.GameID.V {
			continue
		}
		if filter.TokenHash.Valid && p.TokenHash == filter.TokenHash.V {
			found := p
			return &found, nil
		}
	}
	return nil, sql.ErrNoRows
}

// Find hands out a copy, like a database does: the service mutates what it reads.
func (repo *fakePlayerRepo) Find(context.Context, sudokuplayer.PlayerFilter) (sudokuplayer.Players, error) {
	return append(sudokuplayer.Players(nil), repo.seated...), nil
}

func (repo *fakePlayerRepo) Insert(_ context.Context, datum *sudokuplayer.Player) error {
	if repo.err != nil {
		return repo.err
	}
	repo.inserted = append(repo.inserted, *datum)
	return nil
}

var caller = entity.AccountInfo{ID: "acc-1", Username: "Alex"}

func newService() (*service, *fakeTx, *fakeGameRepo, *fakePlayerRepo) {
	tx, games, players := &fakeTx{}, &fakeGameRepo{}, &fakePlayerRepo{}
	return &service{tx: tx, gameRepo: games, playerRepo: players, moveRepo: &fakeMoveRepo{}, hub: newHub()}, tx, games, players
}

func TestCreateSession(t *testing.T) {
	t.Run("single has one seat, no turn", func(t *testing.T) {
		svc, _, games, players := newService()
		resp, err := svc.CreateSession(context.Background(), caller, sudokugame.CreateSessionRequest{Mode: "single", DifficultyRaw: "easy"})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Game.Status != sudokugame.StatusInProgress || resp.Game.CurrentTurn != nil || resp.Game.JoinCode != nil {
			t.Fatalf("unexpected game: %+v", resp.Game)
		}
		if len(players.inserted) != 1 || len(resp.Credentials) != 1 || resp.Game.Version != 1 {
			t.Fatalf("want 1 seat and version 1, got %d seats, version %d", len(players.inserted), resp.Game.Version)
		}
		if games.inserted[0].CreatedBy != caller.ID || players.inserted[0].UserID.String != caller.ID {
			t.Fatal("seat and game must be attributed to the caller")
		}
	})

	t.Run("same-device versus seats a guest and starts p1's turn", func(t *testing.T) {
		svc, _, _, players := newService()
		resp, err := svc.CreateSession(context.Background(), caller, sudokugame.CreateSessionRequest{
			Mode: "versus", DifficultyRaw: "hard", PlayerNames: []string{"ignored", "Sam"},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(players.inserted) != 2 || len(resp.Credentials) != 2 {
			t.Fatalf("want 2 seats, got %d", len(players.inserted))
		}
		if players.inserted[0].Name != "Alex" || players.inserted[1].Name != "Sam" || players.inserted[1].UserID.Valid {
			t.Fatalf("seat 1 must use the account name, seat 2 the guest name with no account: %+v", players.inserted)
		}
		if resp.Game.CurrentTurn == nil || resp.Game.CurrentTurn.PlayerID != "p1" {
			t.Fatalf("p1 should have the turn: %+v", resp.Game.CurrentTurn)
		}
		if got := resp.Game.CurrentTurn.DeadlineAt.Sub(resp.Game.CurrentTurn.StartedAt).Milliseconds(); got != 10_000 {
			t.Fatalf("turn length = %dms", got)
		}
		if resp.Credentials[0].Token == resp.Credentials[1].Token {
			t.Fatal("seats must get distinct tokens")
		}
		if players.inserted[0].TokenHash != sudokuplayer.HashToken(resp.Credentials[0].Token) {
			t.Fatal("only the token hash may be stored")
		}
	})

	t.Run("online opens a lobby with a join code", func(t *testing.T) {
		svc, _, _, players := newService()
		resp, err := svc.CreateSession(context.Background(), caller, sudokugame.CreateSessionRequest{Mode: "versus", DifficultyRaw: "easy", IsOnline: true})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Game.Status != sudokugame.StatusWaiting || resp.Game.CurrentTurn != nil || len(players.inserted) != 1 {
			t.Fatalf("unexpected lobby: %+v", resp.Game)
		}
		if resp.Game.JoinCode == nil || len(*resp.Game.JoinCode) != joinCodeLength {
			t.Fatalf("bad join code: %v", resp.Game.JoinCode)
		}
		for _, char := range *resp.Game.JoinCode {
			if !containsRune(joinCodeAlphabet, char) {
				t.Fatalf("join code char %q outside alphabet", char)
			}
		}
	})

	t.Run("retries on join code collision", func(t *testing.T) {
		svc, tx, games, _ := newService()
		games.collisions = 2
		if _, err := svc.CreateSession(context.Background(), caller, sudokugame.CreateSessionRequest{Mode: "versus", DifficultyRaw: "easy", IsOnline: true}); err != nil {
			t.Fatal(err)
		}
		if len(games.inserted) != 1 || tx.begun != 3 || tx.finished != 3 {
			t.Fatalf("want 3 transactions and 1 game, got %d tx, %d games", tx.begun, len(games.inserted))
		}
	})

	t.Run("gives up after repeated collisions", func(t *testing.T) {
		svc, _, games, _ := newService()
		games.collisions = 100
		_, err := svc.CreateSession(context.Background(), caller, sudokugame.CreateSessionRequest{Mode: "versus", DifficultyRaw: "easy", IsOnline: true})
		assertStatus(t, err, 500)
	})

	t.Run("rolls back when a seat fails to insert", func(t *testing.T) {
		svc, tx, _, players := newService()
		players.err = errors.New("db down")
		_, err := svc.CreateSession(context.Background(), caller, sudokugame.CreateSessionRequest{Mode: "single", DifficultyRaw: "easy"})
		assertStatus(t, err, 500)
		if tx.finished != 1 || tx.lastErr == nil {
			t.Fatal("transaction must finish with the error so it rolls back")
		}
	})

	t.Run("rejects invalid requests before touching the db", func(t *testing.T) {
		for name, args := range map[string]sudokugame.CreateSessionRequest{
			"bad mode":       {Mode: "coop", DifficultyRaw: "easy"},
			"bad difficulty": {Mode: "single", DifficultyRaw: "insane"},
			"online single":  {Mode: "single", DifficultyRaw: "easy", IsOnline: true},
		} {
			svc, tx, _, _ := newService()
			_, err := svc.CreateSession(context.Background(), caller, args)
			assertStatus(t, err, 422)
			if tx.begun != 0 {
				t.Fatalf("%s: opened a transaction for an invalid request", name)
			}
		}
	})
}

func TestGuestName(t *testing.T) {
	for name, tc := range map[string]struct {
		names []string
		want  string
	}{
		"none":      {nil, "Player 2"},
		"only host": {[]string{"Alex"}, "Player 2"},
		"blank":     {[]string{"a", "  "}, "Player 2"},
		"trimmed":   {[]string{"a", "  Sam "}, "Sam"},
		"truncated": {[]string{"a", "abcdefghijklmnopqrstuvwxyz"}, "abcdefghijklmnopqrst"},
		"unicode":   {[]string{"a", "ÄÖÜÄÖÜÄÖÜÄÖÜÄÖÜÄÖÜÄÖÜÄÖÜÄÖÜÄÖÜ"}, "ÄÖÜÄÖÜÄÖÜÄÖÜÄÖÜÄÖÜÄÖ"},
	} {
		args := sudokugame.CreateSessionRequest{PlayerNames: tc.names}
		if got := args.GuestName(); got != tc.want {
			t.Errorf("%s: got %q, want %q", name, got, tc.want)
		}
	}
}

func assertStatus(t *testing.T, err error, want int) {
	t.Helper()
	var httpErr apierrors.HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != want {
		t.Fatalf("want HTTP %d, got %v", want, err)
	}
}

func containsRune(set string, r rune) bool {
	for _, c := range set {
		if c == r {
			return true
		}
	}
	return false
}
