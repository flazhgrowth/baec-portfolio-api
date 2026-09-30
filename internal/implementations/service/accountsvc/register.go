package accountsvc

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/account"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/entity"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/model"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/apierrors"
	"github.com/flazhgrowth/fg-tamagopkg/jwt"
	"github.com/flazhgrowth/fg-tamagopkg/password"
	"github.com/flazhgrowth/fg-tamagopkg/ulid"
)

var (
	getUlid = func() string {
		id, _ := ulid.Generate()
		return id
	}
)

func (svc *service) Register(ctx context.Context, args account.RegisterRequest) (resp *account.LoginResponse, err error) {
	logpath := baselogpath.With("Register")

	// Usernames are stored lowercase: Login looks them up lowercased, and this makes
	// the duplicate check below case-insensitive.
	args.Username = strings.ToLower(args.Username)

	accountFound, err := svc.accountRepo.Get(ctx, account.AccountFilter{
		Username: model.Filter[string]{Valid: true, V: args.Username},
	})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		logpath.With("accountRepo.Get").LogError(ctx, "failed to get data from accounts table", err)
		return nil, apierrors.ErrorInternalServerError()
	}

	if accountFound != nil && accountFound.ID != "" {
		return nil, apierrors.ErrorConflict()
	}

	hashedPwd, salt, err := password.Build(args.Password)
	if err != nil {
		return nil, apierrors.ErrorInternalServerError()
	}

	accountFound = &account.Account{
		BaseULIDModel: entity.BaseULIDModel{
			ID: getUlid(),
		},
		Username: args.Username,
		Password: hashedPwd,
		Name:     args.Username,
		Salt:     salt,
	}
	if err = svc.accountRepo.Insert(ctx, accountFound); err != nil {
		return nil, apierrors.ErrorInternalServerError()
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
