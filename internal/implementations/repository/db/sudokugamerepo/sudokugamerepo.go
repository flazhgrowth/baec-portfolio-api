package sudokugamerepo

import (
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokugame"
	"github.com/flazhgrowth/fg-tamagochi/app"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/sqlator"
	"github.com/flazhgrowth/fg-tamagopkg/logger"
)

var (
	baselogpath logger.LogPath = "sudokugamerepo"
)

type repository struct {
	actuator sqlator.SQLator
}

func New(app *app.App) sudokugame.Repository {
	return &repository{
		actuator: app.GetSQLator(),
	}
}
