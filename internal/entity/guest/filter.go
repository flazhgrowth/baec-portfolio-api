package guest

import (
	"github.com/Masterminds/squirrel"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/model"
)

type (
	GuestFilter struct {
		ID   model.Filter[uint64]
		Size model.Filter[int]
	}
	GuestSorter struct {
		Sorter []string
	}
)

func (filter *GuestFilter) ConditionQuery(builder squirrel.SelectBuilder) squirrel.SelectBuilder {
	builder = filter.ID.ConditionQuery(builder, "id")
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
