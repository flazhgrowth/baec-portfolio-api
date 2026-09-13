package specialentryapi

import (
	"context"
	"net/http"
	"time"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/specialentry"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/request"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/response"
)

func (api *api) CMSRegister(req request.Request, resp response.Response) {
	ctx, cancel := context.WithTimeout(req.GetContext(), time.Second*2)
	defer cancel()

	args := specialentry.CMSRegisterRequest{}
	if err := req.DecodeBody(&args); err != nil {
		resp.RespondJSON(nil, err)
		return
	}

	cmsResp, err := api.specialentrySvc.CMSRegister(ctx, args)
	if err != nil {
		resp.RespondJSON(nil, err)
		return
	}

	resp.RespondJSON(cmsResp, nil, http.StatusCreated)
}
