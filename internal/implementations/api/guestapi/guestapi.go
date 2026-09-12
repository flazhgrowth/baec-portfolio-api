package guestapi

import "github.com/flazhgrowth/baec-portfolio-api/internal/entity/guest"

type api struct {
	guestSvc guest.GuestService
}

func New(guestsvc guest.GuestService) guest.GuestAPI {
	return &api{
		guestSvc: guestsvc,
	}
}
