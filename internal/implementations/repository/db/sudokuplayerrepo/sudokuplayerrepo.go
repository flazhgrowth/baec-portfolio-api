package sudokuplayerrepo

import (
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokuplayer"
	"github.com/flazhgrowth/fg-tamagochi/app"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/sqlator"
	"github.com/flazhgrowth/fg-tamagopkg/logger"
)

var (
	baselogpath logger.LogPath = "sudokuplayerrepo"
)

type repository struct {
	actuator sqlator.SQLator
}

func New(app *app.App) sudokuplayer.Repository {
	return &repository{
		actuator: app.GetSQLator(),
	}
}
