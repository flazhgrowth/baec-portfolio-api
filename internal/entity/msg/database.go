package msg

import (
	"github.com/Masterminds/squirrel"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/entity"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/table"
)

var (
	MsgTable table.Table = table.Table{
		Name:          "msgs",
		SelectColumns: []string{"id", "visit_id", "msg", "created_at", "updated_at"},
		InsertColumns: []string{"visit_id", "msg"},
	}
)

type (
	Msg struct {
		entity.BaseModel
		VisitID string `db:"visit_id"`
		Message string `db:"msg"`
	}
	Msgs []Msg
)

func (datum *Msg) InsertValuesQuery(builder squirrel.InsertBuilder) squirrel.InsertBuilder {
	return builder.
		Columns(MsgTable.InsertColumns...).
		Values(
			datum.VisitID,
			datum.Message,
		)
}
