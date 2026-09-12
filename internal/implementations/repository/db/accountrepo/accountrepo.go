package accountrepo

import (
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/account"
	"github.com/flazhgrowth/fg-tamagochi/app"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/sqlator"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/sqlator/sqltx"
)

type repository struct {
	actuator sqlator.SQLator
	tx       sqltx.SQLTx
}

func New(app *app.App) account.Repository {
	return &repository{
		actuator: app.GetSQLator(),
		tx:       app.GetTxSQLator(),
	}
}
