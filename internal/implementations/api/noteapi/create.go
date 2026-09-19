package noteapi

import (
	"context"
	"time"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/note"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/request"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/response"
)

func (api *api) Create(req request.Request, resp response.Response) {
	ctx, cancel := context.WithTimeout(req.GetContext(), time.Second*2)
	defer cancel()

	args := note.CreateRequest{}
	if err := req.DecodeBody(&args); err != nil {
		resp.RespondJSON(nil, err)
		return
	}

	if err := api.noteSvc.Create(ctx, args); err != nil {
		resp.RespondJSON(nil, err)
		return
	}

	resp.RespondJSON(nil, nil)
}
