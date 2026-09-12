package guest

import (
	"github.com/Masterminds/squirrel"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/model"
)

type (
	GuestFilter struct {
		Page model.Filter[int]
		Size model.Filter[int]
	}
)

func (filter *GuestFilter) ConditionQuery(builder squirrel.SelectBuilder) squirrel.SelectBuilder {
	if filter.Page.Valid && filter.Size.Valid {
		offset := (filter.Page.V - 1) / filter.Size.V
		builder = builder.
			Offset(uint64(offset)).
			Limit(uint64(filter.Size.V))
	}

	return builder
}
