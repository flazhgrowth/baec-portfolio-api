package sudokugame

import (
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudoku"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokuplayer"
)

const (
	ModeSingle = "single"
	ModeVersus = "versus"

	StatusWaiting    = "waiting"
	StatusInProgress = "in_progress"
	StatusCompleted  = "completed"

	DefaultGuestName = "Player 2"

	EndReasonSolved  = "solved"
	EndReasonForfeit = "forfeit"

	JoinCodeLength = 6
	// LobbyTTL is how long an unjoined online lobby stays joinable.
	LobbyTTL      = 30 * time.Minute
	maxNameLength = 20
)

type (
	// { "mode": "versus", "difficulty": "hard", "online": true, "player_names": ["Alex", "Sam"] }
	CreateSessionRequest struct {
		Mode          string            `json:"mode"`
		DifficultyRaw string            `json:"difficulty"`
		Difficulty    sudoku.Difficulty `json:"-"`
		IsOnline      bool              `json:"online"`
		PlayerNames   []string          `json:"player_names"`
	}

	// { "code": "ABC234" }
	JoinSessionRequest struct {
		Code string `json:"code"`
	}

	// POST /games/{gameId}/moves, token from the X-Player-Token header. Row, Col
	// and Value are floats only so a missing or fractional number can be told
	// apart from a valid one and answered with a validation error.
	MoveRequest struct {
		GameID string   `path:"gameId" pathtype:"string" json:"-"`
		Token  string   `json:"-"`
		Row    *float64 `json:"row"`
		Col    *float64 `json:"col"`
		Value  *float64 `json:"value"`
	}
	MoveResponse struct {
		Result    string       `json:"result"`
		Points    int          `json:"points"`
		ElapsedMs *int         `json:"elapsed_ms"`
		Game      GameResponse `json:"game"`
		Events    []GameEvent  `json:"events"`
	}

	// ConflictError is a 409 that carries the game as it now stands, so the
	// client can resync without another call.
	ConflictError struct {
		Code    string
		Message string
		Game    GameResponse
	}

	// POST /games/{gameId}/turn/expire. No token: it is a fallback for the
	// server's own timer, and it only acts once the deadline has passed.
	ExpireTurnRequest struct {
		GameID string `path:"gameId" pathtype:"string" json:"-"`
	}

	// POST /games/{gameId}/forfeit, token from the X-Player-Token header.
	ForfeitRequest struct {
		GameID string `path:"gameId" pathtype:"string" json:"-"`
		Token  string `json:"-"`
	}

	GetGameRequest struct {
		ID string `path:"gameId" pathtype:"string" json:"-"`
	}

	// Rules are the constants a game is played by. They are snapshotted on the
	// game row so changing the defaults never alters a game in flight.
	Rules struct {
		TurnLimitMs           int `json:"turn_limit_ms"`
		MaxPoints             int `json:"max_points"`
		MinPoints             int `json:"min_points"`
		FaultLimit            int `json:"fault_limit"`
		SkipTurnsOnFaultLimit int `json:"skip_turns_on_fault_limit"`
		DisconnectForfeitMs   int `json:"disconnect_forfeit_ms"`
	}

	TurnResponse struct {
		PlayerID   string    `json:"player_id"`
		StartedAt  time.Time `json:"started_at"`
		DeadlineAt time.Time `json:"deadline_at"`
	}

	GameResponse struct {
		ID          string                        `json:"id"`
		Mode        string                        `json:"mode"`
		Online      bool                          `json:"online"`
		Difficulty  string                        `json:"difficulty"`
		Status      string                        `json:"status"`
		JoinCode    *string                       `json:"join_code"`
		Puzzle      [][]int                       `json:"puzzle"`
		Board       [][]int                       `json:"board"`
		Players     []sudokuplayer.PlayerResponse `json:"players"`
		CurrentTurn *TurnResponse                 `json:"current_turn"`
		Rules       Rules                         `json:"rules"`
		StartedAt   time.Time                     `json:"started_at"`
		CompletedAt *time.Time                    `json:"completed_at"`
		EndReason   *string                       `json:"end_reason"`
		WinnerID    *string                       `json:"winner_id"`
		Version     int                           `json:"version"`
		ServerTime  time.Time                     `json:"server_time"`
	}

	// SessionResponse is what create/join return: the game plus one token per
	// seat the caller controls. The solution is never part of it.
	SessionResponse struct {
		Game        GameResponse                      `json:"game"`
		Credentials []sudokuplayer.CredentialResponse `json:"credentials"`
	}
)

