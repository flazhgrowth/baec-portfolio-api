package sudokugame

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudoku"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokuplayer"
)

// ToResponse assembles the API Game. Players must be in seat order. The solution
// is deliberately not read here, so it cannot leak into any response.
func (datum *Game) ToResponse(players sudokuplayer.Players, serverTime time.Time) (resp GameResponse, err error) {
	puzzle, err := sudoku.GridFromString(datum.Puzzle)
	if err != nil {
		return resp, fmt.Errorf("decode puzzle: %w", err)
	}
	board, err := sudoku.GridFromString(datum.Board)
	if err != nil {
		return resp, fmt.Errorf("decode board: %w", err)
	}

	var rules Rules
	if err = json.Unmarshal([]byte(datum.Rules), &rules); err != nil {
		return resp, fmt.Errorf("decode rules: %w", err)
	}

	resp = GameResponse{
		ID:         datum.ID,
		Mode:       datum.Mode,
		Online:     datum.Online,
		Difficulty: datum.Difficulty,
		Status:     datum.Status,
		Puzzle:     puzzle,
		Board:      board,
		Players:    make([]sudokuplayer.PlayerResponse, 0, len(players)),
		Rules:      rules,
		StartedAt:  datum.StartedAt.UTC(),
		Version:    datum.Version,
		ServerTime: serverTime.UTC(),
	}
	for i := range players {
		resp.Players = append(resp.Players, players[i].ToResponse())
	}

	if datum.JoinCode.Valid {
		resp.JoinCode = &datum.JoinCode.String
	}
	if datum.TurnPlayerSeat.Valid {
		resp.CurrentTurn = &TurnResponse{
			PlayerID:   datum.TurnPlayerSeat.String,
			StartedAt:  datum.TurnStartedAt.Time.UTC(),
			DeadlineAt: datum.TurnDeadlineAt.Time.UTC(),
		}
	}
	if datum.CompletedAt.Valid {
		completedAt := datum.CompletedAt.Time.UTC()
		resp.CompletedAt = &completedAt
	}
	if datum.EndReason.Valid {
		resp.EndReason = &datum.EndReason.String
	}
	if datum.WinnerSeat.Valid {
		resp.WinnerID = &datum.WinnerSeat.String
	}

	return resp, nil
}
