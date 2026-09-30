package sudokuplayer

import "context"

type Repository interface {
	// Get returns sql.ErrNoRows when no seat matches.
	Get(ctx context.Context, filter PlayerFilter) (datum *Player, err error)
	Find(ctx context.Context, filter PlayerFilter) (data Players, err error)
	Insert(ctx context.Context, datum *Player) (err error)
	Update(ctx context.Context, fields PlayerUpdateFields, filter PlayerFilter) (err error)
}