func DefaultRules() Rules {
	return Rules{
		TurnLimitMs:           10_000,
		MaxPoints:             10,
		MinPoints:             1,
		FaultLimit:            3,
		SkipTurnsOnFaultLimit: 2,
		DisconnectForfeitMs:   60_000,
	}
}

// Normalize resolves the difficulty slug. An unknown slug leaves Difficulty
// zero-valued, which Validate rejects.
func (args *CreateSessionRequest) Normalize() *CreateSessionRequest {
	switch args.DifficultyRaw {
	case sudoku.DIFF_EASY.Slug:
		args.Difficulty = sudoku.DIFF_EASY
	case sudoku.DIFF_MEDIUM.Slug:
		args.Difficulty = sudoku.DIFF_MEDIUM
	case sudoku.DIFF_HARD.Slug:
		args.Difficulty = sudoku.DIFF_HARD
	}

	return args
}

// Validate returns a human-readable reason, or "" when the request is valid.
func (args *CreateSessionRequest) Validate() string {
	if args.Mode != ModeSingle && args.Mode != ModeVersus {
		return "mode must be 'single' or 'versus'"
	}
	if args.Difficulty.Slug == "" {
		return "difficulty must be 'easy', 'medium' or 'hard'"
	}
	if args.IsOnline && args.Mode != ModeVersus {
		return "online is only valid with mode 'versus'"
	}

	return ""
}

// GuestName is the name of the second seat of a same-device versus game.
// Index 0 is the caller and is ignored: that seat is named after the account.
func (args *CreateSessionRequest) GuestName() string {
	if len(args.PlayerNames) < 2 {
		return DefaultGuestName
	}

	name := strings.TrimSpace(args.PlayerNames[1])
	if name == "" {
		return DefaultGuestName
	}
	if utf8.RuneCountInString(name) > maxNameLength {
		name = string([]rune(name)[:maxNameLength])
	}

	return name
}

// Normalize makes the code case-insensitive and ignores stray whitespace.
func (args *JoinSessionRequest) Normalize() *JoinSessionRequest {
	args.Code = strings.ToUpper(strings.TrimSpace(args.Code))

	return args
}

// Validate returns a human-readable reason, or "" when the request is valid.
func (args *JoinSessionRequest) Validate() string {
	if len(args.Code) != JoinCodeLength {
		return "code must be 6 characters"
	}

	return ""
}

// TurnDue reports whether the running turn's deadline has passed at the given time.
// Only a started versus game has a turn, so anything else is never due.
func (datum *Game) TurnDue(at time.Time) bool {
	return datum.Status == StatusInProgress &&
		datum.TurnDeadlineAt.Valid &&
		!at.Before(datum.TurnDeadlineAt.Time)
}

// LobbyExpired reports whether an unjoined online lobby has outlived LobbyTTL.
func (datum *Game) LobbyExpired(at time.Time) bool {
	return datum.Status == StatusWaiting && at.Sub(datum.CreatedAt) > LobbyTTL
}

func (err *ConflictError) Error() string { return "[" + err.Code + "] " + err.Message }

// Ints returns the move's coordinates and value, or a reason if any is missing or not a whole number.
func (args *MoveRequest) Ints() (row, col, value int, reason string) {
	toInt := func(n *float64) (int, bool) {
		if n == nil || *n != math.Trunc(*n) || math.Abs(*n) > 1e6 {
			return 0, false
		}

		return int(*n), true
	}

	row, okRow := toInt(args.Row)
	col, okCol := toInt(args.Col)
	if !okRow || !okCol {
		return 0, 0, 0, "row and col must be integers between 0 and 8"
	}
	value, okValue := toInt(args.Value)
	if !okValue {
		return 0, 0, 0, "value must be an integer between 1 and 9"
	}

	return row, col, value, ""
}
