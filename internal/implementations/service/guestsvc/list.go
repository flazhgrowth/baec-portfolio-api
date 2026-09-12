package guestsvc

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/guest"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/entity"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/model"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/apierrors"
)

func (svc *service) ListGuests(ctx context.Context, args guest.ListGuestsRequest) (resp *guest.ListGuestsResponse, err error) {
	logpath := baselogpath.With("ListGuests")

	var lastSeenID uint64
	sorter := guest.GuestSorter{Sorter: []string{"id DESC"}}
	if args.Cursor != "" {
		lastSeenID, err = strconv.ParseUint(string(args.LastSeenID), 10, 64)
		if err != nil {
			logpath.With("parseCursor").LogError(ctx, "failed on parsing last seen ID", err)
			return nil, apierrors.ErrorBadRequest("invalid cursor")
		}
		sorter = guest.GuestSorter{Sorter: []string{args.Field + " DESC"}}
	}

	guestsFound, err := svc.guestRepo.Find(ctx, guest.GuestFilter{
		ID:   model.Filter[uint64]{Valid: args.Cursor != "", V: lastSeenID, Comparator: model.Lt},
		Size: model.Filter[int]{Valid: args.Size > 0, V: args.Size + 1},
	}, sorter)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &guest.ListGuestsResponse{}, nil
		}

		logpath.With("guestRepo.Find").LogError(ctx, "failed to find guests", err)
		return nil, apierrors.ErrorInternalServerError()
	}
	hasNext := len(guestsFound) > args.Size
	if hasNext {
		guestsFound = guestsFound[:args.Size]
	}

	count, err := svc.guestRepo.Count(ctx, guest.GuestFilter{})
	if err != nil {
		logpath.With("guestRepo.Count").LogError(ctx, "failed to count guests", err)
		return nil, apierrors.ErrorInternalServerError()
	}

	guestsResp := guest.GuestsResponses{}
	for _, guestFound := range guestsFound {
		guestsResp = append(guestsResp, guest.GuestResponse{
			ID:        guestFound.ID,
			Name:      guestFound.Name,
			VisitedAt: guestFound.CreatedAt,
		})
	}

	resp = &guest.ListGuestsResponse{Data: guestsResp}
	resp.Pagination.Calculate("id", entity.LastSeenVal(fmt.Sprintf("%d", guestsResp[len(guestsResp)-1].ID)), int(count), hasNext)
	return resp, nil
}
