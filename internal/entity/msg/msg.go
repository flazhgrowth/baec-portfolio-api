package msg

import "github.com/flazhgrowth/fg-tamagochi/pkg/db/entity"

type (
	CreateRequest struct {
		VisitID string `json:"visit_id"`
		Msg     string `json:"msg"`
	}
)

type (
	ListMessagesRequest struct {
		entity.CursorPaginationRequest
	}
	MessageResponse struct {
		Msg string `json:"msg"`
	}
	ListMessagesResponse struct {
		Messages   []MessageResponse               `json:"messages"`
		Pagination entity.CursorPaginationResponse `json:"pagination"`
	}
)
