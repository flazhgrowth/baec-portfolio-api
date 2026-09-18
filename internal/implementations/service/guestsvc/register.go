package guestsvc

import (
	"context"
	"database/sql"
	"errors"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/guest"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/model"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/apierrors"
)

func (svc *service) Register(ctx context.Context, args guest.RegisterRequest) (resp *guest.RegisterResponse, err error) {
	logpath := baselogpath.With("Register")

	guestsFound, err := svc.guestRepo.Find(ctx, guest.GuestFilter{VisitID: model.Filter[string]{Valid: true, V: args.VisitID}}, guest.GuestSorter{})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, apierrors.ErrorInternalServerError()
	}
	if len(guestsFound) == 0 {
		guestData := &guest.Guest{Name: args.Name, VisitID: args.VisitID}
		if err = svc.guestRepo.Insert(ctx, guestData); err != nil {
			logpath.With("guestRepo.Insert").LogError(ctx, "failed on inserting new guests", err)
			return nil, apierrors.ErrorInternalServerError()
		}

		return &guest.RegisterResponse{
			ID:        guestData.ID,
			Name:      guestData.Name,
			VisitedAt: now(),
		}, nil
	}

	guestFound := guestsFound[0]
	if err = svc.guestRepo.Update(ctx, guest.GuestFields{Name: sql.NullString{Valid: args.Name != guestFound.Name, String: args.Name}}, guest.GuestFilter{
		ID: model.Filter[uint64]{Valid: true, V: guestFound.ID},
	}); err != nil {
		return nil, apierrors.ErrorInternalServerError()
	}

	return &guest.RegisterResponse{
		ID:        guestFound.ID,
		Name:      args.Name,
		VisitedAt: now(),
	}, nil
}
