package specialentryrepo

import (
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/specialentry"
	"github.com/flazhgrowth/fg-tamagochi/app"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/sqlator"
	"github.com/flazhgrowth/fg-tamagopkg/logger"
)

var (
	baselogpath logger.LogPath = logger.LogPath("specialentryrepo")
)

type repository struct {
	db sqlator.SQLator
}

func New(app *app.App) specialentry.SpecialEntryRepository {
	return &repository{
		db: app.GetSQLator(),
	}
}
