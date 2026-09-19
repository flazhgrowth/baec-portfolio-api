package note

import (
	"time"

	"github.com/flazhgrowth/fg-tamagochi/pkg/db/entity"
)

const (
	MaxNoteLength   = 240
	DefaultListSize = 50
	MaxListSize     = 100
)

type (
	CreateRequest struct {
		VisitID string `json:"visit_id"`
		ArtKey  string `json:"art_key"`
		Note    string `json:"note"`
	}
)

type (
	ListNotesRequest struct {
		ArtKey string `query:"art_key"`
		entity.CursorPaginationRequest
	}
	NoteResponse struct {
		ID     uint64    `json:"id"`
		Name   string    `json:"name"`
		Note   string    `json:"note"`
		LeftAt time.Time `json:"left_at"`
	}
	PaginationResponse struct {
		Total  int     `json:"total"`
		Cursor *string `json:"cursor"`
	}
	ListNotesResponse struct {
		Notes      []NoteResponse     `json:"notes"`
		Pagination PaginationResponse `json:"pagination"`
	}
)

func (req *ListNotesRequest) Normalize() *ListNotesRequest {
	if req.Size <= 0 {
		req.Size = DefaultListSize
	}
	if req.Size > MaxListSize {
		req.Size = MaxListSize
	}

	req.DecodeCursor()
	return req
}

type (
	SummaryResponse struct {
		ArtKeys []string `json:"art_keys"`
	}
)
