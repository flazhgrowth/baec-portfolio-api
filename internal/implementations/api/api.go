package api

import (
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/account"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/guest"
	accountapi "github.com/flazhgrowth/baec-portfolio-api/internal/implementations/api/acccountapi"
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/api/guestapi"
	"github.com/google/wire"
)

var APIWireSet = wire.NewSet(
	accountapi.New,
	guestapi.New,
)

type APIs struct {
	AccountAPI account.API
	GuestAPI   guest.GuestAPI
}
