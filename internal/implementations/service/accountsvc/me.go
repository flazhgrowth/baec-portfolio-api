package accountsvc

import (
	"context"
	"database/sql"
	"errors"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/account"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/model"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/apierrors"
)

func (svc *service) Me(ctx context.Context, id string) (resp *account.UserResponse, err error) {
	logpath := baselogpath.With("Me")

	found, err := svc.accountRepo.Get(ctx, account.AccountFilter{
		ID: model.Filter[string]{Valid: true, V: id},
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// A valid, unexpired token for an account that is gone: the session is over.
			return nil, apierrors.ErrorUnauthorized("invalid token").WithCode("INVALID_TOKEN")
		}

		logpath.With("accountRepo.Get").LogError(ctx, "failed to get account", err)
		return nil, apierrors.ErrorInternalServerError()
	}

	return &account.UserResponse{
		ID:        found.ID,
		Username:  found.Username,
		CreatedAt: found.CreatedAt.UTC(),
	}, nil
}
