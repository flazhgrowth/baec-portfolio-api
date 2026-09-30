package sudokugameapi

import (
	"context"
	"time"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokugame"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/request"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/response"
)

func (api *api) GetGame(req request.Request, resp response.Response) {
	ctx, cancel := context.WithTimeout(req.GetContext(), time.Second*10)
	defer cancel()

	args := sudokugame.GetGameRequest{}
	if err := req.DecodeURLParam(&args); err != nil {
		resp.RespondJSON(nil, err)
		return
	}

	game, err := api.sudokuGameSvc.GetGame(ctx, args)
	if err != nil {
		resp.RespondJSON(nil, err)
		return
	}

	resp.RespondJSON(game, nil)
}
