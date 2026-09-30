package sudokugamesvc

import (
	"context"
	"errors"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokugame"
	contextlib "github.com/flazhgrowth/fg-tamagochi/pkg/context"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/apierrors"
)

// ExpireTurn ends the current turn if its deadline has passed. It is a fallback
// for clients, since every request touching a game already applies a due
// timeout; calling it early, or twice, therefore changes nothing and answers 409.
func (svc *service) ExpireTurn(ctx context.Context, args sudokugame.ExpireTurnRequest) (resp *sudokugame.GameUpdate, err error) {
	logpath := baselogpath.With("ExpireTurn")

	// The answer depends on the current deadline, so never read a lagging replica.
	ctx = contextlib.UseMasterDB(ctx)

	for range maxMoveAttempts {
		resp, err = svc.expireTurnOnce(ctx, args.GameID)
		if errors.Is(err, errVersionConflict) {
			// Someone changed the game between our read and our write. Read again:
			// if they already expired the turn, this now answers TURN_NOT_EXPIRED.
			continue
		}

		return resp, err
	}

	logpath.LogError(ctx, "expiry kept losing write races", err)
	return nil, apierrors.ErrorInternalServerError()
}

func (svc *service) expireTurnOnce(ctx context.Context, id string) (*sudokugame.GameUpdate, error) {
	logpath := baselogpath.With("expireTurnOnce")

	at := now().UTC()
	game, players, err := svc.loadGame(ctx, id)
	if err != nil {
		return nil, err
	}

	match, err := sudokugame.NewMatch(game, players)
	if err != nil {
		logpath.With("NewMatch").LogError(ctx, "failed to read game rules", err)
		return nil, apierrors.ErrorInternalServerError()
	}

	// refuse answers 409 with the game as it stands right now.
	refuse := func(code, message string) (*sudokugame.GameUpdate, error) {
		gameResp, respErr := game.ToResponse(match.Players, at)
		if respErr != nil {
			logpath.With("ToResponse").LogError(ctx, "failed to build game response", respErr)
			return nil, apierrors.ErrorInternalServerError()
		}

		return nil, &sudokugame.ConflictError{Code: code, Message: message, Game: gameResp}
	}

	switch {
	case game.Status == sudokugame.StatusWaiting:
		return refuse("GAME_NOT_STARTED", "Waiting for a second player")
	case game.Status == sudokugame.StatusCompleted:
		return refuse("GAME_COMPLETED", "Game is already completed")
	case !game.TurnPlayerSeat.Valid:
		return refuse("NO_ACTIVE_TURN", "Single-player games have no turn timer")
	}

	loaded := game.Version
	events := match.ApplyDue(at)
	if len(events) == 0 {
		return refuse("TURN_NOT_EXPIRED", "The current turn has not expired yet")
	}
	if err = svc.persist(ctx, match, loaded, nil); err != nil {
		return nil, err
	}

	gameResp, err := game.ToResponse(match.Players, at)
	if err != nil {
		logpath.With("ToResponse").LogError(ctx, "failed to build game response", err)
		return nil, apierrors.ErrorInternalServerError()
	}

	update := sudokugame.NewGameUpdate(gameResp, events...)
	svc.hub.publish(game.ID, update)

	return &update, nil
}
