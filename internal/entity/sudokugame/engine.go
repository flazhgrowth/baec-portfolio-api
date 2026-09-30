package sudokugame

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokuplayer"
)

// RuleError is a rule violation: the request was understood but the game's
// state or rules forbid it. Status is the HTTP status the API answers with.
type RuleError struct {
	Status  int
	Code    string
	Message string
}

func (err *RuleError) Error() string { return fmt.Sprintf("[%s] %s", err.Code, err.Message) }

func ruleConflict(code, message string) *RuleError {
	return &RuleError{Status: http.StatusConflict, Code: code, Message: message}
}

func ruleInvalid(message string) *RuleError {
	return &RuleError{Status: http.StatusUnprocessableEntity, Code: "VALIDATION_ERROR", Message: message}
}

// Match applies the game rules to one game and its seats, in memory. It changes
// Game and Players in place and bumps Game.Version once per change; persisting
// that is the caller's job. Time is always passed in, so it is deterministic.
//
// Players must be in seat order (p1 first): that is turn order.
type Match struct {
	Game    *Game
	Players sudokuplayer.Players
	Rules   Rules
}

func NewMatch(game *Game, players sudokuplayer.Players) (*Match, error) {
	match := &Match{Game: game, Players: players}
	if err := json.Unmarshal([]byte(game.Rules), &match.Rules); err != nil {
		return nil, fmt.Errorf("decode rules: %w", err)
	}

	return match, nil
}

// MoveOutcome is the result of an accepted move (correct or not).
type MoveOutcome struct {
	Seat      string
	Row, Col  int
	Value     int
	Correct   bool
	Points    int
	ElapsedMs *int // nil in single mode, which has no turns
}

// Result is the API's "correct" / "incorrect".
func (out MoveOutcome) Result() string {
	if out.Correct {
		return ResultCorrect
	}

	return ResultIncorrect
}

// ScoreForElapsed is 10 points in the first second, one fewer each further
// second, never below the minimum.
func (rules Rules) ScoreForElapsed(elapsedMs int) int {
	raw := rules.MaxPoints - max(0, elapsedMs)/1000

	return max(rules.MinPoints, min(rules.MaxPoints, raw))
}

// ApplyDue applies whatever is due at the given time. Today that is the turn
// timeout; the contract also puts presence changes and disconnect forfeits here,
// which wait for presence tracking. It returns the events it produced.
func (m *Match) ApplyDue(at time.Time) []GameEvent {
	if !m.Game.TurnDue(at) {
		return nil
	}

	events := []GameEvent{{Type: EventTurnExpired, PlayerID: m.Game.TurnPlayerSeat.String}}
	m.advanceTurn(at, &events)
	m.Game.Version++

	return events
}

// Forfeit ends the game with seat as the loser: the opponent wins and end_reason
// is "forfeit". Only a running online game can be forfeited (a lobby that nobody
// joined just expires). Unlike a move, the caller need not be on turn, and any
// pending timeout is moot because the game is over either way.
func (m *Match) Forfeit(seat string, at time.Time) (events []GameEvent, err error) {
	game := m.Game

	switch {
	case !game.Online:
		return nil, ruleConflict("NOT_ONLINE", "Only online games can be forfeited")
	case game.Status == StatusWaiting:
		return nil, ruleConflict("GAME_NOT_STARTED", "Waiting for a second player")
	case game.Status == StatusCompleted:
		return nil, ruleConflict("GAME_COMPLETED", "Game is already completed")
	}

	events = []GameEvent{{Type: EventPlayerForfeited, PlayerID: seat}}
	m.finish(at, &events, EndReasonForfeit, sql.NullString{Valid: true, String: sudokuplayer.OtherSeat(seat)})
	game.Version++

	return events, nil
}

