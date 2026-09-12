package guest

import (
	"github.com/Masterminds/squirrel"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/entity"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/table"
)

var (
	GuestTable table.Table = table.Table{
		Name:          "guests",
		SelectColumns: []string{"id", "name", "visit_id", "created_at"},
		InsertColumns: []string{"name", "visit_id"},
		CountColumns:  []string{"COUNT(id)"},
	}
)

type (
	Guest struct {
		entity.BaseModel
		Name    string `db:"name"`
		VisitID string `db:"visit_id"`
	}
	Guests []Guest
)

func (datum *Guest) InsertValuesQuery(builder squirrel.InsertBuilder) squirrel.InsertBuilder {
	return builder.
		Columns(GuestTable.InsertColumns...).
		Values(
			datum.Name,
			datum.VisitID,
		)
}
