package specialentrysvc

import (
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/specialentry"
	"github.com/flazhgrowth/fg-tamagochi/app"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/sqlator/sqltx"
	"github.com/flazhgrowth/fg-tamagochi/pkg/logger"
)

var (
	baselogpath = logger.LogPath("specialentrysvc")
)

type service struct {
	tx               sqltx.SQLTx
	specialentryRepo specialentry.SpecialEntryRepository
}

func New(app *app.App, specialentryrepo specialentry.SpecialEntryRepository) specialentry.SpecialEntryService {
	return &service{
		tx:               app.GetTxSQLator(),
		specialentryRepo: specialentryrepo,
	}
}
