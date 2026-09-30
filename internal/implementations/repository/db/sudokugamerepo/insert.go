package sudokugamerepo

import (
	"context"
	"errors"

	"github.com/Masterminds/squirrel"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokugame"
	"github.com/lib/pq"
)

const (
	pqUniqueViolation = "23505"
	joinCodeIndex     = "sudoku_games_join_code_uq"
)

func (repo *repository) Insert(ctx context.Context, datum *sudokugame.Game) (err error) {
	logpath := baselogpath.With("Insert")

	builder := squirrel.
		Insert(sudokugame.GameTable.Name).
		PlaceholderFormat(squirrel.Dollar)
	builder = datum.InsertValuesQuery(builder)
	query, args, err := builder.ToSql()
	if err != nil {
		logpath.With("builder.ToSql").LogError(ctx, "failed to build query", err)
		return err
	}

	if err = repo.actuator.Writer().Insert(ctx, query, args, &datum.ID); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == pqUniqueViolation && pqErr.Constraint == joinCodeIndex {
			return sudokugame.ErrJoinCodeTaken
		}

		logpath.With("Insert").LogError(ctx, "failed query run", err)
		return err
	}

	return nil
}
