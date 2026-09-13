package specialentry

import (
	"database/sql"

	"github.com/Masterminds/squirrel"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/model"
)

type (
	SpecialEntryFilter struct {
		ID   model.Filter[uint64]
		Code model.Filter[string]
	}
	SpecialEntryFields struct {
		EntryCount sql.Null[int]
	}
)

func (filter *SpecialEntryFilter) ConditionQuery(builder squirrel.SelectBuilder) squirrel.SelectBuilder {
	builder = filter.ID.ConditionQuery(builder, "id")
	builder = filter.Code.ConditionQuery(builder, "code")

	return builder
}

func (fields *SpecialEntryFields) UpdateSetQuery(builder squirrel.UpdateBuilder) squirrel.UpdateBuilder {
	if fields.EntryCount.Valid {
		builder = builder.Set("entry_count", fields.EntryCount.V)
	}

	return builder
}
