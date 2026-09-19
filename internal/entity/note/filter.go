package note

import (
	"github.com/Masterminds/squirrel"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/model"
)

type (
	NoteFilter struct {
		ID      model.Filter[uint64]
		VisitID model.Filter[string]
		ArtKey  model.Filter[string]
		Size    model.Filter[int]
	}
	NoteSorter struct {
		Sorter []string
	}
)

func (filter *NoteFilter) ConditionQuery(builder squirrel.SelectBuilder) squirrel.SelectBuilder {
	builder = filter.ID.ConditionQuery(builder, "notes.id")
	builder = filter.VisitID.ConditionQuery(builder, "notes.visit_id")
	builder = filter.ArtKey.ConditionQuery(builder, "notes.art_key")
	if filter.Size.Valid {
		builder = builder.Limit(uint64(filter.Size.V))
	}

	return builder
}

func (sorter *NoteSorter) SortQuery(builder squirrel.SelectBuilder) squirrel.SelectBuilder {
	if len(sorter.Sorter) > 0 {
		builder = builder.OrderBy(sorter.Sorter...)
	}

	return builder
}
