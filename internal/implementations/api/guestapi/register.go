package guestapi

import (
	"context"
	"net/http"
	"time"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/guest"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/request"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/response"
)

func (api *api) Register(req request.Request, resp response.Response) {
	ctx, cancel := context.WithTimeout(req.GetContext(), time.Second*2)
	defer cancel()

	args := guest.RegisterRequest{}
	if err := req.DecodeBody(&args); err != nil {
		resp.RespondJSON(nil, err)
		return
	}

	regisResp, err := api.guestSvc.Register(ctx, args)
	if err != nil {
		resp.RespondJSON(nil, err)
		return
	}

	resp.RespondJSON(regisResp, nil, http.StatusCreated)
}
