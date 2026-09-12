package guestrepo

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/guest"
)

func (repo *repository) Insert(ctx context.Context, datum *guest.Guest) (err error) {
	builder := squirrel.
		Insert(guest.GuestTable.Name).
		PlaceholderFormat(squirrel.Dollar)
	builder = datum.InsertValuesQuery(builder)

	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	if err = repo.db.Writer().Insert(ctx, query, args, &datum.ID); err != nil {
		return err
	}

	return nil
}
