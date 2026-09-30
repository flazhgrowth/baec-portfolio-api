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

// GetGame returns the current state of a game, first applying anything that is
// already due. Today that is turn expiry only: disconnect forfeits need presence
// tracking on top of the event stream, which is not built yet.
func (svc *service) GetGame(ctx context.Context, args sudokugame.GetGameRequest) (resp *sudokugame.GameResponse, err error) {
	// This may write (expiry), and callers resume a game right after acting on
	// it, so read the primary rather than a lagging replica.
	ctx = contextlib.UseMasterDB(ctx)

	gameResp, err := svc.syncGame(ctx, args.ID)
	if err != nil {
		return nil, err
	}

	return &gameResp, nil
}

// syncGame loads a game, applies whatever is due and returns its current state.
// When it changes the game it also publishes the change, so the other device
// hears about a timeout even if only this request noticed it.
func (svc *service) syncGame(ctx context.Context, id string) (gameResp sudokugame.GameResponse, err error) {
	logpath := baselogpath.With("syncGame")

	at := now().UTC()
	game, players, err := svc.loadGame(ctx, id)
	if err != nil {
		return gameResp, err
	}

	var events []sudokugame.GameEvent
	if game.TurnDue(at) {
		match, matchErr := sudokugame.NewMatch(game, players)
		if matchErr != nil {
			logpath.With("NewMatch").LogError(ctx, "failed to read game rules", matchErr)
			return gameResp, apierrors.ErrorInternalServerError()
		}

		loaded := game.Version
		events = match.ApplyDue(at)
		switch persistErr := svc.persist(ctx, match, loaded, nil); {
		case persistErr == nil:
		case errors.Is(persistErr, errVersionConflict):
			// Someone else changed the game first (another request, or a move).
			// Their write is authoritative and they publish it, so serve that
			// state as is.
			events = nil
			if game, players, err = svc.loadGame(ctx, id); err != nil {
				return gameResp, err
			}
		default:
			return gameResp, persistErr
		}
	}

	gameResp, err = game.ToResponse(players, at)
	if err != nil {
		logpath.With("ToResponse").LogError(ctx, "failed to build game response", err)
		return gameResp, apierrors.ErrorInternalServerError()
	}
	if len(events) > 0 {
		svc.hub.publish(game.ID, sudokugame.NewGameUpdate(gameResp, events...))
	}

	return gameResp, nil
}

// loadGame fetches a game and its seats. A lobby nobody joined in time counts as
// discarded, so it is reported as missing.
func (svc *service) loadGame(ctx context.Context, id string) (*sudokugame.Game, sudokuplayer.Players, error) {
	logpath := baselogpath.With("loadGame")

	game, err := svc.gameRepo.Get(ctx, sudokugame.GameFilter{ID: model.Filter[string]{Valid: true, V: id}})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, errGameNotFound()
		}

		logpath.With("gameRepo.Get").LogError(ctx, "failed to get game", err)
		return nil, nil, apierrors.ErrorInternalServerError()
	}
	if game.LobbyExpired(now().UTC()) {
		return nil, nil, errGameNotFound()
	}

	players, err := svc.playerRepo.Find(ctx, sudokuplayer.PlayerFilter{GameID: model.Filter[string]{Valid: true, V: game.ID}})
	if err != nil {
		logpath.With("playerRepo.Find").LogError(ctx, "failed to get players", err)
		return nil, nil, apierrors.ErrorInternalServerError()
	}

	return game, players, nil
}

func errGameNotFound() apierrors.HTTPError {
	return apierrors.ErrorDataNotFound("game not found").WithCode("GAME_NOT_FOUND")
}
