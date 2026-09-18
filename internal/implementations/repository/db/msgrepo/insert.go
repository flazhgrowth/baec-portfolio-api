package msgrepo

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/msg"
)

func (repo *repository) Insert(ctx context.Context, datum *msg.Msg) (err error) {
	builder := squirrel.
		Insert(msg.MsgTable.Name).
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

func (repo *repository) Find(ctx context.Context, filter msg.MsgFilter) (data msg.Msgs, err error) {
	builder := squirrel.
		Select(msg.MsgTable.SelectColumns...).
		From(msg.MsgTable.Name).
		PlaceholderFormat(squirrel.Dollar)
	builder = filter.ConditionQuery(builder)
	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	if err = repo.db.Reader().Find(ctx, query, args, &data); err != nil {
		return nil, err
	}

	return data, nil
}
