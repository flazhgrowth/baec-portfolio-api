package msg

import (
	"github.com/Masterminds/squirrel"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/model"
)

type (
	MsgFilter struct {
		VisitID model.Filter[string]
	}
)

func (filter *MsgFilter) ConditionQuery(builder squirrel.SelectBuilder) squirrel.SelectBuilder {
	builder = filter.VisitID.ConditionQuery(builder, "visit_id")

	return builder
}
