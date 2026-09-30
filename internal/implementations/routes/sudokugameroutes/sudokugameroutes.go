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
		// public: the contract resumes a game by id alone. The response never
		// carries the solution or any token.
		group.Scope(func(public router.Router) {
			public.Get("/{gameId}", apis.SudokuGameAPI.GetGame, &router.RouterDocs{
				Request:     sudokugame.GetGameRequest{},
				Response:    sudokugame.GameResponse{},
				Tags:        tag,
				Title:       "Get Game",
				Description: "Fetch the current state of a game, applying any due turn expiry first",
			})
			// Public on purpose: a fallback for the server's own timer, it only acts once the deadline has passed.
			public.Post("/{gameId}/turn/expire", apis.SudokuGameAPI.ExpireTurn, &router.RouterDocs{
				Response:    sudokugame.GameUpdate{},
				Tags:        tag,
				Title:       "Expire Turn",
				Description: "Report that the current turn ran out of time. Does nothing before the deadline (409 TURN_NOT_EXPIRED)",
			})
			// The player token travels as ?token= because browsers cannot set headers on EventSource.
			public.Get("/{gameId}/events", apis.SudokuGameAPI.StreamEvents, &router.RouterDocs{
				Request:     sudokugame.StreamEventsRequest{},
				Tags:        tag,
				Title:       "Stream Game Events",
				Description: "Server-Sent Events: a full `update` on connect, then one per state change. Takes the player token as ?token=",
			})
		})

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
			auth.Post("/join", apis.SudokuGameAPI.JoinSession, &router.RouterDocs{
				Security:    router.SecAuths{router.SecurityBearerAuth},
				Request:     sudokugame.JoinSessionRequest{},
				Response:    sudokugame.SessionResponse{},
				Tags:        tag,
				Title:       "Join Game",
				Description: "Join an online lobby by its join code",
			})
		})

		// The seat is identified by the X-Player-Token header, not the account JWT.
		group.Scope(func(player router.Router) {
			player.Post("/{gameId}/moves", apis.SudokuGameAPI.SubmitMove, &router.RouterDocs{
				Request:     sudokugame.MoveRequest{},
				Response:    sudokugame.MoveResponse{},
				Tags:        tag,
				Title:       "Submit Move",
				Description: "Fill one box. A wrong number is a normal 200 with result 'incorrect'. Needs the X-Player-Token header",
			})
			player.Post("/{gameId}/forfeit", apis.SudokuGameAPI.Forfeit, &router.RouterDocs{
				Response:    sudokugame.GameUpdate{},
				Tags:        tag,
				Title:       "Forfeit Game",
				Description: "Leave a running online game: the caller loses. Needs the X-Player-Token header",
			})
		})
	})
}
