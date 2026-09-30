package sudokuplayer

import "context"

type Repository interface {
	Insert(ctx context.Context, datum *Player) (err error)
}
