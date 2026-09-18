package guest

import (
	"database/sql"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/model"
)

type (
	GuestFilter struct {
		ID      model.Filter[uint64]
		VisitID model.Filter[string]
		Size    model.Filter[int]
	}
	GuestFields struct {
		Name sql.NullString
	}
	GuestSorter struct {
		Sorter []string
	}
)

func (filter *GuestFilter) ConditionQuery(builder squirrel.SelectBuilder) squirrel.SelectBuilder {
	builder = filter.ID.ConditionQuery(builder, "id")
	builder = filter.VisitID.ConditionQuery(builder, "visit_id")
	if filter.Size.Valid {
		builder = builder.Limit(uint64(filter.Size.V))
	}

	return builder
}

func (sorter *GuestSorter) SortQuery(builder squirrel.SelectBuilder) squirrel.SelectBuilder {
	if len(sorter.Sorter) > 0 {
		builder = builder.OrderBy(sorter.Sorter...)
	}

	return builder
}

func (fields *GuestFields) UpdateSetQuery(builder squirrel.UpdateBuilder) squirrel.UpdateBuilder {
	if fields.Name.Valid {
		builder = builder.Set("name", fields.Name.String)
	}

	return builder.Set("updated_at", time.Now())
}
