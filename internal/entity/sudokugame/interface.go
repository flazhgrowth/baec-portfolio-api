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
	}

	Service interface {
		CreateSession(ctx context.Context, caller entity.AccountInfo, args CreateSessionRequest) (resp *SessionResponse, err error)
	}

	Repository interface {
		Insert(ctx context.Context, datum *Game) (err error)
	}
)
