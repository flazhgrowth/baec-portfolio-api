package noterepo

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/note"
)

func (repo *repository) Insert(ctx context.Context, datum *note.Note) (err error) {
	builder := squirrel.
		Insert(note.NoteTable.Name).
		PlaceholderFormat(squirrel.Dollar)
	builder = datum.InsertValuesQuery(builder)

	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	if err = repo.db.Writer().Insert(ctx, query, args, &datum.ID); err != nil {
		return err
	}

	return nil
}
