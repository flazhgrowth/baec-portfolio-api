package sudokuplayer

import (
	"github.com/Masterminds/squirrel"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/model"
)

type PlayerFilter struct {
	GameID    model.Filter[string]
	Seat      model.Filter[string]
	TokenHash model.Filter[string]
}

func (filter *PlayerFilter) ConditionQuery(builder squirrel.SelectBuilder) squirrel.SelectBuilder {
	builder = filter.GameID.ConditionQuery(builder, "game_id")
	builder = filter.Seat.ConditionQuery(builder, "seat")
	builder = filter.TokenHash.ConditionQuery(builder, "token_hash")

	return builder
}

// PlayerUpdateFields is the rules-driven state of a seat.
type PlayerUpdateFields struct {
	Score              int
	Faults             int
	Mistakes           int
	SkipTurnsRemaining int
}

func (fields *PlayerUpdateFields) UpdateSetQuery(builder squirrel.UpdateBuilder) squirrel.UpdateBuilder {
	return builder.
		Set("score", fields.Score).
		Set("faults", fields.Faults).
		Set("mistakes", fields.Mistakes).
		Set("skip_turns_remaining", fields.SkipTurnsRemaining)
}

// StateFields is the update that persists what the rules engine changed on a seat.
func (datum *Player) StateFields() PlayerUpdateFields {
	return PlayerUpdateFields{
		Score:              datum.Score,
		Faults:             datum.Faults,
		Mistakes:           datum.Mistakes,
		SkipTurnsRemaining: datum.SkipTurnsRemaining,
	}
}
