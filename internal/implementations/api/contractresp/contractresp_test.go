package contractresp

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/flazhgrowth/fg-tamagochi/pkg/http/apierrors"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/response"
)

func TestContractResponses(t *testing.T) {
	t.Run("success is the bare body, not wrapped", func(t *testing.T) {
		rec := httptest.NewRecorder()
		JSON(response.New(rec), http.StatusCreated, map[string]string{"id": "u1"})
		if rec.Code != 201 || strings.TrimSpace(rec.Body.String()) != `{"id":"u1"}` || rec.Header().Get("Content-Type") != "application/json" {
			t.Fatalf("%d %q %q", rec.Code, rec.Body.String(), rec.Header())
		}
	})

	t.Run("204 has no body and no content type", func(t *testing.T) {
		rec := httptest.NewRecorder()
		NoContent(response.New(rec), http.StatusNoContent)
		if rec.Code != 204 || rec.Body.Len() != 0 {
			t.Fatalf("%d %q", rec.Code, rec.Body.String())
		}
	})

	t.Run("an API error becomes {error:{code,message}} with its status", func(t *testing.T) {
		rec := httptest.NewRecorder()
		Error(response.New(rec), apierrors.ErrorUnauthorized("bad token").WithCode("INVALID_TOKEN"))
		if rec.Code != 401 || strings.TrimSpace(rec.Body.String()) != `{"error":{"code":"INVALID_TOKEN","message":"bad token"}}` {
			t.Fatalf("%d %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("an unknown error is a 500 that does not leak its text", func(t *testing.T) {
		rec := httptest.NewRecorder()
		Error(response.New(rec), errors.New("pq: password authentication failed for user baeclatant"))
		if rec.Code != 500 || strings.Contains(rec.Body.String(), "baeclatant") || !strings.Contains(rec.Body.String(), "INTERNAL_ERROR") {
			t.Fatalf("%d %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("a conflict carries the game under error.game", func(t *testing.T) {
		rec := httptest.NewRecorder()
		Conflict(response.New(rec), "NOT_YOUR_TURN", "It is not your turn", map[string]int{"version": 3})
		var body struct {
			Error struct {
				Code string         `json:"code"`
				Game map[string]int `json:"game"`
			} `json:"error"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || rec.Code != 409 || body.Error.Code != "NOT_YOUR_TURN" || body.Error.Game["version"] != 3 {
			t.Fatalf("%d %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("an error without a game omits the key", func(t *testing.T) {
		rec := httptest.NewRecorder()
		Error(response.New(rec), apierrors.ErrorDataNotFound("nope").WithCode("GAME_NOT_FOUND"))
		if strings.Contains(rec.Body.String(), `"game"`) {
			t.Fatal(rec.Body.String())
		}
	})
}
