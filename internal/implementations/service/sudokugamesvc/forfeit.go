package sudokugamesvc

import (
	"context"
	"errors"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokugame"
	contextlib "github.com/flazhgrowth/fg-tamagochi/pkg/context"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/apierrors"
)

// Forfeit makes the caller leave a running online game: they lose, and the
// opponent wins with end_reason "forfeit".
func (svc *service) Forfeit(ctx context.Context, args sudokugame.ForfeitRequest) (resp *sudokugame.GameUpdate, err error) {
	logpath := baselogpath.With("Forfeit")

	ctx = contextlib.UseMasterDB(ctx)

	for range maxMoveAttempts {
		resp, err = svc.forfeitOnce(ctx, args)
		if errors.Is(err, errVersionConflict) {
			continue
		}

		return resp, err
	}

	logpath.LogError(ctx, "forfeit kept losing write races", err)
	return nil, apierrors.ErrorInternalServerError()
}

func (svc *service) forfeitOnce(ctx context.Context, args sudokugame.ForfeitRequest) (*sudokugame.GameUpdate, error) {
	logpath := baselogpath.With("forfeitOnce")

	at := now().UTC()
	game, players, err := svc.loadGame(ctx, args.GameID)
	if err != nil {
		return nil, err
	}

	seat, found := seatForToken(players, args.Token)
	if !found {
		return nil, errInvalidToken()
	}

	match, err := sudokugame.NewMatch(game, players)
	if err != nil {
		logpath.With("NewMatch").LogError(ctx, "failed to read game rules", err)
		return nil, apierrors.ErrorInternalServerError()
	}

	loaded := game.Version
	events, ruleErr := match.Forfeit(seat, at)
	if ruleErr == nil {
		if err = svc.persist(ctx, match, loaded, nil); err != nil {
			return nil, err
		}
	}

	gameResp, err := game.ToResponse(match.Players, at)
	if err != nil {
		logpath.With("ToResponse").LogError(ctx, "failed to build game response", err)
		return nil, apierrors.ErrorInternalServerError()
	}

	if ruleErr != nil {
		var violation *sudokugame.RuleError
		if !errors.As(ruleErr, &violation) {
			logpath.With("match.Forfeit").LogError(ctx, "unexpected rules error", ruleErr)
			return nil, apierrors.ErrorInternalServerError()
		}

		return nil, &sudokugame.ConflictError{Code: violation.Code, Message: violation.Message, Game: gameResp}
	}

	update := sudokugame.NewGameUpdate(gameResp, events...)
	svc.hub.publish(game.ID, update)

	return &update, nil
}
