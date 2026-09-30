package sudokugamerepo

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokugame"
	"github.com/flazhgrowth/fg-tamagopkg/logger"
)

// Update applies fields to every row matching filter and returns sql.ErrNoRows
// if none matched.
//
// It goes through Writer().Insert, not Write, on purpose: for Postgres, Write
// cannot report how many rows changed, while Insert appends `RETURNING id` and
// scans it. That turns a guarded UPDATE (e.g. `WHERE status = 'waiting'`) into
// an atomic claim: of two concurrent callers, exactly one gets a row back.
func (repo *repository) Update(ctx context.Context, fields sudokugame.GameUpdateFields, filter sudokugame.GameFilter) (err error) {
	logpath := baselogpath.With("Update")

	builder := squirrel.
		Update(sudokugame.GameTable.Name).
		PlaceholderFormat(squirrel.Dollar)
	builder = fields.UpdateSetQuery(builder)
	builder = squirrel.UpdateBuilder(filter.ConditionQuery(squirrel.SelectBuilder(builder)))

	query, args, err := builder.ToSql()
	if err != nil {
		logpath.With("builder.ToSql").LogError(ctx, "failed to build query", err)
		return err
	}

	logpath.LogDebug(ctx, "executing query..", logger.LogMeta{"query": query, "args": args})
	var id string
	if err = repo.actuator.Writer().Insert(ctx, query, args, &id); err != nil {
		return err
	}

	return nil
}
