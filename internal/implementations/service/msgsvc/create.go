package msgsvc

import (
	"context"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/guest"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/msg"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/model"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/apierrors"
)

func (svc *service) Create(ctx context.Context, args msg.CreateRequest) (err error) {
	count, err := svc.guestRepo.Count(ctx, guest.GuestFilter{
		VisitID: model.Filter[string]{Valid: true, V: args.VisitID},
	})
	if err != nil {
		return apierrors.ErrorInternalServerError()
	}
	if count == 0 {
		return apierrors.ErrorDataNotFound("invalid visit_id")
	}

	msgData := &msg.Msg{
		VisitID: args.VisitID,
		Message: args.Msg,
	}
	if err = svc.msgRepo.Insert(ctx, msgData); err != nil {
		return apierrors.ErrorBadRequest("visit_id already sent one message earlier")
	}

	return nil
}
