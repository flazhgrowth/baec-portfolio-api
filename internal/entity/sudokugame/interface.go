package sudokugame

import (
	"context"

	"github.com/flazhgrowth/fg-tamagochi/pkg/db/entity"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/request"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/response"
)

type (
	API interface {
		CreateSession(req request.Request, resp response.Response)
		JoinSession(req request.Request, resp response.Response)
		GetGame(req request.Request, resp response.Response)
		StreamEvents(req request.Request, resp response.Response)
		SubmitMove(req request.Request, resp response.Response)
		ExpireTurn(req request.Request, resp response.Response)
		Forfeit(req request.Request, resp response.Response)
	}

	Service interface {
		CreateSession(ctx context.Context, caller entity.AccountInfo, args CreateSessionRequest) (resp *SessionResponse, err error)
		JoinSession(ctx context.Context, caller entity.AccountInfo, args JoinSessionRequest) (resp *SessionResponse, err error)
		GetGame(ctx context.Context, args GetGameRequest) (resp *GameResponse, err error)
		Subscribe(ctx context.Context, args StreamEventsRequest) (sub *Subscription, err error)
		// SubmitMove returns *ConflictError (HTTP 409 with the game) for rule violations.
		SubmitMove(ctx context.Context, args MoveRequest) (resp *MoveResponse, err error)
		// ExpireTurn returns *ConflictError (HTTP 409 with the game) when there is nothing to expire.
		ExpireTurn(ctx context.Context, args ExpireTurnRequest) (resp *GameUpdate, err error)
		// Forfeit returns *ConflictError (HTTP 409 with the game) when the game cannot be forfeited.
		Forfeit(ctx context.Context, args ForfeitRequest) (resp *GameUpdate, err error)
	}

	Repository interface {
		Get(ctx context.Context, filter GameFilter) (datum *Game, err error)
		Insert(ctx context.Context, datum *Game) (err error)
		// Update returns sql.ErrNoRows when no row matches the filter, which makes
		// a filtered update usable as an atomic claim.
		Update(ctx context.Context, fields GameUpdateFields, filter GameFilter) (err error)
	}
)
