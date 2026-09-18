package guestrepo

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/guest"
)

func (repo *repository) Update(ctx context.Context, fields guest.GuestFields, filter guest.GuestFilter) (err error) {
	builder := squirrel.
		Update(guest.GuestTable.Name).
		PlaceholderFormat(squirrel.Dollar)
	builder = fields.UpdateSetQuery(builder)
	builder = squirrel.UpdateBuilder(filter.ConditionQuery(squirrel.SelectBuilder(builder)))
	query, args, err := builder.ToSql()
	if err != nil {
		return nil
	}

	if _, err = repo.db.Writer().Write(ctx, query, args); err != nil {
		return err
	}

	return nil
}
