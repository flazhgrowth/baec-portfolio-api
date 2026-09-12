package db

import (
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/account"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/guest"
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/repository/db/accountrepo"
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/repository/db/guestrepo"
	"github.com/google/wire"
)

var DBRepositoryWireSet = wire.NewSet(
	accountrepo.New,
	guestrepo.New,
)

type DBRepositories struct {
	AccountRepo account.Repository
	GuestRepo   guest.GuestRepository
}
