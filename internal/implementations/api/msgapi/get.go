package msgapi

import (
	"context"
	"time"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/msg"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/request"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/response"
)

func (api *api) ListMessages(req request.Request, resp response.Response) {
	ctx, cancel := context.WithTimeout(req.GetContext(), time.Second*2)
	defer cancel()

	args := msg.ListMessagesRequest{}
	if err := req.DecodeQueryParam(&args); err != nil {
		resp.RespondJSON(nil, err)
		return
	}

	listResp, err := api.msgSvc.ListMessages(ctx, args)
	if err != nil {
		resp.RespondJSON(nil, err)
		return
	}

	resp.RespondJSON(listResp, nil)
}
