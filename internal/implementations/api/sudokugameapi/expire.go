package sudokugameapi

import (
	"context"
	"errors"
	"time"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokugame"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/request"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/response"
)

func (api *api) ExpireTurn(req request.Request, resp response.Response) {
	ctx, cancel := context.WithTimeout(req.GetContext(), time.Second*10)
	defer cancel()

	args := sudokugame.ExpireTurnRequest{}
	if err := req.DecodeURLParam(&args); err != nil {
		resp.RespondJSON(nil, err)
		return
	}

	update, err := api.sudokuGameSvc.ExpireTurn(ctx, args)
	if err != nil {
		var conflict *sudokugame.ConflictError
		if errors.As(err, &conflict) {
			respondConflict(resp, conflict)
			return
		}

		resp.RespondJSON(nil, err)
		return
	}

	resp.RespondJSON(update, nil)
}
