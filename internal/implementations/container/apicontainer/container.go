//go:build wireinject
// +build wireinject

package apicontainer

import (
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/api"
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/repository/db"
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/service"
	"github.com/flazhgrowth/fg-tamagochi/app"
	"github.com/google/wire"
)

func InitAPI(app *app.App) (apis *api.APIs) {
	wire.Build(
		db.DBRepositoryWireSet,
		service.ServiceWireSet,
		api.APIWireSet,
		wire.Struct(new(api.APIs), "*"),
	)

	return
}
