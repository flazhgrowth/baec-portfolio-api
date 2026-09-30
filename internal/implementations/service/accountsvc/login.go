package accountsvc

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/account"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/model"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/apierrors"
	"github.com/flazhgrowth/fg-tamagopkg/jwt"
)

func (svc *service) Login(ctx context.Context, args account.LoginRequest) (resp *account.LoginResponse, err error) {
	logpath := baselogpath.With("Login")

	accountFound, err := svc.accountRepo.Get(ctx, account.AccountFilter{
		Username: model.Filter[string]{Valid: true, V: strings.ToLower(args.Username)},
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apierrors.ErrorUnauthorized().WithCode("invalid_credentials")
		}

		logpath.With("accountRepo.Get").LogError(ctx, "failed to get data from accounts", err)
		return nil, apierrors.ErrorInternalServerError()
	}

	if err = accountFound.ValidatePassword(args.Password); err != nil {
		return nil, err
	}

	token := jwt.NewJWT()
	accessToken, err := token.GenerateToken(jwt.NewClaims(getJwtTTL(), jwt.ClaimsArgs{
		ID:        accountFound.ID,
		Username:  accountFound.Username,
		Firstname: accountFound.Name,
	}), getJwtSecret())
	if err != nil {
		return nil, apierrors.ErrorInternalServerError()
	}
	refreshToken, err := token.GenerateToken(jwt.NewClaims(getJwtRefreshTTL(), jwt.ClaimsArgs{
		ID:        accountFound.ID,
		Username:  accountFound.Username,
		Firstname: accountFound.Name,
	}), getJwtSecret())

	return &account.LoginResponse{
		ID:           accountFound.ID,
		Username:     accountFound.Username,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
