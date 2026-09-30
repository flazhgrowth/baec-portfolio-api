package accountrepo

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/account"
	"github.com/flazhgrowth/fg-tamagopkg/logger"
)

func (repo *repository) Update(ctx context.Context, fields account.AccountUpdateFields, filter account.AccountFilter) (err error) {
	logpath := baselogpath.With("Update")

	builder := squirrel.
		Update(account.AccountTable.Name).
		PlaceholderFormat(squirrel.Dollar)
	builder = fields.UpdateSetQuery(builder)
	builder = squirrel.UpdateBuilder(filter.ConditionQuery(squirrel.SelectBuilder(builder)))

	query, args, err := builder.ToSql()
	if err != nil {
		logpath.With("builder.ToSql").LogError(ctx, "failed to build query", err)
		return err
	}

	logpath.LogDebug(ctx, "executing query..", logger.LogMeta{"query": query, "args": args})
	if _, err = repo.actuator.Writer().Write(ctx, query, args); err != nil {
		logpath.With("Write").LogError(ctx, "failed query run", err)
		return err
	}

	return nil
}
