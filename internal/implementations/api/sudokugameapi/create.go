package sudokugameapi

import (
	"context"
	"net/http"
	"time"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokugame"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/request"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/response"
)

func (api *api) CreateSession(req request.Request, resp response.Response) {
	ctx, cancel := context.WithTimeout(req.GetContext(), time.Second*10)
	defer cancel()

	caller, err := req.GetAccountInfo()
	if err != nil {
		resp.RespondJSON(nil, err)
		return
	}

	args := sudokugame.CreateSessionRequest{}
	if err = req.DecodeBody(&args); err != nil {
		resp.RespondJSON(nil, err)
		return
	}

	session, err := api.sudokuGameSvc.CreateSession(ctx, caller, args)
	if err != nil {
		resp.RespondJSON(nil, err)
		return
	}

	resp.RespondJSON(session, nil, http.StatusCreated)
}
