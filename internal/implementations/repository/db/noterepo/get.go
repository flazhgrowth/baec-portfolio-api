package noterepo

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/note"
	"github.com/flazhgrowth/fg-tamagochi/pkg/logger"
)

func (repo *repository) Find(ctx context.Context, filter note.NoteFilter, sorter note.NoteSorter) (data note.NotesWithGuest, err error) {
	logpath := baselogpath.With("Find")

	builder := squirrel.
		Select(note.NoteTable.SelectColumns...).
		From(note.NoteTable.Name).
		Join("guests ON guests.visit_id = notes.visit_id").
		PlaceholderFormat(squirrel.Dollar)
	builder = filter.ConditionQuery(builder)
	builder = sorter.SortQuery(builder)

	query, args, err := builder.ToSql()
	if err != nil {
		logpath.With("builder.ToSql").LogError(ctx, "failed to build query", err)
		return nil, err
	}

	logpath.LogDebug(ctx, "executing query..", logger.LogMeta{"query": query, "args": args})
	data = note.NotesWithGuest{}
	if err = repo.db.Reader().Find(ctx, query, args, &data); err != nil {
		logpath.With("Find").LogError(ctx, "failed query run", err)
		return nil, err
	}

	return data, nil
}

func (repo *repository) Count(ctx context.Context, filter note.NoteFilter) (count int64, err error) {
	logpath := baselogpath.With("Count")

	builder := squirrel.
		Select(note.NoteTable.CountColumns...).
		From(note.NoteTable.Name).
		PlaceholderFormat(squirrel.Dollar)
	builder = filter.ConditionQuery(builder)

	query, args, err := builder.ToSql()
	if err != nil {
		logpath.With("builder.ToSql").LogError(ctx, "failed to build query", err)
		return 0, err
	}

	logpath.LogDebug(ctx, "executing query..", logger.LogMeta{"query": query, "args": args})
	if err = repo.db.Reader().Get(ctx, query, args, &count); err != nil {
		logpath.With("Get").LogError(ctx, "failed query run", err)
		return 0, err
	}

	return count, nil
}

func (repo *repository) ListArtKeys(ctx context.Context) (artKeys []string, err error) {
	logpath := baselogpath.With("ListArtKeys")

	query, args, err := squirrel.
		Select("art_key").
		Distinct().
		From(note.NoteTable.Name).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		logpath.With("builder.ToSql").LogError(ctx, "failed to build query", err)
		return nil, err
	}

	logpath.LogDebug(ctx, "executing query..", logger.LogMeta{"query": query, "args": args})
	artKeys = []string{}
	res := note.Notes{}
	if err = repo.db.Reader().Find(ctx, query, args, &res); err != nil {
		logpath.With("Find").LogError(ctx, "failed query run", err)
		return nil, err
	}
	for _, r := range res {
		artKeys = append(artKeys, r.ArtKey)
	}

	return artKeys, nil
}
