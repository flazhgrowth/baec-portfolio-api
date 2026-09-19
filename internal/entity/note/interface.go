package note

import (
	"context"

	"github.com/flazhgrowth/fg-tamagochi/pkg/http/request"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/response"
)

type NoteRepository interface {
	Insert(ctx context.Context, datum *Note) (err error)
	Find(ctx context.Context, filter NoteFilter, sorter NoteSorter) (data NotesWithGuest, err error)
	Count(ctx context.Context, filter NoteFilter) (count int64, err error)
	ListArtKeys(ctx context.Context) (artKeys []string, err error)
}

type NoteService interface {
	Create(ctx context.Context, args CreateRequest) (err error)
	ListNotes(ctx context.Context, args ListNotesRequest) (resp *ListNotesResponse, err error)
	Summary(ctx context.Context) (resp *SummaryResponse, err error)
}

type NoteAPI interface {
	Create(req request.Request, resp response.Response)
	ListNotes(req request.Request, resp response.Response)
	Summary(req request.Request, resp response.Response)
}
