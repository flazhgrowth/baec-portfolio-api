package sudokugameroutes

import (
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokugame"
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/api"
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/middleware"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/router"
)

var (
	tag = "Sudoku Game"
)

func Routes(version router.Router, apis *api.APIs) {
	version.Group("/games", func(group router.Router) {
		group.Scope(func(auth router.Router) {
			auth.Use(middleware.MIDDLEWARE_AUTH)
			auth.Post("/", apis.SudokuGameAPI.CreateSession, &router.RouterDocs{
				Security:    router.SecAuths{router.SecurityBearerAuth},
				Request:     sudokugame.CreateSessionRequest{},
				Response:    sudokugame.SessionResponse{},
				Tags:        tag,
				Title:       "Create Game",
				Description: "Start a single / same-device versus game, or open an online lobby",
			})
		})
	})
}
