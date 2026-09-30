package guestsvc

import (
	"time"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/guest"
	"github.com/flazhgrowth/fg-tamagopkg/logger"
)

var (
	baselogpath logger.LogPath = logger.LogPath("guestsvc")
	now                        = func() time.Time {
		return time.Now()
	}
)

type service struct {
	guestRepo guest.GuestRepository
}

func New(guestrepo guest.GuestRepository) guest.GuestService {
	return &service{
		guestRepo: guestrepo,
	}
}
