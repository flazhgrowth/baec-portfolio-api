package sudokugamesvc

import (
	"context"
	"database/sql"
	"errors"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokugame"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokumove"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokuplayer"
	contextlib "github.com/flazhgrowth/fg-tamagochi/pkg/context"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/apierrors"
)

// maxMoveAttempts bounds how often a move restarts after losing a write race.
// Two players on one game make a second consecutive loss very unlikely.
const maxMoveAttempts = 5

// SubmitMove plays one move for the seat that owns the token.
func (svc *service) SubmitMove(ctx context.Context, args sudokugame.MoveRequest) (resp *sudokugame.MoveResponse, err error) {
	logpath := baselogpath.With("SubmitMove")

	// The state read here decides the outcome, so it must not come from a lagging replica.
	ctx = contextlib.UseMasterDB(ctx)

	for range maxMoveAttempts {
		resp, err = svc.submitMoveOnce(ctx, args)
		if errors.Is(err, errVersionConflict) {
			continue
		}

		return resp, err
	}

	logpath.LogError(ctx, "move kept losing write races", err)
	return nil, apierrors.ErrorInternalServerError()
}

func (svc *service) submitMoveOnce(ctx context.Context, args sudokugame.MoveRequest) (*sudokugame.MoveResponse, error) {
	logpath := baselogpath.With("submitMoveOnce")

	at := now().UTC()
	game, players, err := svc.loadGame(ctx, args.GameID)
	if err != nil {
		return nil, err
	}

	seat, found := seatForToken(players, args.Token)
	if !found {
		return nil, errInvalidToken()
	}

	row, col, value, reason := args.Ints()
	if reason != "" {
		return nil, invalidRequest(reason)
	}

	match, err := sudokugame.NewMatch(game, players)
	if err != nil {
		logpath.With("NewMatch").LogError(ctx, "failed to read game rules", err)
		return nil, apierrors.ErrorInternalServerError()
	}

	loaded := game.Version
	outcome, events, ruleErr := match.SubmitMove(seat, row, col, value, at)

	var ruleViolation *sudokugame.RuleError
	if ruleErr != nil && !errors.As(ruleErr, &ruleViolation) {
		logpath.With("match.SubmitMove").LogError(ctx, "unexpected rules error", ruleErr)
		return nil, apierrors.ErrorInternalServerError()
	}
	if ruleViolation != nil && ruleViolation.Status == 422 {
		return nil, invalidRequest(ruleViolation.Message)
	}

	// State changed if the move was accepted, or if a late move triggered a
	// timeout before being rejected. That timeout is saved either way; only then
	// is the move refused, so the caller sees the new state rather than the old.
	if ruleViolation == nil || len(events) > 0 {
		var move *sudokumove.Move
		if ruleViolation == nil {
			move = moveRecord(game.ID, outcome)
		}
		if err = svc.persist(ctx, match, loaded, move); err != nil {
			return nil, err
		}
	}

	gameResp, err := game.ToResponse(match.Players, at)
	if err != nil {
		logpath.With("ToResponse").LogError(ctx, "failed to build game response", err)
		return nil, apierrors.ErrorInternalServerError()
	}
	if len(events) > 0 {
		svc.hub.publish(game.ID, sudokugame.NewGameUpdate(gameResp, events...))
	}

	if ruleViolation != nil {
		return nil, &sudokugame.ConflictError{Code: ruleViolation.Code, Message: ruleViolation.Message, Game: gameResp}
	}

	return &sudokugame.MoveResponse{
		Result:    outcome.Result(),
		Points:    outcome.Points,
		ElapsedMs: outcome.ElapsedMs,
		Game:      gameResp,
		Events:    events,
	}, nil
}

// seatForToken finds the seat whose token hash matches.
func seatForToken(players sudokuplayer.Players, token string) (seat string, found bool) {
	if token == "" {
		return "", false
	}

	hash := sudokuplayer.HashToken(token)
	for i := range players {
		if players[i].TokenHash == hash {
			return players[i].Seat, true
		}
	}

	return "", false
}

func moveRecord(gameID string, outcome sudokugame.MoveOutcome) *sudokumove.Move {
	move := &sudokumove.Move{
		GameID:  gameID,
		Seat:    outcome.Seat,
		Row:     outcome.Row,
		Col:     outcome.Col,
		Value:   outcome.Value,
		Correct: outcome.Correct,
		Points:  outcome.Points,
	}
	if outcome.ElapsedMs != nil {
		move.ElapsedMs = sql.NullInt64{Valid: true, Int64: int64(*outcome.ElapsedMs)}
	}

	return move
}

func invalidRequest(reason string) apierrors.HTTPError {
	return apierrors.ErrorUnprocessableEntity(reason).WithCode("VALIDATION_ERROR")
}
