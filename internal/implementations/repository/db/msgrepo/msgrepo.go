package msgrepo

import (
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/msg"
	"github.com/flazhgrowth/fg-tamagochi/app"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/sqlator"
)

type repository struct {
	db sqlator.SQLator
}

func New(app *app.App) msg.MsgRepository {
	return &repository{
		db: app.GetSQLator(),
	}
}
