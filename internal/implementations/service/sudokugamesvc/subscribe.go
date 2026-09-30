package sudokugamesvc

import (
	"context"
	"database/sql"
	"errors"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokugame"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokuplayer"
	contextlib "github.com/flazhgrowth/fg-tamagochi/pkg/context"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/model"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/apierrors"
)

// Subscribe opens a live feed of a game for the seat that owns args.Token.
func (svc *service) Subscribe(ctx context.Context, args sudokugame.StreamEventsRequest) (sub *sudokugame.Subscription, err error) {
	logpath := baselogpath.With("Subscribe")

	ctx = contextlib.UseMasterDB(ctx)

	// A missing game is 404 and a token that does not belong to it is 401. The
	// game is checked first so a wrong id never reads as a bad token.
	if _, _, err = svc.loadGame(ctx, args.GameID); err != nil {
		return nil, err
	}
	if args.Token == "" {
		return nil, errInvalidToken()
	}
	if _, err = svc.playerRepo.Get(ctx, sudokuplayer.PlayerFilter{
		GameID:    model.Filter[string]{Valid: true, V: args.GameID},
		TokenHash: model.Filter[string]{Valid: true, V: sudokuplayer.HashToken(args.Token)},
	}); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errInvalidToken()
		}

		logpath.With("playerRepo.Get").LogError(ctx, "failed to resolve player token", err)
		return nil, apierrors.ErrorInternalServerError()
	}

	// Subscribe before reading the state: anything that changes from here on
	// reaches the channel, and one change that lands in both the snapshot and
	// the channel is harmless because clients drop versions they already hold.
	updates, cancel := svc.hub.subscribe(args.GameID)
	gameResp, err := svc.syncGame(ctx, args.GameID)
	if err != nil {
		cancel()
		return nil, err
	}

	return &sudokugame.Subscription{
		Initial: sudokugame.NewGameUpdate(gameResp),
		Updates: updates,
		Close:   cancel,
	}, nil
}

func errInvalidToken() apierrors.HTTPError {
	return apierrors.ErrorUnauthorized("invalid token").WithCode("INVALID_TOKEN")
}
