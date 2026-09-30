package sudokugame

import (
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudoku"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokuplayer"
)

var t0 = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

func ms(n int) time.Time { return t0.Add(time.Duration(n) * time.Millisecond) }

// newMatch builds a started game: "single", or "versus" with both seats and p1 to move at t0.
func newMatch(t *testing.T, mode string) *Match {
	t.Helper()
	board := sudoku.NewBoard()
	board.Generate()
	puzzle := board.ToPuzzle()
	puzzle.Generate(sudoku.DIFF_EASY)

	rules, _ := json.Marshal(DefaultRules())
	game := &Game{
		Mode: mode, Status: StatusInProgress, Difficulty: "easy",
		Puzzle: puzzle.Puzzle.String(), Board: puzzle.Puzzle.String(), Solution: puzzle.Solution.String(),
		Rules: string(rules), StartedAt: t0, Version: 1,
	}
	players := sudokuplayer.Players{{Seat: "p1", Name: "A"}}
	if mode == ModeVersus {
		players = append(players, sudokuplayer.Player{Seat: "p2", Name: "B"})
	}
	match, err := NewMatch(game, players)
	if err != nil {
		t.Fatal(err)
	}
	if mode == ModeVersus {
		match.startTurn("p1", t0)
	}
	return match
}

// target is the first empty cell with its right value and a value that is wrong.
func (m *Match) target(t *testing.T) (row, col, right, wrong int) {
	t.Helper()
	for i := range 81 {
		if m.Game.Board[i] == '0' {
			right = int(m.Game.Solution[i] - '0')
			wrong = right%9 + 1
			return i / 9, i % 9, right, wrong
		}
	}
	t.Fatal("no empty cell")
	return
}

// play submits the next empty cell, right or wrong, as seat at the given time.
func (m *Match) play(t *testing.T, seat string, at time.Time, wrong bool) (MoveOutcome, []GameEvent, error) {
	t.Helper()
	row, col, right, bad := m.target(t)
	value := right
	if wrong {
		value = bad
	}
	return m.SubmitMove(seat, row, col, value, at)
}

func mustPlay(t *testing.T, m *Match, seat string, at time.Time, wrong bool) (MoveOutcome, []GameEvent) {
	t.Helper()
	out, events, err := m.play(t, seat, at, wrong)
	if err != nil {
		t.Fatalf("%s at %s: %v", seat, at.Sub(t0), err)
	}
	return out, events
}

func turn(m *Match) string { return m.Game.TurnPlayerSeat.String }

func ruleCode(err error) string {
	var ruleErr *RuleError
	if errors.As(err, &ruleErr) {
		return ruleErr.Code
	}
	return "no rule error"
}

func count(events []GameEvent, kind string) (n int) {
	for _, e := range events {
		if e.Type == kind {
			n++
		}
	}
	return
}

func TestScoreForElapsed(t *testing.T) {
	rules := DefaultRules()
	for elapsed, want := range map[int]int{0: 10, 999: 10, 1000: 9, 5500: 5, 9999: 1, 10_999: 1, 60_000: 1, -5: 10} {
		if got := rules.ScoreForElapsed(elapsed); got != want {
			t.Errorf("ScoreForElapsed(%d) = %d, want %d", elapsed, got, want)
		}
	}
}

