package sudokugameapi

import "github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokugame"

type api struct {
	sudokuGameSvc sudokugame.Service
}

func New(sudokugamesvc sudokugame.Service) sudokugame.API {
	return &api{
		sudokuGameSvc: sudokugamesvc,
	}
}
