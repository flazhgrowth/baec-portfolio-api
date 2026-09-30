package sudokugamesvc

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokugame"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokuplayer"
	contextlib "github.com/flazhgrowth/fg-tamagochi/pkg/context"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/entity"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/model"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/apierrors"
)

func (svc *service) JoinSession(ctx context.Context, caller entity.AccountInfo, args sudokugame.JoinSessionRequest) (resp *sudokugame.SessionResponse, err error) {
	logpath := baselogpath.With("JoinSession")

	args.Normalize()
	if reason := args.Validate(); reason != "" {
		return nil, invalidRequest(reason)
	}

	// The lobby was written moments ago, possibly through another connection,
	// so read it from the primary rather than a lagging replica.
	ctx = contextlib.UseMasterDB(ctx)

	game, err := svc.gameRepo.Get(ctx, sudokugame.GameFilter{
		JoinCode: model.Filter[string]{Valid: true, V: args.Code},
		Status:   model.Filter[string]{Valid: true, V: sudokugame.StatusWaiting},
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errJoinCodeNotFound()
		}

		logpath.With("gameRepo.Get").LogError(ctx, "failed to get lobby", err)
		return nil, apierrors.ErrorInternalServerError()
	}

	joinedAt := now().UTC()
	if game.LobbyExpired(joinedAt) {
		// Nobody joined in time: the lobby is treated as discarded.
		return nil, errJoinCodeNotFound()
	}

	seated, err := svc.playerRepo.Find(ctx, sudokuplayer.PlayerFilter{
		GameID: model.Filter[string]{Valid: true, V: game.ID},
	})
	if err != nil || len(seated) != 1 || seated[0].Seat != sudokuplayer.SeatOne {
		logpath.With("playerRepo.Find").LogError(ctx, "lobby does not have exactly its host seated", err)
		return nil, apierrors.ErrorInternalServerError()
	}

	token, hash, err := sudokuplayer.NewToken()
	if err != nil {
		logpath.With("NewToken").LogError(ctx, "failed to generate player token", err)
		return nil, apierrors.ErrorInternalServerError()
	}
	guest := sudokuplayer.Player{
		GameID:     game.ID,
		Seat:       sudokuplayer.SeatTwo,
		UserID:     sql.NullString{Valid: true, String: caller.ID},
		Name:       caller.Username,
		TokenHash:  hash,
		Connected:  true,
		LastSeenAt: joinedAt,
	}

	rules := sudokugame.DefaultRules()
	deadline := joinedAt.Add(time.Duration(rules.TurnLimitMs) * time.Millisecond)

	// Commit before announcing: the host must never hear about a join that then rolls back.
	err = func() (err error) {
		txCtx, err := svc.tx.Begin(ctx)
		if err != nil {
			logpath.With("tx.Begin").LogError(ctx, "failed to begin transaction", err)
			return apierrors.ErrorInternalServerError()
		}
		defer svc.tx.Finish(txCtx, &err)

		// The status guard makes this the claim on the seat: if another guest got
		// here first the row no longer matches and we lose the race.
		if err = svc.gameRepo.Update(txCtx, sudokugame.GameUpdateFields{
			Status:           sql.NullString{Valid: true, String: sudokugame.StatusInProgress},
			ClearJoinCode:    true,
			StartedAt:        sql.NullTime{Valid: true, Time: joinedAt},
			TurnPlayerSeat:   sql.NullString{Valid: true, String: sudokuplayer.SeatOne},
			TurnStartedAt:    sql.NullTime{Valid: true, Time: joinedAt},
			TurnDeadlineAt:   sql.NullTime{Valid: true, Time: deadline},
			IncrementVersion: true,
		}, sudokugame.GameFilter{
			ID:     model.Filter[string]{Valid: true, V: game.ID},
			Status: model.Filter[string]{Valid: true, V: sudokugame.StatusWaiting},
		}); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return apierrors.ErrorConflict("the game has already started").WithCode("GAME_FULL")
			}

			logpath.With("gameRepo.Update").LogError(txCtx, "failed to start game", err)
			return apierrors.ErrorInternalServerError()
		}
		if err = svc.playerRepo.Insert(txCtx, &guest); err != nil {
			logpath.With("playerRepo.Insert").LogError(txCtx, "failed to seat guest", err)
			return apierrors.ErrorInternalServerError()
		}

		return nil
	}()
	if err != nil {
		return nil, err
	}

	// Reflect what the update did, so the response matches the stored row.
	game.Status = sudokugame.StatusInProgress
	game.JoinCode = sql.NullString{}
	game.StartedAt = joinedAt
	game.TurnPlayerSeat = sql.NullString{Valid: true, String: sudokuplayer.SeatOne}
	game.TurnStartedAt = sql.NullTime{Valid: true, Time: joinedAt}
	game.TurnDeadlineAt = sql.NullTime{Valid: true, Time: deadline}
	game.Version++

	gameResp, err := game.ToResponse(append(seated, guest), joinedAt)
	if err != nil {
		logpath.With("ToResponse").LogError(ctx, "failed to build game response", err)
		return nil, apierrors.ErrorInternalServerError()
	}

	svc.hub.publish(game.ID, sudokugame.NewGameUpdate(gameResp, sudokugame.GameEvent{
		Type:     sudokugame.EventPlayerJoined,
		PlayerID: guest.Seat,
	}))

	return &sudokugame.SessionResponse{
		Game:        gameResp,
		Credentials: []sudokuplayer.CredentialResponse{{PlayerID: guest.Seat, Token: token}},
	}, nil
}

func errJoinCodeNotFound() apierrors.HTTPError {
	return apierrors.ErrorDataNotFound("no open lobby with that code").WithCode("JOIN_CODE_NOT_FOUND")
}