func TestVersusTurns(t *testing.T) {
	t.Run("a correct fill is scored by elapsed time and passes the turn", func(t *testing.T) {
		m := newMatch(t, ModeVersus)
		row, col, right, _ := m.target(t)
		out, events, err := m.SubmitMove("p1", row, col, right, ms(2500))
		if err != nil {
			t.Fatal(err)
		}
		if out.Result() != ResultCorrect || out.Points != 8 || *out.ElapsedMs != 2500 {
			t.Fatalf("outcome: %+v", out)
		}
		mv := events[0]
		if mv.Type != EventMove || mv.PlayerID != "p1" || *mv.Points != 8 || *mv.Row != row || *mv.Col != col || *mv.Value != right || mv.Result != ResultCorrect {
			t.Fatalf("move event: %+v", mv)
		}
		if m.Players[0].Score != 8 || m.Game.Board[row*9+col] != byte('0'+right) {
			t.Fatal("score and board must update")
		}
		if turn(m) != "p2" || !m.Game.TurnStartedAt.Time.Equal(ms(2500)) || m.Game.TurnDeadlineAt.Time.Sub(ms(2500)) != 10*time.Second {
			t.Fatalf("the next turn must start at the move time: %s", turn(m))
		}
	})

	t.Run("rejects the wrong seat and an occupied cell", func(t *testing.T) {
		m := newMatch(t, ModeVersus)
		if _, events, err := m.play(t, "p2", ms(100), false); ruleCode(err) != "NOT_YOUR_TURN" || len(events) != 0 {
			t.Fatalf("got %v, events %v", err, events)
		}
		row, col, right, _ := m.target(t)
		mustPlay(t, m, "p1", ms(100), false)
		if _, _, err := m.SubmitMove("p2", row, col, right, ms(200)); ruleCode(err) != "CELL_NOT_EMPTY" {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("a late move expires the turn, still rejects the move, and reports the expiry", func(t *testing.T) {
		m := newMatch(t, ModeVersus)
		row, col, right, _ := m.target(t)
		_, events, err := m.SubmitMove("p1", row, col, right, ms(10_000))
		if ruleCode(err) != "TURN_EXPIRED" {
			t.Fatalf("got %v", err)
		}
		if len(events) != 1 || events[0].Type != EventTurnExpired || events[0].PlayerID != "p1" {
			t.Fatalf("the expiry must come back with the error so it can be saved: %+v", events)
		}
		if turn(m) != "p2" || m.Game.Version != 2 {
			t.Fatalf("turn=%s version=%d", turn(m), m.Game.Version)
		}
		if m.Game.Board[row*9+col] != '0' || m.Players[0].Faults != 0 {
			t.Fatal("a late move is no move: no fill, no fault")
		}
	})

	t.Run("a late move by the other player is also TURN_EXPIRED", func(t *testing.T) {
		m := newMatch(t, ModeVersus)
		if _, _, err := m.play(t, "p2", ms(15_000), false); ruleCode(err) != "TURN_EXPIRED" {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("ApplyDue is a no-op until the deadline", func(t *testing.T) {
		m := newMatch(t, ModeVersus)
		if events := m.ApplyDue(ms(9999)); events != nil || m.Game.Version != 1 {
			t.Fatalf("events=%v version=%d", events, m.Game.Version)
		}
		events := m.ApplyDue(ms(10_000))
		if len(events) != 1 || events[0].PlayerID != "p1" || turn(m) != "p2" {
			t.Fatalf("events=%v turn=%s", events, turn(m))
		}
	})

	t.Run("version bumps once per state change", func(t *testing.T) {
		m := newMatch(t, ModeVersus)
		m.ApplyDue(ms(1000))
		if m.Game.Version != 1 {
			t.Fatal("nothing due must not bump")
		}
		mustPlay(t, m, "p1", ms(1500), false)
		if m.Game.Version != 2 {
			t.Fatalf("after a move: %d", m.Game.Version)
		}
		m.ApplyDue(ms(11_500))
		if m.Game.Version != 3 {
			t.Fatalf("after an expiry: %d", m.Game.Version)
		}
	})
}

func TestFaults(t *testing.T) {
	t.Run("a wrong fill scores nothing, adds a fault, is not written, ends the turn", func(t *testing.T) {
		m := newMatch(t, ModeVersus)
		row, col, _, _ := m.target(t)
		out, _ := mustPlay(t, m, "p1", ms(500), true)
		if out.Result() != ResultIncorrect || out.Points != 0 || m.Game.Board[row*9+col] != '0' {
			t.Fatalf("outcome %+v", out)
		}
		if p := m.Players[0]; p.Faults != 1 || p.Mistakes != 1 || p.Score != 0 {
			t.Fatalf("player: %+v", p)
		}
		if turn(m) != "p2" {
			t.Fatal("the turn must pass")
		}
	})

	t.Run("three faults cost two turns, so the opponent plays three in a row", func(t *testing.T) {
		m := newMatch(t, ModeVersus)
		at := 0
		play := func(seat string, wrong bool) []GameEvent {
			at += 100
			_, events := mustPlay(t, m, seat, ms(at), wrong)
			return events
		}

		play("p1", true)
		play("p2", false)
		play("p1", true)
		play("p2", false)
		third := play("p1", true)
		var limit *GameEvent
		for i := range third {
			if third[i].Type == EventFaultLimitReached {
				limit = &third[i]
			}
		}
		if limit == nil || limit.PlayerID != "p1" || *limit.SkipTurns != 2 {
			t.Fatalf("fault_limit_reached missing: %+v", third)
		}
		if p := m.Players[0]; p.Faults != 3 || p.SkipTurnsRemaining != 2 || turn(m) != "p2" {
			t.Fatalf("p1=%+v turn=%s", p, turn(m))
		}

		if ev := play("p2", false); count(ev, EventTurnSkipped) != 1 || turn(m) != "p2" || m.Players[0].SkipTurnsRemaining != 1 || m.Players[0].Faults != 3 {
			t.Fatalf("first skip: %+v p1=%+v turn=%s", ev, m.Players[0], turn(m))
		}
		if ev := play("p2", false); count(ev, EventTurnSkipped) != 1 || turn(m) != "p2" || m.Players[0].SkipTurnsRemaining != 0 || m.Players[0].Faults != 0 {
			t.Fatalf("second skip must reset faults: %+v p1=%+v turn=%s", ev, m.Players[0], turn(m))
		}
		play("p2", false)
		if turn(m) != "p1" {
			t.Fatalf("p1 should be back, turn=%s", turn(m))
		}
	})

	t.Run("skips are also consumed when a turn simply times out", func(t *testing.T) {
		m := newMatch(t, ModeVersus)
		m.Players[1].SkipTurnsRemaining = 1 // p2 owes a turn
		events := m.ApplyDue(ms(10_000))    // p1 times out; p2 would be next
		if count(events, EventTurnSkipped) != 1 || turn(m) != "p1" || m.Players[1].SkipTurnsRemaining != 0 {
			t.Fatalf("events=%+v turn=%s p2=%+v", events, turn(m), m.Players[1])
		}
	})
}

func TestCompletion(t *testing.T) {
	// almostDone fills every empty cell but one and returns that cell's coordinates.
	almostDone := func(t *testing.T, mode string) (*Match, int, int, int) {
		m := newMatch(t, mode)
		board := []byte(m.Game.Board)
		last := -1
		for i := range 81 {
			if board[i] == '0' {
				if last >= 0 {
					board[last] = m.Game.Solution[last]
				}
				last = i
			}
		}
		m.Game.Board = string(board)
		return m, last / 9, last % 9, int(m.Game.Solution[last] - '0')
	}

	t.Run("versus: the higher score wins", func(t *testing.T) {
		m, row, col, value := almostDone(t, ModeVersus)
		m.Players[0].Score, m.Players[1].Score = 12, 7
		_, events, err := m.SubmitMove("p1", row, col, value, ms(1000))
		if err != nil {
			t.Fatal(err)
		}
		if count(events, EventGameCompleted) != 1 || events[len(events)-1].Type != EventGameCompleted {
			t.Fatalf("game_completed must close the events: %+v", events)
		}
		g := m.Game
		if g.Status != StatusCompleted || g.EndReason.String != "solved" || g.WinnerSeat.String != "p1" || g.TurnPlayerSeat.Valid || !g.CompletedAt.Valid {
			t.Fatalf("game: %+v", g)
		}
	})

	t.Run("a draw has no winner", func(t *testing.T) {
		m, row, col, value := almostDone(t, ModeVersus)
		m.Players[1].Score = 10 // p1 earns 10 on the last fill
		if _, _, err := m.SubmitMove("p1", row, col, value, ms(500)); err != nil {
			t.Fatal(err)
		}
		if m.Game.WinnerSeat.Valid || m.Game.Status != StatusCompleted {
			t.Fatalf("%+v", m.Game)
		}
	})

	t.Run("a wrong last fill does not complete the game", func(t *testing.T) {
		m, row, col, value := almostDone(t, ModeVersus)
		if _, _, err := m.SubmitMove("p1", row, col, value%9+1, ms(500)); err != nil {
			t.Fatal(err)
		}
		if m.Game.Status != StatusInProgress {
			t.Fatal("still in progress")
		}
	})

	t.Run("single has no turns, points or timeout, and only counts mistakes", func(t *testing.T) {
		m := newMatch(t, ModeSingle)
		if m.Game.TurnPlayerSeat.Valid {
			t.Fatal("single has no turn")
		}
		out, _ := mustPlay(t, m, "p1", ms(3_600_000), false)
		if out.Result() != ResultCorrect || out.Points != 0 || out.ElapsedMs != nil {
			t.Fatalf("%+v", out)
		}
		mustPlay(t, m, "p1", ms(3_600_001), true)
		if p := m.Players[0]; p.Mistakes != 1 || p.SkipTurnsRemaining != 0 {
			t.Fatalf("%+v", p)
		}
		if m.ApplyDue(ms(99_999_999)) != nil {
			t.Fatal("single never times out")
		}
	})

	t.Run("single: solving it ends the game with no winner", func(t *testing.T) {
		m, row, col, value := almostDone(t, ModeSingle)
		if _, _, err := m.SubmitMove("p1", row, col, value, ms(1)); err != nil {
			t.Fatal(err)
		}
		if m.Game.Status != StatusCompleted || m.Game.WinnerSeat.Valid {
			t.Fatalf("%+v", m.Game)
		}
	})
}

func TestSubmitMoveGuards(t *testing.T) {
	t.Run("validation", func(t *testing.T) {
		m := newMatch(t, ModeVersus)
		for name, c := range map[string][3]int{"row high": {9, 0, 1}, "row low": {-1, 0, 1}, "col high": {0, 9, 1}, "value 0": {0, 0, 0}, "value 10": {0, 0, 10}} {
			if _, events, err := m.SubmitMove("p1", c[0], c[1], c[2], ms(1)); ruleCode(err) != "VALIDATION_ERROR" || len(events) != 0 {
				t.Errorf("%s: %v", name, err)
			}
		}
	})

	t.Run("a lobby and a finished game accept no moves", func(t *testing.T) {
		m := newMatch(t, ModeVersus)
		m.Game.Status = StatusWaiting
		m.Game.TurnPlayerSeat = sql.NullString{}
		if _, _, err := m.play(t, "p1", ms(1), false); ruleCode(err) != "GAME_NOT_STARTED" {
			t.Fatalf("%v", err)
		}
		m.Game.Status = StatusCompleted
		if _, _, err := m.play(t, "p1", ms(1), false); ruleCode(err) != "GAME_COMPLETED" {
			t.Fatalf("%v", err)
		}
	})

	t.Run("conflicts carry HTTP 409, validation 422", func(t *testing.T) {
		m := newMatch(t, ModeVersus)
		var ruleErr *RuleError
		_, _, err := m.play(t, "p2", ms(1), false)
		if !errors.As(err, &ruleErr) || ruleErr.Status != 409 {
			t.Fatalf("%v", err)
		}
		_, _, err = m.SubmitMove("p1", 9, 9, 9, ms(1))
		if !errors.As(err, &ruleErr) || ruleErr.Status != 422 {
			t.Fatalf("%v", err)
		}
	})
}

func TestForfeit(t *testing.T) {
	online := func(t *testing.T) *Match {
		m := newMatch(t, ModeVersus)
		m.Game.Online = true
		return m
	}

	t.Run("the caller loses, the opponent wins, the game ends", func(t *testing.T) {
		for seat, winner := range map[string]string{"p1": "p2", "p2": "p1"} {
			m := online(t)
			events, err := m.Forfeit(seat, ms(2000))
			if err != nil {
				t.Fatal(err)
			}
			if len(events) != 2 || events[0].Type != EventPlayerForfeited || events[0].PlayerID != seat || events[1].Type != EventGameCompleted {
				t.Fatalf("%s: events %+v", seat, events)
			}
			g := m.Game
			if g.Status != StatusCompleted || g.EndReason.String != EndReasonForfeit || g.WinnerSeat.String != winner || g.TurnPlayerSeat.Valid || !g.CompletedAt.Time.Equal(ms(2000)) || g.Version != 2 {
				t.Fatalf("%s: game %+v", seat, g)
			}
		}
	})

	t.Run("it does not have to be your turn, and the score does not matter", func(t *testing.T) {
		m := online(t)
		m.Players[0].Score = 50 // p1 is far ahead and on turn, but p2 forfeits
		if _, err := m.Forfeit("p2", ms(1)); err != nil || m.Game.WinnerSeat.String != "p1" {
			t.Fatalf("%v %+v", err, m.Game.WinnerSeat)
		}
		m = online(t)
		m.Players[1].Score = 50 // p2 leads but p1 forfeits: the score is irrelevant
		if _, err := m.Forfeit("p1", ms(1)); err != nil || m.Game.WinnerSeat.String != "p2" {
			t.Fatalf("%v %+v", err, m.Game.WinnerSeat)
		}
	})

	t.Run("even with the turn overdue", func(t *testing.T) {
		m := online(t)
		if _, err := m.Forfeit("p1", ms(60_000)); err != nil || m.Game.Status != StatusCompleted {
			t.Fatalf("%v", err)
		}
	})

	t.Run("refused: not online, lobby, finished; nothing changes", func(t *testing.T) {
		cases := map[string]struct {
			setup func(*Match)
			want  string
		}{
			"same-device": {func(m *Match) { m.Game.Online = false }, "NOT_ONLINE"},
			"lobby":       {func(m *Match) { m.Game.Status = StatusWaiting }, "GAME_NOT_STARTED"},
			"finished":    {func(m *Match) { m.Game.Status = StatusCompleted }, "GAME_COMPLETED"},
		}
		for name, c := range cases {
			m := online(t)
			c.setup(m)
			version, status := m.Game.Version, m.Game.Status
			events, err := m.Forfeit("p1", ms(1))
			if ruleCode(err) != c.want || events != nil || m.Game.Version != version || m.Game.Status != status {
				t.Errorf("%s: err=%v events=%v", name, err, events)
			}
		}
		single := newMatch(t, ModeSingle)
		if _, err := single.Forfeit("p1", ms(1)); ruleCode(err) != "NOT_ONLINE" {
			t.Fatalf("single: %v", err)
		}
	})

	t.Run("a solved game keeps end_reason solved", func(t *testing.T) {
		m := online(t)
		board := []byte(m.Game.Solution)
		board[0] = '0'
		m.Game.Board = string(board)
		if _, _, err := m.SubmitMove("p1", 0, 0, int(m.Game.Solution[0]-'0'), ms(500)); err != nil {
			t.Fatal(err)
		}
		if m.Game.EndReason.String != EndReasonSolved {
			t.Fatalf("got %q", m.Game.EndReason.String)
		}
	})
}
