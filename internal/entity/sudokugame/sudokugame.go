package sudokugame

import (
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
	maxNameLength    = 20
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
