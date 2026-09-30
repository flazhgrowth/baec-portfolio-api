package sudokugameapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokugame"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/apierrors"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/request"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/response"
)

const playerTokenHeader = "X-Player-Token"

func (api *api) SubmitMove(req request.Request, resp response.Response) {
	ctx, cancel := context.WithTimeout(req.GetContext(), time.Second*10)
	defer cancel()

	args := sudokugame.MoveRequest{}
	if err := req.DecodeURLParam(&args); err != nil {
		resp.RespondJSON(nil, err)
		return
	}
	args.Token = req.GetNetHTTPHeaders().Get(playerTokenHeader)
	if err := decodeBody(req, &args); err != nil {
		resp.RespondJSON(nil, err)
		return
	}

	move, err := api.sudokuGameSvc.SubmitMove(ctx, args)
	if err != nil {
		var conflict *sudokugame.ConflictError
		if errors.As(err, &conflict) {
			respondConflict(resp, conflict)
			return
		}

		resp.RespondJSON(nil, err)
		return
	}

	resp.RespondJSON(move, nil)
}

// decodeBody reads a JSON body. The contract answers a malformed body with 422
// VALIDATION_ERROR, where the shared decoder would answer 400.
func decodeBody(req request.Request, dest any) error {
	if err := req.DecodeBody(dest); err != nil {
		return apierrors.ErrorUnprocessableEntity("request body is not valid").WithCode("VALIDATION_ERROR")
	}

	return nil
}

// respondConflict answers 409 with the current game under data.game. The
// shared RespondJSON always nulls data on an error, so this writes the same
// envelope itself.
func respondConflict(resp response.Response, conflict *sudokugame.ConflictError) {
	impl, ok := resp.(*response.ResponseImpl)
	if !ok {
		resp.RespondJSON(nil, apierrors.ErrorConflict(conflict.Message).WithCode(conflict.Code))
		return
	}

	body, err := json.Marshal(response.BaseResponse{
		Code:       conflict.Code,
		Message:    conflict.Message,
		Data:       map[string]any{"game": conflict.Game},
		ServerTime: time.Now().Unix(),
	})
	if err != nil {
		resp.RespondJSON(nil, apierrors.ErrorInternalServerError())
		return
	}

	impl.Header().Set("Content-Type", "application/json")
	impl.WriteHeader(http.StatusConflict)
	impl.Write(body)
}
