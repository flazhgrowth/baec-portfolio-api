package guest

import (
	"time"

	"github.com/flazhgrowth/fg-tamagochi/pkg/db/entity"
)

type (
	ListGuestsRequest struct {
		entity.PaginationRequest
	}
	ListGuestsResponse struct {
		Data       GuestsResponses           `json:"guests"`
		Pagination entity.PaginationResponse `json:"pagination"`
	}
)

func (req *ListGuestsRequest) Normalize() *ListGuestsRequest {
	if req.Page == 0 {
		req.Page = 1
	}
	if req.Size == 0 {
		req.Size = 10
	}

	return req
}

type (
	GuestResponse struct {
		Name      string    `json:"name"`
		VisitedAt time.Time `json:"visited_at"`
	}
	GuestsResponses []GuestResponse
)

type (
	RegisterRequest struct {
		Name string `json:"name"`
	}
)
