package msgapi

import "github.com/flazhgrowth/baec-portfolio-api/internal/entity/msg"

type api struct {
	msgSvc msg.MsgService
}

func New(msgsvc msg.MsgService) msg.MsgAPI {
	return &api{
		msgSvc: msgsvc,
	}
}
