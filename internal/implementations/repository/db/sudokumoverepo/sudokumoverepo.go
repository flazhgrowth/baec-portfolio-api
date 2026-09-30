package sudokumoverepo

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokumove"
	"github.com/flazhgrowth/fg-tamagochi/app"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/sqlator"
	"github.com/flazhgrowth/fg-tamagopkg/logger"
)

var (
	baselogpath logger.LogPath = "sudokumoverepo"
)

type repository struct {
	actuator sqlator.SQLator
}

func New(app *app.App) sudokumove.Repository {
	return &repository{
		actuator: app.GetSQLator(),
	}
}

func (repo *repository) Insert(ctx context.Context, datum *sudokumove.Move) (err error) {
	logpath := baselogpath.With("Insert")

	builder := squirrel.
		Insert(sudokumove.MoveTable.Name).
		PlaceholderFormat(squirrel.Dollar)
	builder = datum.InsertValuesQuery(builder)
	query, args, err := builder.ToSql()
	if err != nil {
		logpath.With("builder.ToSql").LogError(ctx, "failed to build query", err)
		return err
	}

	if _, err = repo.actuator.Writer().Write(ctx, query, args); err != nil {
		logpath.With("Write").LogError(ctx, "failed query run", err)
		return err
	}

	return nil
}
