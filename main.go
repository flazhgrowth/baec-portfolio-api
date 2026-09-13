package main

import (
	"net/http"

	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/middleware"
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/routes"
	"github.com/flazhgrowth/fg-tamagochi/cmd"
	"github.com/flazhgrowth/fg-tamagochi/cmd/serve"
	libmw "github.com/flazhgrowth/fg-tamagochi/pkg/http/middleware"
	"github.com/go-chi/cors"
)

func main() {
	cmd.Conjure(cmd.CmdArgs{
		ServeCmdArgs: serve.ServeCmdArgs{
			GetRoutesFn: routes.Routes,
			CorsOpts: &libmw.CorsOpt{
				Opts: cors.Options{
					AllowedOrigins:   []string{"*"},
					AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
					AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "Content-Length", "X-CSRF-Token", "Accept-Encoding", "X-Callback-Token", "X-API-Key", "X-API-Version", "Idempotency-Key"},
					ExposedHeaders:   []string{"Link"},
					AllowCredentials: false,
					MaxAge:           300, // Maximum value not ignored by any of major browsers
				},
			},
			Middlewares: map[libmw.HTTPMiddleware]func(next http.Handler) http.Handler{
				middleware.MIDDLEWARE_CMS_KEY: libmw.BasicAPIKeyMiddleware("cms"),
			},
			UseDB: true,
		},
	})
}
