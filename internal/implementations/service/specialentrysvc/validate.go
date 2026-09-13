package specialentrysvc

import (
	"context"
	"database/sql"
	"errors"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/specialentry"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/model"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/apierrors"
	"github.com/flazhgrowth/fg-tamagochi/pkg/logger"
)

func (svc *service) Validate(ctx context.Context, args specialentry.ValidateRequest) (err error) {
	logpath := baselogpath.With("Validate")

	specialEntryFound, err := svc.specialentryRepo.Get(ctx, specialentry.SpecialEntryFilter{Code: model.Filter[string]{Valid: true, V: args.Token}})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return apierrors.ErrorDataNotFound("invalid token")
		}

		logpath.With("specialentryRepo.Get").LogError(ctx, "failed to get special entry code", err, logger.LogMeta{"token": args.Token})
		return apierrors.ErrorInternalServerError()
	}

	if specialEntryFound.EntryCount+1 > specialEntryFound.MaxEntries {
		return apierrors.ErrorForbidden("token usage exceeded")
	}

	ctx, err = svc.tx.Begin(ctx)
	defer svc.tx.Finish(ctx, &err)
	if err = svc.specialentryRepo.Update(ctx, specialentry.SpecialEntryFields{
		EntryCount: sql.Null[int]{Valid: true, V: specialEntryFound.EntryCount + 1},
	}, specialentry.SpecialEntryFilter{
		ID: model.Filter[uint64]{Valid: true, V: specialEntryFound.ID},
	}); err != nil {
		logpath.With("specialentryRepo.Update").LogError(ctx, "failed on updating special_entry data", err)
		return apierrors.ErrorInternalServerError()
	}

	return nil
}