// SubmitMove plays one move for seat, in the order the contract checks things:
// input, game state, then anything already due, then whose turn it is, then the cell.
//
// It returns the events that changed state even when it also returns an error:
// a move that arrives after the deadline still expires the turn, and that has to
// be saved and announced before the move is rejected. An error with no events
// means nothing changed.
func (m *Match) SubmitMove(seat string, row, col, value int, at time.Time) (outcome MoveOutcome, events []GameEvent, err error) {
	game := m.Game

	if row < 0 || row > 8 || col < 0 || col > 8 {
		return outcome, nil, ruleInvalid("row and col must be integers between 0 and 8")
	}
	if value < 1 || value > 9 {
		return outcome, nil, ruleInvalid("value must be an integer between 1 and 9")
	}
	switch game.Status {
	case StatusWaiting:
		return outcome, nil, ruleConflict("GAME_NOT_STARTED", "Waiting for a second player")
	case StatusCompleted:
		return outcome, nil, ruleConflict("GAME_COMPLETED", "Game is already completed")
	}

	wasDue := game.TurnDue(at)
	events = m.ApplyDue(at)
	if wasDue {
		return outcome, events, ruleConflict("TURN_EXPIRED", "Your turn has expired")
	}
	if game.TurnPlayerSeat.Valid && game.TurnPlayerSeat.String != seat {
		return outcome, events, ruleConflict("NOT_YOUR_TURN", "It is not your turn")
	}

	cell := row*9 + col
	if game.Board[cell] != '0' {
		return outcome, events, ruleConflict("CELL_NOT_EMPTY", "Cell is already filled")
	}

	versus := game.Mode == ModeVersus
	player := m.player(seat)
	outcome = MoveOutcome{
		Seat: seat, Row: row, Col: col, Value: value,
		Correct: game.Solution[cell] == byte('0'+value),
	}
	if game.TurnStartedAt.Valid {
		elapsed := int(at.Sub(game.TurnStartedAt.Time).Milliseconds())
		outcome.ElapsedMs = &elapsed
	}

	var afterMove []GameEvent
	if outcome.Correct {
		board := []byte(game.Board)
		board[cell] = byte('0' + value)
		game.Board = string(board)
		if versus {
			outcome.Points = m.Rules.ScoreForElapsed(*outcome.ElapsedMs)
			player.Score += outcome.Points
		}
	} else {
		player.Faults++
		player.Mistakes++
		if versus && player.Faults >= m.Rules.FaultLimit {
			player.SkipTurnsRemaining = m.Rules.SkipTurnsOnFaultLimit
			skip := player.SkipTurnsRemaining
			afterMove = append(afterMove, GameEvent{Type: EventFaultLimitReached, PlayerID: seat, SkipTurns: &skip})
		}
	}

	points := outcome.Points
	events = append(events, GameEvent{
		Type: EventMove, PlayerID: seat, Row: &row, Col: &col, Value: &value,
		Result: outcome.Result(), Points: &points,
	})
	events = append(events, afterMove...)

	switch {
	case game.Board == game.Solution:
		m.finish(at, &events, EndReasonSolved, m.winnerByScore(versus))
	case versus:
		m.advanceTurn(at, &events)
	}

	game.Version++
	return outcome, events, nil
}

func (m *Match) player(seat string) *sudokuplayer.Player {
	for i := range m.Players {
		if m.Players[i].Seat == seat {
			return &m.Players[i]
		}
	}

	return nil
}

// winnerByScore is the higher score, or nobody on a draw. Single games have no winner.
func (m *Match) winnerByScore(versus bool) sql.NullString {
	if !versus || len(m.Players) < 2 {
		return sql.NullString{}
	}

	a, b := m.Players[0], m.Players[1]
	switch {
	case a.Score > b.Score:
		return sql.NullString{Valid: true, String: a.Seat}
	case b.Score > a.Score:
		return sql.NullString{Valid: true, String: b.Seat}
	}

	return sql.NullString{}
}

func (m *Match) startTurn(seat string, at time.Time) {
	deadline := at.Add(time.Duration(m.Rules.TurnLimitMs) * time.Millisecond)
	m.Game.TurnPlayerSeat = sql.NullString{Valid: true, String: seat}
	m.Game.TurnStartedAt = sql.NullTime{Valid: true, Time: at}
	m.Game.TurnDeadlineAt = sql.NullTime{Valid: true, Time: deadline}
}

// advanceTurn hands the turn to the other seat, consuming any skipped turns
// that seat owes. A seat's fault counter resets when its last skip is consumed.
func (m *Match) advanceTurn(at time.Time, events *[]GameEvent) {
	current := m.Game.TurnPlayerSeat.String
	next := m.player(sudokuplayer.OtherSeat(current))

	for next.SkipTurnsRemaining > 0 {
		next.SkipTurnsRemaining--
		if next.SkipTurnsRemaining == 0 {
			next.Faults = 0
		}
		*events = append(*events, GameEvent{Type: EventTurnSkipped, PlayerID: next.Seat})
		next = m.player(current)
	}

	m.startTurn(next.Seat, at)
}

func (m *Match) finish(at time.Time, events *[]GameEvent, reason string, winner sql.NullString) {
	game := m.Game
	game.Status = StatusCompleted
	game.CompletedAt = sql.NullTime{Valid: true, Time: at}
	game.EndReason = sql.NullString{Valid: true, String: reason}
	game.WinnerSeat = winner
	game.TurnPlayerSeat, game.TurnStartedAt, game.TurnDeadlineAt = sql.NullString{}, sql.NullTime{}, sql.NullTime{}
	*events = append(*events, GameEvent{Type: EventGameCompleted})
}
