package accountrepo

import (
	"context"
	"database/sql"

	"github.com/Masterminds/squirrel"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/account"
	"github.com/flazhgrowth/fg-tamagopkg/logger"
)

func (repo *repository) Get(ctx context.Context, filter account.AccountFilter) (datum *account.Account, err error) {
	logpath := baselogpath.With("Get")

	builder := squirrel.
		Select(account.AccountTable.SelectColumns...).
		From(account.AccountTable.Name).
		PlaceholderFormat(squirrel.Dollar)
	builder = filter.ConditionQuery(builder)

	query, args, err := builder.ToSql()
	if err != nil {
		logpath.With("builder.ToSql").LogError(ctx, "failed to build query", err)
		return nil, err
	}

	logpath.LogDebug(ctx, "executing query..", logger.LogMeta{"query": query, "args": args})
	datum = &account.Account{}
	if err = repo.actuator.Reader().Get(ctx, query, args, datum); err != nil {
		logpath.With("Get").LogError(ctx, "failed query run", err)
		return nil, err
	}

	if datum.ID == "" {
		return nil, sql.ErrNoRows
	}

	return datum, nil
}

func (repo *repository) Find(ctx context.Context, filter account.AccountFilter, sorter account.AccountSorter) (data account.Accounts, err error) {
	logpath := baselogpath.With("Find")

	builder := squirrel.
		Select(account.AccountTable.SelectColumns...).
		From(account.AccountTable.Name).
		PlaceholderFormat(squirrel.Dollar)
	builder = filter.ConditionQuery(builder)
	builder = sorter.SortQuery(builder)

	query, args, err := builder.ToSql()
	if err != nil {
		logpath.With("builder.ToSql").LogError(ctx, "failed to build query", err)
		return nil, err
	}

	logpath.LogDebug(ctx, "executing query..", logger.LogMeta{"query": query, "args": args})
	data = account.Accounts{}
	if err = repo.actuator.Reader().Find(ctx, query, args, &data); err != nil {
		logpath.With("Find").LogError(ctx, "failed query run", err)
		return nil, err
	}

	return data, nil
}
