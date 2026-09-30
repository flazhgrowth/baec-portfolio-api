package sudokugamesvc

import (
	"time"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokugame"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokumove"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokuplayer"
	"github.com/flazhgrowth/fg-tamagochi/app"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/sqlator/sqltx"
	"github.com/flazhgrowth/fg-tamagopkg/logger"
	"github.com/flazhgrowth/fg-tamagopkg/ulid"
)

var (
	baselogpath = logger.LogPath("sudokugamesvc")

	now     = time.Now
	getUlid = func() string {
		id, _ := ulid.Generate()
		return id
	}
)

type service struct {
	tx         sqltx.SQLTx
	gameRepo   sudokugame.Repository
	playerRepo sudokuplayer.Repository
	moveRepo   sudokumove.Repository
	hub        *hub
}

func New(app *app.App, gamerepo sudokugame.Repository, playerrepo sudokuplayer.Repository, moverepo sudokumove.Repository) sudokugame.Service {
	return &service{
		tx:         app.GetTxSQLator(),
		gameRepo:   gamerepo,
		playerRepo: playerrepo,
		moveRepo:   moverepo,
		hub:        newHub(),
	}
}
