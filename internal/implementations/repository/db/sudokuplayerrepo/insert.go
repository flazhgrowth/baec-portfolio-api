package sudokuplayerrepo

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokuplayer"
)

// Insert uses Write rather than Insert: the table's key is (game_id, seat), and
// the writer's Insert appends `RETURNING id`, which this table has no column for.
func (repo *repository) Insert(ctx context.Context, datum *sudokuplayer.Player) (err error) {
	logpath := baselogpath.With("Insert")

	builder := squirrel.
		Insert(sudokuplayer.PlayerTable.Name).
		PlaceholderFormat(squirrel.Dollar)
	builder = datum.InsertValuesQuery(builder)
	query, args, err := builder.ToSql()
	if err != nil {
		logpath.With("builder.ToSql").LogError(ctx, "failed to build query", err)
		return err
	}

	if _, err = repo.actuator.Writer().Write(ctx, query, args); err != nil {
		logpath.With("Write").LogError(ctx, "failed query run", err)
		return err
	}

	return nil
}
