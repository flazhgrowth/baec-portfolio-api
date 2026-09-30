package accountsvc

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/account"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/model"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/apierrors"
	"github.com/flazhgrowth/fg-tamagopkg/hash/argon"
	"github.com/flazhgrowth/fg-tamagopkg/hash/bcrypt"
)

func (svc *service) ChangePassword(ctx context.Context, args account.ChangePasswordRequest) (err error) {
	logpath := baselogpath.With("ChangePassword")

	accountFilter := account.AccountFilter{
		ID: model.Filter[string]{Valid: true, V: args.AccountInfo.ID},
	}
	accountFound, err := svc.accountRepo.Get(ctx, accountFilter)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return apierrors.ErrorDataNotFound("account not found").WithCode("account_not_found")
		}

		return apierrors.ErrorInternalServerError()
	}

	salt, err := base64.RawStdEncoding.DecodeString(accountFound.Salt)
	if err != nil {
		logpath.With("RawStdEncoding.DecodeString").LogError(ctx, "error on decoding salt", err)
		return apierrors.ErrorInternalServerError()
	}
	hashedPassword := argon.Hash(args.Password, []byte(salt))
	hashedPassword, err = bcrypt.Hash(hashedPassword)
	if err != nil {
		logpath.With("bcrypt.Hash").LogError(ctx, "error on hashing password", err)
		return apierrors.ErrorInternalServerError()
	}

	if err = svc.accountRepo.Update(ctx, account.AccountUpdateFields{Password: sql.NullString{Valid: true, String: hashedPassword}}, accountFilter); err != nil {
		logpath.With("accountRepo.Update").LogError(ctx, "error on updating account data", err)
		return apierrors.ErrorInternalServerError()
	}

	return nil
}
