package notesvc

import (
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/guest"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/note"
	"github.com/flazhgrowth/fg-tamagochi/pkg/logger"
)

var baselogpath logger.LogPath = logger.LogPath("notesvc")

type service struct {
	noteRepo  note.NoteRepository
	guestRepo guest.GuestRepository
}

func New(noterepo note.NoteRepository, guestrepo guest.GuestRepository) note.NoteService {
	return &service{
		noteRepo:  noterepo,
		guestRepo: guestrepo,
	}
}
