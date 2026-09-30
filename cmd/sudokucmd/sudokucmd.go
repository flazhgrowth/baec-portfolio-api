package sudokucmd

import (
	"fmt"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudoku"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	commands := &cobra.Command{
		Use:   "sudoku",
		Short: "generate sudoku solution",
		Run: func(cmd *cobra.Command, args []string) {
			play()
		},
	}

	return commands
}

func play() {
	// init an empty board
	board := sudoku.NewBoard()
	board.Generate()

	puzzle := board.ToPuzzle()
	puzzle.Solution.PrintSolution()
	fmt.Println("===================")
	puzzle.Puzzle.PrintPuzzle()
	puzzle.Generate(sudoku.DIFF_HARD)
	fmt.Println("===================")
	puzzle.Puzzle.PrintPuzzle()
}
