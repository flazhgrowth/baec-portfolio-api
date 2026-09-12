package guestapi

import (
	"context"
	"time"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/guest"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/request"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/response"
)

func (api *api) ListGuests(req request.Request, resp response.Response) {
	ctx, cancel := context.WithTimeout(req.GetContext(), time.Second*2)
	defer cancel()

	args := guest.ListGuestsRequest{}
	if err := req.DecodeQueryParam(&args); err != nil {
		resp.RespondJSON(nil, err)
		return
	}

	listResp, err := api.guestSvc.ListGuests(ctx, *args.Normalize())
	if err != nil {
		resp.RespondJSON(nil, err)
		return
	}

	resp.RespondJSON(listResp, nil)
}
