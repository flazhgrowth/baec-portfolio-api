package noteapi

import "github.com/flazhgrowth/baec-portfolio-api/internal/entity/note"

type api struct {
	noteSvc note.NoteService
}

func New(notesvc note.NoteService) note.NoteAPI {
	return &api{
		noteSvc: notesvc,
	}
}
