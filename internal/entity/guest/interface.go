package guest

import (
	"context"

	"github.com/flazhgrowth/fg-tamagochi/pkg/http/request"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/response"
)

type GuestRepository interface {
	Find(ctx context.Context, filter GuestFilter) (data Guests, err error)
	Count(ctx context.Context, filter GuestFilter) (count int64, err error)
	Insert(ctx context.Context, datum *Guest) (err error)
}

type GuestService interface {
	Register(ctx context.Context, args RegisterRequest) (err error)
	ListGuests(ctx context.Context, args ListGuestsRequest) (resp *ListGuestsResponse, err error)
}

type GuestAPI interface {
	Register(req request.Request, resp response.Response)
	ListGuests(req request.Request, resp response.Response)
}
