package service

import (
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/account"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/guest"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/specialentry"
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/service/accountsvc"
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/service/guestsvc"
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/service/specialentrysvc"
	"github.com/google/wire"
)

var ServiceWireSet = wire.NewSet(
	accountsvc.New,
	guestsvc.New,
	specialentrysvc.New,
)

type Services struct {
	AccountSvc      account.Service
	GuestSvc        guest.GuestService
	SpecialEntrySvc specialentry.SpecialEntryService
}
