package guest

import (
	"time"

	"github.com/flazhgrowth/fg-tamagochi/pkg/db/entity"
)

type (
	ListGuestsRequest struct {
		entity.CursorPaginationRequest
	}
	ListGuestsResponse struct {
		Data       GuestsResponses                 `json:"guests"`
		Pagination entity.CursorPaginationResponse `json:"pagination"`
	}
)

func (req *ListGuestsRequest) Normalize() *ListGuestsRequest {
	if req.Size == 0 {
		req.Size = 10
	}

	req.DecodeCursor()
	return req
}

type (
	GuestResponse struct {
		ID        uint64    `json:"id"`
		Name      string    `json:"name"`
		VisitedAt time.Time `json:"visited_at"`
	}
	GuestsResponses []GuestResponse
)

type (
	RegisterRequest struct {
		Name    string `json:"name"`
		VisitID string `json:"visit_id"`
	}
	RegisterResponse struct {
		ID        uint64    `json:"id"`
		Name      string    `json:"name"`
		VisitedAt time.Time `json:"visited_at"`
	}
)
