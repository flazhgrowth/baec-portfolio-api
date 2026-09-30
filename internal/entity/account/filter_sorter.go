package account

import (
	"database/sql"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/model"
)

type (
	AccountFilter struct {
		Username model.Filter[string]
	}
	AccountSorter struct {
		Sorter []string
	}
	AccountUpdateFields struct {
		Name sql.NullString
	}
)

func (datum *Account) InsertValuesQuery(builder squirrel.InsertBuilder) squirrel.InsertBuilder {
	return builder.
		Columns(AccountTable.InsertColumns...).
		Values(
			datum.ID,
			datum.Username,
			datum.Password,
			datum.Name,
			datum.Salt,
		)
}

func (filter *AccountFilter) ConditionQuery(builder squirrel.SelectBuilder) squirrel.SelectBuilder {
	builder = filter.Username.ConditionQuery(builder, "username")

	return builder
}

func (sorter *AccountSorter) SortQuery(builder squirrel.SelectBuilder) squirrel.SelectBuilder {
	if len(sorter.Sorter) > 0 {
		builder = builder.OrderBy(sorter.Sorter...)
	}

	return builder
}

func (fields *AccountUpdateFields) UpdateSetQuery(builder squirrel.UpdateBuilder) squirrel.UpdateBuilder {
	if fields.Name.Valid {
		builder = builder.Set("name", fields.Name.String)
	}

	return builder.Set("updated_at", time.Now())
}
