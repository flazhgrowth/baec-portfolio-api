package accountrepo

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/account"
)

func (repo *repository) Insert(ctx context.Context, datum *account.Account) (err error) {
	logpath := baselogpath.With("Insert")

	builder := squirrel.
		Insert(account.AccountTable.Name).
		PlaceholderFormat(squirrel.Dollar)
	builder = datum.InsertValuesQuery(builder)
	query, args, err := builder.ToSql()
	if err != nil {
		logpath.With("builder.ToSql").LogError(ctx, "failed to build query", err)
		return err
	}

	if err = repo.actuator.Writer().Insert(ctx, query, args, &datum.ID); err != nil {
		return err
	}

	return nil
}
