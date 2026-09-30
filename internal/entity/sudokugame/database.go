package sudokugame

import (
	"database/sql"
	"errors"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/entity"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/table"
)

// ErrJoinCodeTaken is returned by Repository.Insert when the join code collides
// with another open lobby. The caller is expected to retry with a fresh code.
var ErrJoinCodeTaken = errors.New("join code already in use")

var (
	GameTable table.Table = table.Table{
		Name: "sudoku_games",
		SelectColumns: []string{
			"id", "mode", "online", "difficulty", "status", "join_code", "puzzle", "solution", "board", "rules",
			"turn_player_seat", "turn_started_at", "turn_deadline_at", "started_at", "completed_at",
			"end_reason", "winner_seat", "version", "created_by", "created_at", "updated_at",
		},
		InsertColumns: []string{
			"id", "mode", "online", "difficulty", "status", "join_code", "puzzle", "solution", "board", "rules",
			"turn_player_seat", "turn_started_at", "turn_deadline_at", "started_at", "created_by",
		},
	}
)

type (
	Game struct {
		entity.BaseULIDModel
		Mode           string         `db:"mode"`
		Online         bool           `db:"online"`
		Difficulty     string         `db:"difficulty"`
		Status         string         `db:"status"`
		JoinCode       sql.NullString `db:"join_code"`
		Puzzle         string         `db:"puzzle"`
		Solution       string         `db:"solution"`
		Board          string         `db:"board"`
		Rules          string         `db:"rules"` // JSON
		TurnPlayerSeat sql.NullString `db:"turn_player_seat"`
		TurnStartedAt  sql.NullTime   `db:"turn_started_at"`
		TurnDeadlineAt sql.NullTime   `db:"turn_deadline_at"`
		StartedAt      time.Time      `db:"started_at"`
		CompletedAt    sql.NullTime   `db:"completed_at"`
		EndReason      sql.NullString `db:"end_reason"`
		WinnerSeat     sql.NullString `db:"winner_seat"`
		Version        int            `db:"version"`
		CreatedBy      string         `db:"created_by"`
	}
	Games []Game
)

func (datum *Game) InsertValuesQuery(builder squirrel.InsertBuilder) squirrel.InsertBuilder {
	return builder.
		Columns(GameTable.InsertColumns...).
		Values(
			datum.ID,
			datum.Mode,
			datum.Online,
			datum.Difficulty,
			datum.Status,
			datum.JoinCode,
			datum.Puzzle,
			datum.Solution,
			datum.Board,
			datum.Rules,
			datum.TurnPlayerSeat,
			datum.TurnStartedAt,
			datum.TurnDeadlineAt,
			datum.StartedAt,
			datum.CreatedBy,
		)
}
