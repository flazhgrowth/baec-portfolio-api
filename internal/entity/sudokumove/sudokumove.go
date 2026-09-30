package sudokumove

import (
	"context"
	"database/sql"

	"github.com/Masterminds/squirrel"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/table"
)

var (
	MoveTable table.Table = table.Table{
		Name:          "sudoku_moves",
		InsertColumns: []string{"game_id", "seat", "row", "col", "value", "correct", "points", "elapsed_ms", "game_version"},
	}
)

type (
	// Move is one accepted move, correct or not. The log is append-only.
	Move struct {
		GameID      string        `db:"game_id"`
		Seat        string        `db:"seat"`
		Row         int           `db:"row"`
		Col         int           `db:"col"`
		Value       int           `db:"value"`
		Correct     bool          `db:"correct"`
		Points      int           `db:"points"`
		ElapsedMs   sql.NullInt64 `db:"elapsed_ms"` // null in single mode
		GameVersion int           `db:"game_version"`
	}

	Repository interface {
		Insert(ctx context.Context, datum *Move) (err error)
	}
)

func (datum *Move) InsertValuesQuery(builder squirrel.InsertBuilder) squirrel.InsertBuilder {
	return builder.
		Columns(MoveTable.InsertColumns...).
		Values(
			datum.GameID,
			datum.Seat,
			datum.Row,
			datum.Col,
			datum.Value,
			datum.Correct,
			datum.Points,
			datum.ElapsedMs,
			datum.GameVersion,
		)
}
