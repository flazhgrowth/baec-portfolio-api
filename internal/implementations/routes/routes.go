package routes

import (
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/container/apicontainer"
	"github.com/flazhgrowth/fg-tamagochi/app"
	"github.com/flazhgrowth/fg-tamagochi/pkg/config"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/router"
)

func Routes(app *app.App) router.Router {
	rtr := router.NewRouter(router.OpenAPISpecInfo{
		Title:   "baec-portfolio-api",
		Version: "v0.0.0",
		Desc:    "API Documentations for baec-portfolio-api",
	})

	apis := apicontainer.InitAPI(app)

	rtr.Group("/api/v1", func(version router.Router) {
		v1(version, apis)
	})
	if !config.GetConfig().IsEnvProduction() {
		rtr.ServeDocs()
		rtr.ServeProfiler()
	}

	return rtr
}
