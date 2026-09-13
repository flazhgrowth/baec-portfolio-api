package specialentry

import (
	"github.com/Masterminds/squirrel"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/entity"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/table"
)

var (
	TableSpecialEntry table.Table = table.Table{
		Name:          "special_entries",
		SelectColumns: []string{"id", "code", "max_entry", "entry_count"},
		InsertColumns: []string{"code", "max_entry", "entry_count"},
	}
)

type (
	SpecialEntry struct {
		entity.BaseModel
		Code       string `db:"code"`
		MaxEntries int    `db:"max_entry"`
		EntryCount int    `db:"entry_count"`
	}
)

func (datum *SpecialEntry) InsertValuesQuery(builder squirrel.InsertBuilder) squirrel.InsertBuilder {
	return builder.
		Columns(TableSpecialEntry.InsertColumns...).
		Values(
			datum.Code,
			datum.MaxEntries,
			datum.EntryCount,
		)
}
