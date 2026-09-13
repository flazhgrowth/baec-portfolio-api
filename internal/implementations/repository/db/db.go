package db

import (
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/account"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/guest"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/specialentry"
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/repository/db/accountrepo"
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/repository/db/guestrepo"
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/repository/db/specialentryrepo"
	"github.com/google/wire"
)

var DBRepositoryWireSet = wire.NewSet(
	accountrepo.New,
	guestrepo.New,
	specialentryrepo.New,
)

type DBRepositories struct {
	AccountRepo      account.Repository
	GuestRepo        guest.GuestRepository
	SpecialEntryRepo specialentry.SpecialEntryRepository
}
