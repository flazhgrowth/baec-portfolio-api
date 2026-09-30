package sudokugamesvc

import (
	"context"
	"database/sql"
	"errors"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokugame"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokumove"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokuplayer"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/model"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/apierrors"
)

// errVersionConflict means another request changed the game after this one read
// it, so nothing was written and the request should start over from a fresh read.
var errVersionConflict = errors.New("game changed concurrently")

// persist saves what the rules engine changed, all or nothing: the game row, both
// seats, and (for an accepted move) the move log.
//
// The game update is guarded by the version that was loaded, which is what makes
// concurrent requests safe without a row lock: of two requests that read the same
// version, only one update matches, and the other gets errVersionConflict. The
// winner's row lock also holds the loser's update back until the winner commits.
func (svc *service) persist(ctx context.Context, match *sudokugame.Match, loadedVersion int, move *sudokumove.Move) (err error) {
	logpath := baselogpath.With("persist")

	txCtx, err := svc.tx.Begin(ctx)
	if err != nil {
		logpath.With("tx.Begin").LogError(ctx, "failed to begin transaction", err)
		return apierrors.ErrorInternalServerError()
	}
	defer svc.tx.Finish(txCtx, &err)

	game := match.Game
	if err = svc.gameRepo.Update(txCtx, game.StateFields(), sudokugame.GameFilter{
		ID:      model.Filter[string]{Valid: true, V: game.ID},
		Version: model.Filter[int]{Valid: true, V: loadedVersion},
	}); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errVersionConflict
		}

		logpath.With("gameRepo.Update").LogError(ctx, "failed to save game", err)
		return apierrors.ErrorInternalServerError()
	}

	for i := range match.Players {
		player := &match.Players[i]
		if err = svc.playerRepo.Update(txCtx, player.StateFields(), sudokuplayer.PlayerFilter{
			GameID: model.Filter[string]{Valid: true, V: game.ID},
			Seat:   model.Filter[string]{Valid: true, V: player.Seat},
		}); err != nil {
			logpath.With("playerRepo.Update").LogError(ctx, "failed to save player", err)
			return apierrors.ErrorInternalServerError()
		}
	}

	if move != nil {
		move.GameVersion = game.Version
		if err = svc.moveRepo.Insert(txCtx, move); err != nil {
			logpath.With("moveRepo.Insert").LogError(ctx, "failed to log move", err)
			return apierrors.ErrorInternalServerError()
		}
	}

	return nil
}
