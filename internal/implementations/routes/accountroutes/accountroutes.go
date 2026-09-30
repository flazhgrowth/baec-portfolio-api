package accountroutes

import (
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/account"
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/api"
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/middleware"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/router"
)

var (
	tag = "Account"
)

func Routes(version router.Router, apis *api.APIs) {
	version.Group("/auth", func(group router.Router) {
		// public
		group.Scope(func(public router.Router) {
			public.Post("/register", apis.AccountAPI.Register, &router.RouterDocs{
				Request:     account.RegisterRequest{},
				Response:    account.LoginResponse{},
				Tags:        tag,
				Title:       "Register New Account",
				Description: "Register new account",
			})
			public.Post("/login", apis.AccountAPI.Login, &router.RouterDocs{
				Request:     account.LoginRequest{},
				Response:    account.LoginResponse{},
				Tags:        tag,
				Title:       "Login",
				Description: "Login for existing account",
			})
		})

		// auth
		group.Scope(func(auth router.Router) {
			auth.Use(middleware.MIDDLEWARE_AUTH)
			auth.Post("/me", apis.AccountAPI.Me, &router.RouterDocs{
				Security:    router.SecAuths{router.SecurityBearerAuth},
				Response:    account.MeResponse{},
				Tags:        tag,
				Title:       "Me",
				Description: "Get authenticated account info",
			})
		})
	})
}
