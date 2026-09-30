package sudokugame

import (
	"database/sql"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/model"
)

type (
	GameFilter struct {
		ID     model.Filter[string]
		Status model.Filter[string]
		// Version makes an update optimistic: it only applies if nobody changed the game since it was read.
		Version model.Filter[int]
		// JoinCode must be upper-case. It matches on upper(join_code) so the
		// partial unique index on open lobbies serves the lookup.
		JoinCode model.Filter[string]
	}
	GameUpdateFields struct {
		Status         sql.NullString
		ClearJoinCode  bool
		StartedAt      sql.NullTime
		TurnPlayerSeat sql.NullString
		TurnStartedAt  sql.NullTime
		TurnDeadlineAt sql.NullTime
		Board          sql.NullString
		// ClearTurn nulls the three turn columns; a game that is waiting or finished has no turn.
		ClearTurn   bool
		CompletedAt sql.NullTime
		EndReason   sql.NullString
		WinnerSeat  sql.NullString
		// IncrementVersion bumps version by one, done in SQL so it is atomic.
		IncrementVersion bool
	}
)

func (filter *GameFilter) ConditionQuery(builder squirrel.SelectBuilder) squirrel.SelectBuilder {
	builder = filter.ID.ConditionQuery(builder, "id")
	builder = filter.Status.ConditionQuery(builder, "status")
	builder = filter.Version.ConditionQuery(builder, "version")
	builder = filter.JoinCode.ConditionQuery(builder, "upper(join_code)")

	return builder
}

func (fields *GameUpdateFields) UpdateSetQuery(builder squirrel.UpdateBuilder) squirrel.UpdateBuilder {
	if fields.Status.Valid {
		builder = builder.Set("status", fields.Status.String)
	}
	if fields.ClearJoinCode {
		builder = builder.Set("join_code", nil)
	}
	if fields.StartedAt.Valid {
		builder = builder.Set("started_at", fields.StartedAt.Time)
	}
	if fields.TurnPlayerSeat.Valid {
		builder = builder.Set("turn_player_seat", fields.TurnPlayerSeat.String)
	}
	if fields.TurnStartedAt.Valid {
		builder = builder.Set("turn_started_at", fields.TurnStartedAt.Time)
	}
	if fields.TurnDeadlineAt.Valid {
		builder = builder.Set("turn_deadline_at", fields.TurnDeadlineAt.Time)
	}
	if fields.Board.Valid {
		builder = builder.Set("board", fields.Board.String)
	}
	if fields.ClearTurn {
		builder = builder.Set("turn_player_seat", nil).Set("turn_started_at", nil).Set("turn_deadline_at", nil)
	}
	if fields.CompletedAt.Valid {
		builder = builder.Set("completed_at", fields.CompletedAt.Time)
	}
	if fields.EndReason.Valid {
		builder = builder.Set("end_reason", fields.EndReason.String)
	}
	if fields.WinnerSeat.Valid {
		builder = builder.Set("winner_seat", fields.WinnerSeat.String)
	}
	if fields.IncrementVersion {
		builder = builder.Set("version", squirrel.Expr("version + 1"))
	}

	return builder.Set("updated_at", time.Now())
}

// StateFields is the update that persists the game's mutable state after the
// rules engine has changed it in memory: status, board, turn, completion, and a
// version bump. The caller guards it with the version it loaded.
func (datum *Game) StateFields() GameUpdateFields {
	fields := GameUpdateFields{
		Status:           sql.NullString{Valid: true, String: datum.Status},
		Board:            sql.NullString{Valid: true, String: datum.Board},
		IncrementVersion: true,
	}

	if datum.TurnPlayerSeat.Valid {
		fields.TurnPlayerSeat = datum.TurnPlayerSeat
		fields.TurnStartedAt = datum.TurnStartedAt
		fields.TurnDeadlineAt = datum.TurnDeadlineAt
	} else {
		fields.ClearTurn = true
	}
	if datum.CompletedAt.Valid {
		fields.CompletedAt = datum.CompletedAt
		fields.EndReason = datum.EndReason
		fields.WinnerSeat = datum.WinnerSeat
	}

	return fields
}
