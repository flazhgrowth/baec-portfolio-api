package notesvc

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/note"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/entity"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/model"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/apierrors"
)

func (svc *service) ListNotes(ctx context.Context, args note.ListNotesRequest) (resp *note.ListNotesResponse, err error) {
	logpath := baselogpath.With("ListNotes")

	if !note.ArtKey(args.ArtKey).Valid() {
		return nil, apierrors.ErrorBadRequest("invalid art_key")
	}

	var lastSeenID uint64
	if args.Cursor != "" {
		lastSeenID, err = strconv.ParseUint(string(args.LastSeenID), 10, 64)
		if err != nil {
			logpath.With("parseCursor").LogError(ctx, "failed on parsing last seen ID", err)
			return nil, apierrors.ErrorBadRequest("invalid cursor")
		}
	}

	artKeyFilter := model.Filter[string]{Valid: true, V: args.ArtKey}
	notesFound, err := svc.noteRepo.Find(ctx, note.NoteFilter{
		ID:     model.Filter[uint64]{Valid: args.Cursor != "", V: lastSeenID, Comparator: model.Lt},
		ArtKey: artKeyFilter,
		Size:   model.Filter[int]{Valid: true, V: args.Size + 1},
	}, note.NoteSorter{Sorter: []string{"notes.id DESC"}})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		logpath.With("noteRepo.Find").LogError(ctx, "failed to find notes", err)
		return nil, apierrors.ErrorInternalServerError()
	}
	hasNext := len(notesFound) > args.Size
	if hasNext {
		notesFound = notesFound[:args.Size]
	}

	count, err := svc.noteRepo.Count(ctx, note.NoteFilter{ArtKey: artKeyFilter})
	if err != nil {
		logpath.With("noteRepo.Count").LogError(ctx, "failed to count notes", err)
		return nil, apierrors.ErrorInternalServerError()
	}

	resp = &note.ListNotesResponse{Notes: make([]note.NoteResponse, 0, len(notesFound))}
	for _, noteFound := range notesFound {
		resp.Notes = append(resp.Notes, note.NoteResponse{
			ID:     noteFound.ID,
			Name:   noteFound.Name,
			Note:   noteFound.Note,
			LeftAt: noteFound.CreatedAt,
		})
	}

	var pager entity.CursorPaginationResponse
	if len(resp.Notes) > 0 {
		lastID := resp.Notes[len(resp.Notes)-1].ID
		if err = pager.Calculate("id", entity.LastSeenVal(fmt.Sprintf("%d", lastID)), int(count), hasNext); err != nil {
			logpath.With("pager.Calculate").LogError(ctx, "failed to build cursor", err)
			return nil, apierrors.ErrorInternalServerError()
		}
	}
	resp.Pagination = note.PaginationResponse{Total: int(count), Cursor: pager.Cursor}

	return resp, nil
}

func (svc *service) Summary(ctx context.Context) (resp *note.SummaryResponse, err error) {
	logpath := baselogpath.With("Summary")

	artKeys, err := svc.noteRepo.ListArtKeys(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		logpath.With("noteRepo.ListArtKeys").LogError(ctx, "failed to list art keys", err)
		return nil, apierrors.ErrorInternalServerError()
	}
	if artKeys == nil {
		artKeys = []string{}
	}

	return &note.SummaryResponse{ArtKeys: artKeys}, nil
}
