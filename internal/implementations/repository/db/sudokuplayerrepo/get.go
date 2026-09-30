package sudokuplayerrepo

import (
	"context"
	"database/sql"

	"github.com/Masterminds/squirrel"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokuplayer"
	"github.com/flazhgrowth/fg-tamagopkg/logger"
)

// Find returns the matching seats ordered by seat (p1 before p2), which is turn order.
func (repo *repository) Find(ctx context.Context, filter sudokuplayer.PlayerFilter) (data sudokuplayer.Players, err error) {
	logpath := baselogpath.With("Find")

	builder := squirrel.
		Select(sudokuplayer.PlayerTable.SelectColumns...).
		From(sudokuplayer.PlayerTable.Name).
		PlaceholderFormat(squirrel.Dollar).
		OrderBy("seat ASC")
	builder = filter.ConditionQuery(builder)

	query, args, err := builder.ToSql()
	if err != nil {
		logpath.With("builder.ToSql").LogError(ctx, "failed to build query", err)
		return nil, err
	}

	logpath.LogDebug(ctx, "executing query..", logger.LogMeta{"query": query, "args": args})
	data = sudokuplayer.Players{}
	if err = repo.actuator.Reader().Find(ctx, query, args, &data); err != nil {
		logpath.With("Find").LogError(ctx, "failed query run", err)
		return nil, err
	}

	return data, nil
}

func (repo *repository) Get(ctx context.Context, filter sudokuplayer.PlayerFilter) (datum *sudokuplayer.Player, err error) {
	logpath := baselogpath.With("Get")

	builder := squirrel.
		Select(sudokuplayer.PlayerTable.SelectColumns...).
		From(sudokuplayer.PlayerTable.Name).
		PlaceholderFormat(squirrel.Dollar)
	builder = filter.ConditionQuery(builder)

	query, args, err := builder.ToSql()
	if err != nil {
		logpath.With("builder.ToSql").LogError(ctx, "failed to build query", err)
		return nil, err
	}

	logpath.LogDebug(ctx, "executing query..", logger.LogMeta{"query": query, "args": args})
	datum = &sudokuplayer.Player{}
	if err = repo.actuator.Reader().Get(ctx, query, args, datum); err != nil {
		if err != sql.ErrNoRows {
			logpath.With("Get").LogError(ctx, "failed query run", err)
		}
		return nil, err
	}

	return datum, nil
}
