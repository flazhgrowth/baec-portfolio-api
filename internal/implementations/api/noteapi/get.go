package noteapi

import (
	"context"
	"time"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/note"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/request"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/response"
)

func (api *api) ListNotes(req request.Request, resp response.Response) {
	ctx, cancel := context.WithTimeout(req.GetContext(), time.Second*2)
	defer cancel()

	args := note.ListNotesRequest{}
	if err := req.DecodeQueryParam(&args); err != nil {
		resp.RespondJSON(nil, err)
		return
	}

	listResp, err := api.noteSvc.ListNotes(ctx, *args.Normalize())
	if err != nil {
		resp.RespondJSON(nil, err)
		return
	}

	resp.RespondJSON(listResp, nil)
}

func (api *api) Summary(req request.Request, resp response.Response) {
	ctx, cancel := context.WithTimeout(req.GetContext(), time.Second*2)
	defer cancel()

	summaryResp, err := api.noteSvc.Summary(ctx)
	if err != nil {
		resp.RespondJSON(nil, err)
		return
	}

	resp.RespondJSON(summaryResp, nil)
}
