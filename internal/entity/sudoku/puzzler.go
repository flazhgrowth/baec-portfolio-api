package sudoku

import (
	"math/rand/v2"
)

func (board BoardSpace) ToPuzzle() Puzzle {
	return Puzzle{
		Solution: board,
		Puzzle:   PuzzleSpace(board.clone()),
	}
}

// clone deep-copies the board. Rows are slices, so a plain maps.Clone would
// share them and carving the puzzle would blank the solution too.
func (board BoardSpace) clone() BoardSpace {
	cloned := make(BoardSpace, len(board))
	for row, cells := range board {
		cloned[row] = append([]int(nil), cells...)
	}

	return cloned
}

func (puzzle *Puzzle) Generate(difficulty Difficulty) {
	puzzle.Puzzle.generatePuzzle(difficulty)
}

func (board PuzzleSpace) PrintPuzzle() {
	BoardSpace(board).PrintSolution()
}

// validForRemoval reports whether the board still has a unique solution
// (i.e. fewer than 2). The board is left unchanged.
func (board PuzzleSpace) validForRemoval() bool {
	return board.countSolutions(0, 2) < 2
}

// countSolutions counts the solutions reachable from pos by backtracking,
// stopping early once limit is reached. Cells are restored on return.
func (board PuzzleSpace) countSolutions(pos, limit int) int {
	for pos < 81 && !board.cellIsEmpty(pos/9, pos%9) {
		pos++
	}

	if pos == 81 {
		return 1
	}

	row, col := pos/9, pos%9
	count := 0

	for num := 1; num <= 9; num++ {
		if !BoardSpace(board).Safe(pos, row, col, num) {
			continue
		}

		board.place(row, col, num)
		count += board.countSolutions(pos+1, limit-count)
		board.remove(row, col)

		if count >= limit {
			break
		}
	}

	return count
}

func (board PuzzleSpace) generatePuzzle(difficulty Difficulty) {
	emptySpace := 81 - difficulty.Clue
	checker := map[int]struct{}{}

	firstChecker := 0
	for emptySpace > 0 {
		pos := rand.IntN(81)
		if _, found := checker[pos]; found {
			// emptySpace -= 1
			continue
		}

		checker[pos] = struct{}{}
		row, col := pos/9, pos%9

		// 1.
		ogVal := board[row][col]
		board.remove(row, col)
		if firstChecker == 0 {
			emptySpace -= 1
			firstChecker += 1
			continue
		}

		// 2, 3
		if !board.validForRemoval() {
			// 4a
			board.place(row, col, ogVal)
			continue
		}

		emptySpace -= 1
	}
}
