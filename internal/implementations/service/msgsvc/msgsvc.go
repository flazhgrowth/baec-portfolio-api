package msgsvc

import (
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/guest"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/msg"
)

type service struct {
	msgRepo   msg.MsgRepository
	guestRepo guest.GuestRepository
}

func New(msgrepo msg.MsgRepository, guestrepo guest.GuestRepository) msg.MsgService {
	return &service{
		msgRepo:   msgrepo,
		guestRepo: guestrepo,
	}
}
