package sudokuplayer

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/table"
)

const (
	SeatOne = "p1"
	SeatTwo = "p2"
)

var (
	PlayerTable table.Table = table.Table{
		Name: "sudoku_game_players",
		SelectColumns: []string{
			"game_id", "seat", "user_id", "name", "token_hash", "score", "faults", "mistakes",
			"skip_turns_remaining", "connected", "last_seen_at", "forfeit_at",
		},
		InsertColumns: []string{
			"game_id", "seat", "user_id", "name", "token_hash", "score", "faults", "mistakes",
			"skip_turns_remaining", "connected", "last_seen_at",
		},
	}
)

type (
	// Player is one seat in one game. Seat is what the API calls Player.id.
	Player struct {
		GameID             string         `db:"game_id"`
		Seat               string         `db:"seat"`
		UserID             sql.NullString `db:"user_id"`
		Name               string         `db:"name"`
		TokenHash          string         `db:"token_hash"`
		Score              int            `db:"score"`
		Faults             int            `db:"faults"`
		Mistakes           int            `db:"mistakes"`
		SkipTurnsRemaining int            `db:"skip_turns_remaining"`
		Connected          bool           `db:"connected"`
		LastSeenAt         time.Time      `db:"last_seen_at"`
		ForfeitAt          sql.NullTime   `db:"forfeit_at"`
	}
	Players []Player

	PlayerResponse struct {
		ID                 string     `json:"id"`
		Name               string     `json:"name"`
		UserID             *string    `json:"user_id"`
		Score              int        `json:"score"`
		Faults             int        `json:"faults"`
		Mistakes           int        `json:"mistakes"`
		SkipTurnsRemaining int        `json:"skip_turns_remaining"`
		Connected          bool       `json:"connected"`
		ForfeitAt          *time.Time `json:"forfeit_at"`
	}
	CredentialResponse struct {
		PlayerID string `json:"player_id"`
		Token    string `json:"token"`
	}
)

func (datum *Player) InsertValuesQuery(builder squirrel.InsertBuilder) squirrel.InsertBuilder {
	return builder.
		Columns(PlayerTable.InsertColumns...).
		Values(
			datum.GameID,
			datum.Seat,
			datum.UserID,
			datum.Name,
			datum.TokenHash,
			datum.Score,
			datum.Faults,
			datum.Mistakes,
			datum.SkipTurnsRemaining,
			datum.Connected,
			datum.LastSeenAt,
		)
}

func (datum *Player) ToResponse() PlayerResponse {
	resp := PlayerResponse{
		ID:                 datum.Seat,
		Name:               datum.Name,
		Score:              datum.Score,
		Faults:             datum.Faults,
		Mistakes:           datum.Mistakes,
		SkipTurnsRemaining: datum.SkipTurnsRemaining,
		Connected:          datum.Connected,
	}
	if datum.UserID.Valid {
		resp.UserID = &datum.UserID.String
	}
	if datum.ForfeitAt.Valid {
		resp.ForfeitAt = &datum.ForfeitAt.Time
	}

	return resp
}

// NewToken returns a random player token (sent to the client once) and its
// SHA-256 hash (the only thing stored). The token has 256 bits of entropy, so a
// fast hash is enough; it is not a password.
func NewToken() (token, hash string, err error) {
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return "", "", err
	}

	token = base64.RawURLEncoding.EncodeToString(raw)
	return token, HashToken(token), nil
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
