package noteroutes

import (
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/note"
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/api"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/router"
)

var (
	tag = "Note"
)

func Routes(version router.Router, apis *api.APIs) {
	version.Group("/notes", func(noteR router.Router) {
		noteR.Get("/", apis.NoteAPI.ListNotes, &router.RouterDocs{
			Request:     note.ListNotesRequest{},
			Response:    note.ListNotesResponse{},
			Tags:        tag,
			Title:       "List Notes",
			Description: "List notes left on one artwork, newest first",
		})
		noteR.Get("/summary", apis.NoteAPI.Summary, &router.RouterDocs{
			Response:    note.SummaryResponse{},
			Tags:        tag,
			Title:       "Notes Summary",
			Description: "List art keys that have at least one note",
		})
		noteR.Post("/", apis.NoteAPI.Create, &router.RouterDocs{
			Request:     note.CreateRequest{},
			Tags:        tag,
			Title:       "Leave Note",
			Description: "Leave a note on a specific artwork",
		})
	})
}
