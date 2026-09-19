package noterepo

import (
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/note"
	"github.com/flazhgrowth/fg-tamagochi/app"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/sqlator"
	"github.com/flazhgrowth/fg-tamagochi/pkg/logger"
)

var baselogpath logger.LogPath = logger.LogPath("noterepo")

type repository struct {
	db sqlator.SQLator
}

func New(app *app.App) note.NoteRepository {
	return &repository{
		db: app.GetSQLator(),
	}
}
