package sudokugamesvc

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudoku"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokugame"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokuplayer"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/entity"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/apierrors"
)

const (
	joinCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	joinCodeLength   = sudokugame.JoinCodeLength
	// Codes are random over 32^6 (~1e9) values, so a collision with another open
	// lobby is rare; a few retries make a failure practically impossible.
	maxJoinCodeAttempts = 5
)

func (svc *service) CreateSession(ctx context.Context, caller entity.AccountInfo, args sudokugame.CreateSessionRequest) (resp *sudokugame.SessionResponse, err error) {
	logpath := baselogpath.With("CreateSession")

	args.Normalize()
	if reason := args.Validate(); reason != "" {
		return nil, invalidRequest(reason)
	}

	board := sudoku.NewBoard()
	board.Generate()
	puzzle := board.ToPuzzle()
	puzzle.Generate(args.Difficulty)

	rules, err := json.Marshal(sudokugame.DefaultRules())
	if err != nil {
		logpath.With("json.Marshal").LogError(ctx, "failed to encode rules", err)
		return nil, apierrors.ErrorInternalServerError()
	}

	for range maxJoinCodeAttempts {
		resp, err = svc.createOnce(ctx, caller, args, puzzle, string(rules))
		if errors.Is(err, sudokugame.ErrJoinCodeTaken) {
			continue
		}

		return resp, err
	}

	logpath.LogError(ctx, "could not find a free join code", err)
	return nil, apierrors.ErrorInternalServerError()
}

// createOnce inserts the game and its seats in one transaction. It returns
// ErrJoinCodeTaken untouched so the caller can retry with a new code.
func (svc *service) createOnce(ctx context.Context, caller entity.AccountInfo, args sudokugame.CreateSessionRequest, puzzle sudoku.Puzzle, rules string) (resp *sudokugame.SessionResponse, err error) {
	logpath := baselogpath.With("createOnce")

	createdAt := now().UTC()
	game := &sudokugame.Game{
		BaseULIDModel: entity.BaseULIDModel{ID: getUlid()},
		Mode:          args.Mode,
		Online:        args.IsOnline,
		Difficulty:    args.Difficulty.Slug,
		Status:        sudokugame.StatusInProgress,
		Puzzle:        puzzle.Puzzle.String(),
		Solution:      puzzle.Solution.String(),
		Board:         puzzle.Puzzle.String(),
		Rules:         rules,
		StartedAt:     createdAt,
		Version:       1, // matches the column default; the response is built before the row is read back
		CreatedBy:     caller.ID,
	}

	players := sudokuplayer.Players{
		{Seat: sudokuplayer.SeatOne, Name: caller.Username, UserID: sql.NullString{Valid: true, String: caller.ID}},
	}
	switch {
	case args.IsOnline:
		// Lobby: host only, waiting for a guest to join with the code.
		code, codeErr := newJoinCode()
		if codeErr != nil {
			logpath.With("newJoinCode").LogError(ctx, "failed to generate join code", codeErr)
			return nil, apierrors.ErrorInternalServerError()
		}
		game.Status = sudokugame.StatusWaiting
		game.JoinCode = sql.NullString{Valid: true, String: code}
	case args.Mode == sudokugame.ModeVersus:
		// Same device: both seats exist from the start and p1 moves first. The
		// second seat is a guest with no account.
		players = append(players, sudokuplayer.Player{Seat: sudokuplayer.SeatTwo, Name: args.GuestName()})
		game.TurnPlayerSeat = sql.NullString{Valid: true, String: sudokuplayer.SeatOne}
		game.TurnStartedAt = sql.NullTime{Valid: true, Time: createdAt}
		game.TurnDeadlineAt = sql.NullTime{Valid: true, Time: createdAt.Add(time.Duration(sudokugame.DefaultRules().TurnLimitMs) * time.Millisecond)}
	}

	credentials := make([]sudokuplayer.CredentialResponse, 0, len(players))
	for i := range players {
		token, hash, tokenErr := sudokuplayer.NewToken()
		if tokenErr != nil {
			logpath.With("NewToken").LogError(ctx, "failed to generate player token", tokenErr)
			return nil, apierrors.ErrorInternalServerError()
		}

		players[i].GameID = game.ID
		players[i].TokenHash = hash
		players[i].Connected = true
		players[i].LastSeenAt = createdAt
		credentials = append(credentials, sudokuplayer.CredentialResponse{PlayerID: players[i].Seat, Token: token})
	}

	gameResp, err := game.ToResponse(players, createdAt)
	if err != nil {
		logpath.With("ToResponse").LogError(ctx, "failed to build game response", err)
		return nil, apierrors.ErrorInternalServerError()
	}

	ctx, err = svc.tx.Begin(ctx)
	if err != nil {
		logpath.With("tx.Begin").LogError(ctx, "failed to begin transaction", err)
		return nil, apierrors.ErrorInternalServerError()
	}
	defer svc.tx.Finish(ctx, &err)

	if err = svc.gameRepo.Insert(ctx, game); err != nil {
		if errors.Is(err, sudokugame.ErrJoinCodeTaken) {
			return nil, err
		}

		logpath.With("gameRepo.Insert").LogError(ctx, "failed to insert game", err)
		return nil, apierrors.ErrorInternalServerError()
	}
	for i := range players {
		if err = svc.playerRepo.Insert(ctx, &players[i]); err != nil {
			logpath.With("playerRepo.Insert").LogError(ctx, "failed to insert player", err)
			return nil, apierrors.ErrorInternalServerError()
		}
	}

	return &sudokugame.SessionResponse{Game: gameResp, Credentials: credentials}, nil
}

func newJoinCode() (string, error) {
	raw := make([]byte, joinCodeLength)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}

	code := make([]byte, joinCodeLength)
	for i, b := range raw {
		// 256 is a multiple of 32, so the modulo is unbiased.
		code[i] = joinCodeAlphabet[int(b)%len(joinCodeAlphabet)]
	}

	return string(code), nil
}
