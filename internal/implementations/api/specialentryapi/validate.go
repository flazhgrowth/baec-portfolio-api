package specialentryapi

import (
	"context"
	"time"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/specialentry"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/request"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/response"
)

func (api *api) Validate(req request.Request, resp response.Response) {
	ctx, cancel := context.WithTimeout(req.GetContext(), time.Second*2)
	defer cancel()

	args := specialentry.ValidateRequest{}
	if err := req.DecodeBody(&args); err != nil {
		resp.RespondJSON(nil, err)
		return
	}

	if err := api.specialentrySvc.Validate(ctx, args); err != nil {
		resp.RespondJSON(nil, err)
		return
	}

	resp.RespondJSON(nil, nil)
}
