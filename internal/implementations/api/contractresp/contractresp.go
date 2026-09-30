// Package contractresp writes responses in the shape the Sudoku API contract
// defines: success bodies are the bare resource, and failures are
// {"error": {"code", "message", "game"?}}. The shared response.RespondJSON wraps
// both in the repo's {code, message, data, servertime} envelope instead, which
// the Sudoku client does not parse.
package contractresp

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/flazhgrowth/fg-tamagochi/pkg/http/apierrors"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/response"
)

type (
	errorBody struct {
		Error errorDetail `json:"error"`
	}
	errorDetail struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Game    any    `json:"game,omitempty"`
	}
)

// JSON writes body as the response with the given status.
func JSON(resp response.Response, status int, body any) {
	write(resp, status, body)
}

// NoContent writes an empty response with the given status (204).
func NoContent(resp response.Response, status int) {
	write(resp, status, nil)
}

// Error writes a contract error. Anything that is not an apierrors.HTTPError is
// reported as a 500 without leaking its text.
func Error(resp response.Response, err error) {
	var httpErr apierrors.HTTPError
	if !errors.As(err, &httpErr) {
		httpErr = apierrors.ErrorInternalServerError().WithCode("INTERNAL_ERROR")
		httpErr.Message = "internal server error"
	}

	write(resp, int(httpErr.StatusCode), errorBody{Error: errorDetail{Code: httpErr.Code, Message: httpErr.Message}})
}

// Conflict writes a 409 that carries the game as it now stands, so the client
// can resync without another call.
func Conflict(resp response.Response, code, message string, game any) {
	write(resp, http.StatusConflict, errorBody{Error: errorDetail{Code: code, Message: message, Game: game}})
}

func write(resp response.Response, status int, body any) {
	impl, ok := resp.(*response.ResponseImpl)
	if !ok {
		resp.RespondJSON(body, nil, status)
		return
	}

	if body == nil {
		impl.WriteHeader(status)
		return
	}

	data, err := json.Marshal(body)
	if err != nil {
		impl.WriteHeader(http.StatusInternalServerError)
		return
	}

	impl.Header().Set("Content-Type", "application/json")
	impl.WriteHeader(status)
	impl.Write(data)
}
