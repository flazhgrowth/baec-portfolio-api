package notesvc

import (
	"context"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/guest"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/note"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/model"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/apierrors"
)

func errAlreadyNoted() apierrors.HTTPError {
	return apierrors.HTTPError{
		StatusCode: http.StatusConflict,
		Code:       "conflict",
		Message:    "visit_id already left a note on this artwork",
	}
}

func (svc *service) Create(ctx context.Context, args note.CreateRequest) (err error) {
	logpath := baselogpath.With("Create")

	args.Note = strings.TrimSpace(args.Note)
	if !note.ArtKey(args.ArtKey).Valid() {
		return apierrors.ErrorBadRequest("invalid art_key")
	}
	if args.Note == "" {
		return apierrors.ErrorBadRequest("note must not be empty")
	}
	if utf8.RuneCountInString(args.Note) > note.MaxNoteLength {
		return apierrors.ErrorBadRequest("note must not exceed 240 characters")
	}

	guestCount, err := svc.guestRepo.Count(ctx, guest.GuestFilter{
		VisitID: model.Filter[string]{Valid: true, V: args.VisitID},
	})
	if err != nil {
		logpath.With("guestRepo.Count").LogError(ctx, "failed to count guests", err)
		return apierrors.ErrorInternalServerError()
	}
	if guestCount == 0 {
		return apierrors.ErrorDataNotFound("invalid visit_id")
	}

	sameNoteFilter := note.NoteFilter{
		VisitID: model.Filter[string]{Valid: true, V: args.VisitID},
		ArtKey:  model.Filter[string]{Valid: true, V: args.ArtKey},
	}
	noteCount, err := svc.noteRepo.Count(ctx, sameNoteFilter)
	if err != nil {
		logpath.With("noteRepo.Count").LogError(ctx, "failed to count notes", err)
		return apierrors.ErrorInternalServerError()
	}
	if noteCount > 0 {
		return errAlreadyNoted()
	}

	if err = svc.noteRepo.Insert(ctx, &note.Note{
		VisitID: args.VisitID,
		ArtKey:  args.ArtKey,
		Note:    args.Note,
	}); err != nil {
		// A concurrent request may have won the race against the unique (visit_id, art_key) index.
		if noteCount, countErr := svc.noteRepo.Count(ctx, sameNoteFilter); countErr == nil && noteCount > 0 {
			return errAlreadyNoted()
		}

		logpath.With("noteRepo.Insert").LogError(ctx, "failed to insert note", err)
		return apierrors.ErrorInternalServerError()
	}

	return nil
}
