package guestsvc

import (
	"context"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/guest"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/apierrors"
)

func (svc *service) Register(ctx context.Context, args guest.RegisterRequest) (err error) {
	logpath := baselogpath.With("Register")

	guestData := &guest.Guest{Name: args.Name}
	if err = svc.guestRepo.Insert(ctx, guestData); err != nil {
		logpath.With("guestRepo.Insert").LogError(ctx, "failed on inserting new guests", err)
		return apierrors.ErrorInternalServerError()
	}

	return nil
}
