package guestroutes

import (
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/guest"
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/api"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/router"
)

var (
	tag = "Guest"
)

func Routes(version router.Router, api *api.APIs) {
	version.Group("/guests", func(guests router.Router) {
		guests.Get("/", api.GuestAPI.ListGuests, &router.RouterDocs{
			Request:     guest.ListGuestsRequest{},
			Response:    guest.ListGuestsResponse{},
			Tags:        tag,
			Title:       "List Guests",
			Description: "List visitors",
		})
		guests.Post("/", api.GuestAPI.Register, &router.RouterDocs{
			Request:     guest.RegisterRequest{},
			Tags:        tag,
			Title:       "Register New Guest",
			Description: "Add new guest to the visitor list",
		})
	})
}
