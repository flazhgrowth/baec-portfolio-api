package guestrepo

import (
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/guest"
	"github.com/flazhgrowth/fg-tamagochi/app"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/sqlator"
	"github.com/flazhgrowth/fg-tamagochi/pkg/logger"
)

var (
	baselogpath = logger.LogPath("guestrepo")
)

type repository struct {
	db sqlator.SQLator
}

func New(app *app.App) guest.GuestRepository {
	return &repository{
		db: app.GetSQLator(),
	}
}
