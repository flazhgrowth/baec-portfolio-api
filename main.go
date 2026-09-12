package main

import (
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/routes"
	"github.com/flazhgrowth/fg-tamagochi/cmd"
	"github.com/flazhgrowth/fg-tamagochi/cmd/serve"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/middleware"
	"github.com/go-chi/cors"
)

func main() {
	cmd.Conjure(cmd.CmdArgs{
		ServeCmdArgs: serve.ServeCmdArgs{
			GetRoutesFn: routes.Routes,
			CorsOpts: &middleware.CorsOpt{
				Opts: cors.Options{},
			},
			UseDB: true,
		},
	})
}
