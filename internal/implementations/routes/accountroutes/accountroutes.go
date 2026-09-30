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
			// Stateless JWTs leave nothing to invalidate, so this is a public no-op that always answers 204.
			public.Post("/logout", apis.AccountAPI.Logout, &router.RouterDocs{
				Tags:        tag,
				Title:       "Logout",
				Description: "Log out. A no-op: tokens are stateless, the client discards its token. Always 204",
			})
		})

		// auth
		group.Scope(func(auth router.Router) {
			auth.Use(middleware.MIDDLEWARE_AUTH)
			auth.Get("/me", apis.AccountAPI.Me, &router.RouterDocs{
				Security:    router.SecAuths{router.SecurityBearerAuth},
				Response:    account.UserResponse{},
				Tags:        tag,
				Title:       "Me",
				Description: "Resolve the account behind the bearer token (restore a session)",
			})
			auth.Put("/password", apis.AccountAPI.ChangePassword, &router.RouterDocs{
				Security:    router.SecAuths{router.SecurityBearerAuth},
				Request:     account.ChangePasswordRequest{},
				Tags:        tag,
				Title:       "Change Password",
				Description: "Change password for authenticated user",
			})
		})
	})
}
