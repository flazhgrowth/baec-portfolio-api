package api

import (
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/account"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/guest"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/specialentry"
	accountapi "github.com/flazhgrowth/baec-portfolio-api/internal/implementations/api/acccountapi"
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/api/guestapi"
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/api/specialentryapi"
	"github.com/google/wire"
)

var APIWireSet = wire.NewSet(
	accountapi.New,
	guestapi.New,
	specialentryapi.New,
)

type APIs struct {
	AccountAPI      account.API
	GuestAPI        guest.GuestAPI
	SpecialEntryAPI specialentry.SpecialEntryAPI
}
