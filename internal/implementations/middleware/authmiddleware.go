package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/api/contractresp"
	"github.com/flazhgrowth/fg-tamagochi/constant"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/entity"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/apierrors"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/request"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/response"
	"github.com/flazhgrowth/fg-tamagochi/pkg/vault"
	"github.com/flazhgrowth/fg-tamagopkg/jwt"
)

// errInvalidToken is what the contract answers for a missing, malformed, forged or
// expired account token alike, so a client cannot tell them apart.
var errInvalidToken = apierrors.ErrorUnauthorized("Missing or invalid account token").WithCode("INVALID_TOKEN")

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := request.New(r)
		resp := response.New(w)
		secHeaders := req.SecurityHeaders()
		if !secHeaders.IsAuth {
			contractresp.Error(resp, errInvalidToken)
			return
		}

		token := jwt.NewJWT()
		claims, err := token.ValidateToken(secHeaders.Authorization, vault.GetVault().GetStringWithDefault("secret.jwt", ""))
		if err != nil {
			contractresp.Error(resp, errInvalidToken)
			return
		}
		if claims.ExpiresAt.Before(time.Now()) || claims.ID == "" {
			contractresp.Error(resp, errInvalidToken)
			return
		}
		ctx := r.Context()
		ctx = context.WithValue(ctx, constant.CtxKeyAccountInfo, entity.AccountInfo{
			ID:        claims.ID,
			Username:  claims.Username,
			Firstname: claims.Firstname,
		})

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
