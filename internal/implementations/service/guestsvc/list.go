package guestsvc

import (
	"context"
	"database/sql"
	"errors"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/guest"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/model"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/apierrors"
)

func (svc *service) ListGuests(ctx context.Context, args guest.ListGuestsRequest) (resp *guest.ListGuestsResponse, err error) {
	logpath := baselogpath.With("ListGuests")

	guestsFound, err := svc.guestRepo.Find(ctx, guest.GuestFilter{
		Page: model.Filter[int]{Valid: args.Page > 0, V: args.Page},
		Size: model.Filter[int]{Valid: args.Size > 0, V: args.Size},
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &guest.ListGuestsResponse{}, nil
		}

		logpath.With("guestRepo.Find").LogError(ctx, "failed to find guests", err)
		return nil, apierrors.ErrorInternalServerError()
	}
	count, err := svc.guestRepo.Count(ctx, guest.GuestFilter{})
	if err != nil {
		logpath.With("guestRepo.Count").LogError(ctx, "failed to count guests", err)
		return nil, apierrors.ErrorInternalServerError()
	}

	guestsResp := guest.GuestsResponses{}
	for _, guestFound := range guestsFound {
		guestsResp = append(guestsResp, guest.GuestResponse{
			Name:      guestFound.Name,
			VisitedAt: guestFound.CreatedAt,
		})
	}

	return &guest.ListGuestsResponse{
		Data:       guestsResp,
		Pagination: args.Calculate(int(count)),
	}, nil
}
