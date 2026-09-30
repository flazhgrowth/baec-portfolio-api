package db

import (
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/account"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/guest"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/msg"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/note"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/specialentry"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokugame"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokumove"
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokuplayer"
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/repository/db/accountrepo"
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/repository/db/guestrepo"
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/repository/db/msgrepo"
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/repository/db/noterepo"
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/repository/db/specialentryrepo"
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/repository/db/sudokugamerepo"
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/repository/db/sudokumoverepo"
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/repository/db/sudokuplayerrepo"
	"github.com/google/wire"
)

var DBRepositoryWireSet = wire.NewSet(
	accountrepo.New,
	guestrepo.New,
	specialentryrepo.New,
	msgrepo.New,
	noterepo.New,
	sudokugamerepo.New,
	sudokuplayerrepo.New,
	sudokumoverepo.New,
)

type DBRepositories struct {
	AccountRepo      account.Repository
	GuestRepo        guest.GuestRepository
	SpecialEntryRepo specialentry.SpecialEntryRepository
	MsgRepo          msg.MsgRepository
	NoteRepo         note.NoteRepository
	SudokuGameRepo   sudokugame.Repository
	SudokuPlayerRepo sudokuplayer.Repository
	SudokuMoveRepo   sudokumove.Repository
}
