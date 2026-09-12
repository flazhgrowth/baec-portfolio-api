package guestrepo

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/guest"
	"github.com/flazhgrowth/fg-tamagochi/pkg/logger"
)

func (repo *repository) Find(ctx context.Context, filter guest.GuestFilter) (data guest.Guests, err error) {
	logpath := baselogpath.With("Find")

	builder := squirrel.
		Select(guest.GuestTable.SelectColumns...).
		From(guest.GuestTable.Name).
		PlaceholderFormat(squirrel.Dollar)
	builder = filter.ConditionQuery(builder)

	query, args, err := builder.ToSql()
	if err != nil {
		logpath.With("builder.ToSql").LogError(ctx, "failed to build query", err)
		return nil, err
	}

	logpath.LogDebug(ctx, "executing query..", logger.LogMeta{"query": query, "args": args})
	data = guest.Guests{}
	if err = repo.db.Reader().Find(ctx, query, args, &data); err != nil {
		logpath.With("Find").LogError(ctx, "failed query run", err)
		return nil, err
	}

	return data, nil
}

func (repo *repository) Count(ctx context.Context, filter guest.GuestFilter) (count int64, err error) {
	logpath := baselogpath.With("Count")

	builder := squirrel.
		Select(guest.GuestTable.CountColumns...).
		From(guest.GuestTable.Name).
		PlaceholderFormat(squirrel.Dollar)
	builder = filter.ConditionQuery(builder)

	query, args, err := builder.ToSql()
	if err != nil {
		logpath.With("builder.ToSql").LogError(ctx, "failed to build query", err)
		return 0, err
	}

	logpath.LogDebug(ctx, "executing query..", logger.LogMeta{"query": query, "args": args})
	if err = repo.db.Reader().Get(ctx, query, args, &count); err != nil {
		logpath.With("Get").LogError(ctx, "failed query run", err)
		return 0, err
	}

	return count, nil
}
