package sudokugameapi

import (
	"context"
	"time"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokugame"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/request"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/response"
)

func (api *api) JoinSession(req request.Request, resp response.Response) {
	ctx, cancel := context.WithTimeout(req.GetContext(), time.Second*10)
	defer cancel()

	caller, err := req.GetAccountInfo()
	if err != nil {
		resp.RespondJSON(nil, err)
		return
	}

	args := sudokugame.JoinSessionRequest{}
	if err = decodeBody(req, &args); err != nil {
		resp.RespondJSON(nil, err)
		return
	}

	session, err := api.sudokuGameSvc.JoinSession(ctx, caller, args)
	if err != nil {
		resp.RespondJSON(nil, err)
		return
	}

	resp.RespondJSON(session, nil)
}
