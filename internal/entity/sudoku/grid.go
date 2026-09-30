package sudoku

import "fmt"

const gridCells = 81

// String encodes the board row-major as 81 digits, 0 meaning an empty cell.
func (board BoardSpace) String() string {
	out := make([]byte, 0, gridCells)
	for row := range 9 {
		for col := range 9 {
			out = append(out, byte('0'+board[row][col]))
		}
	}

	return string(out)
}

func (board PuzzleSpace) String() string {
	return BoardSpace(board).String()
}

// Grid returns the board as [row][col], the shape the API exposes.
func (board BoardSpace) Grid() [][]int {
	grid := make([][]int, 9)
	for row := range 9 {
		grid[row] = append([]int(nil), board[row]...)
	}

	return grid
}

func (board PuzzleSpace) Grid() [][]int {
	return BoardSpace(board).Grid()
}

// GridFromString decodes the 81-digit encoding produced by BoardSpace.String.
func GridFromString(encoded string) ([][]int, error) {
	if len(encoded) != gridCells {
		return nil, fmt.Errorf("grid must be %d cells, got %d", gridCells, len(encoded))
	}

	grid := make([][]int, 9)
	for row := range 9 {
		grid[row] = make([]int, 9)
		for col := range 9 {
			char := encoded[row*9+col]
			if char < '0' || char > '9' {
				return nil, fmt.Errorf("invalid cell %q at %d,%d", char, row, col)
			}
			grid[row][col] = int(char - '0')
		}
	}

	return grid, nil
}
