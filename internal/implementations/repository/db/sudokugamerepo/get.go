package sudokugamerepo

import (
	"context"
	"database/sql"

	"github.com/Masterminds/squirrel"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokugame"
	"github.com/flazhgrowth/fg-tamagopkg/logger"
)

func (repo *repository) Get(ctx context.Context, filter sudokugame.GameFilter) (datum *sudokugame.Game, err error) {
	logpath := baselogpath.With("Get")

	builder := squirrel.
		Select(sudokugame.GameTable.SelectColumns...).
		From(sudokugame.GameTable.Name).
		PlaceholderFormat(squirrel.Dollar)
	builder = filter.ConditionQuery(builder)

	query, args, err := builder.ToSql()
	if err != nil {
		logpath.With("builder.ToSql").LogError(ctx, "failed to build query", err)
		return nil, err
	}

	logpath.LogDebug(ctx, "executing query..", logger.LogMeta{"query": query, "args": args})
	datum = &sudokugame.Game{}
	if err = repo.actuator.Reader().Get(ctx, query, args, datum); err != nil {
		if err != sql.ErrNoRows {
			logpath.With("Get").LogError(ctx, "failed query run", err)
		}
		return nil, err
	}

	return datum, nil
}
