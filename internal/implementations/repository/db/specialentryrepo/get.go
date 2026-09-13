package specialentryrepo

import (
	"context"
	"database/sql"

	"github.com/Masterminds/squirrel"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/specialentry"
	"github.com/flazhgrowth/fg-tamagochi/pkg/logger"
)

func (repo *repository) Get(ctx context.Context, filter specialentry.SpecialEntryFilter) (datum *specialentry.SpecialEntry, err error) {
	logpath := baselogpath.With("Get")

	builder := squirrel.
		Select(specialentry.TableSpecialEntry.SelectColumns...).
		From(specialentry.TableSpecialEntry.Name).
		PlaceholderFormat(squirrel.Dollar)
	builder = filter.ConditionQuery(builder)

	query, args, err := builder.ToSql()
	if err != nil {
		logpath.With("ToSql").LogError(ctx, "failed on building query", err)
		return nil, err
	}

	logpath.LogDebug(ctx, "executing query..", logger.LogMeta{"query": query, "args": args})
	datum = &specialentry.SpecialEntry{}
	if err = repo.db.Reader().Get(ctx, query, args, datum); err != nil {
		logpath.With("Get").LogError(ctx, "failed on executing query", err)
		return nil, err
	}

	if datum.ID == 0 {
		return nil, sql.ErrNoRows
	}

	return datum, nil
}

func (repo *repository) Insert(ctx context.Context, datum *specialentry.SpecialEntry) (err error) {
	logpath := baselogpath.With("Insert")

	builder := squirrel.
		Insert(specialentry.TableSpecialEntry.Name).
		PlaceholderFormat(squirrel.Dollar)
	builder = datum.InsertValuesQuery(builder)

	query, args, err := builder.ToSql()
	if err != nil {
		logpath.With("ToSql").LogError(ctx, "failed on building query", err)
		return err
	}

	logpath.LogDebug(ctx, "executing query..", logger.LogMeta{"query": query, "args": args})
	if err = repo.db.Writer().Insert(ctx, query, args, &datum.ID); err != nil {
		logpath.With("Insert").LogError(ctx, "faild on executing query", err)
		return err
	}

	return nil
}

func (repo *repository) Update(ctx context.Context, fields specialentry.SpecialEntryFields, filter specialentry.SpecialEntryFilter) (err error) {
	logpath := baselogpath.With("Update")

	builder := squirrel.
		Update(specialentry.TableSpecialEntry.Name).
		PlaceholderFormat(squirrel.Dollar)
	builder = fields.UpdateSetQuery(builder)
	builder = squirrel.UpdateBuilder(filter.ConditionQuery(squirrel.SelectBuilder(builder)))

	query, args, err := builder.ToSql()
	if err != nil {
		logpath.With("ToSql").LogError(ctx, "failed on building query", err)
		return err
	}

	logpath.LogDebug(ctx, "executing query..", logger.LogMeta{"query": query, "args": args})
	if _, err = repo.db.Writer().Write(ctx, query, args); err != nil {
		logpath.With("Write").LogError(ctx, "faild on executing query", err)
		return err
	}

	return nil
}
