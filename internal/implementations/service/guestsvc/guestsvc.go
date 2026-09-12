package guestsvc

import (
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/guest"
	"github.com/flazhgrowth/fg-tamagochi/pkg/logger"
)

var (
	baselogpath logger.LogPath = logger.LogPath("guestsvc")
)

type service struct {
	guestRepo guest.GuestRepository
}

func New(guestrepo guest.GuestRepository) guest.GuestService {
	return &service{
		guestRepo: guestrepo,
	}
}
