package note

import (
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/entity"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/table"
)

var (
	NoteTable table.Table = table.Table{
		Name:          "notes",
		SelectColumns: []string{"notes.id", "guests.name", "notes.note", "notes.created_at"},
		InsertColumns: []string{"visit_id", "art_key", "note"},
		CountColumns:  []string{"COUNT(notes.id)"},
	}
)

type (
	Note struct {
		entity.BaseModel
		VisitID string `db:"visit_id"`
		ArtKey  string `db:"art_key"`
		Note    string `db:"note"`
	}
	Notes []Note

	// NoteWithGuest is a note joined with the name of the guest who left it.
	NoteWithGuest struct {
		ID        uint64    `db:"id"`
		Name      string    `db:"name"`
		Note      string    `db:"note"`
		CreatedAt time.Time `db:"created_at"`
	}
	NotesWithGuest []NoteWithGuest
)

func (datum *Note) InsertValuesQuery(builder squirrel.InsertBuilder) squirrel.InsertBuilder {
	return builder.
		Columns(NoteTable.InsertColumns...).
		Values(
			datum.VisitID,
			datum.ArtKey,
			datum.Note,
		)
}
