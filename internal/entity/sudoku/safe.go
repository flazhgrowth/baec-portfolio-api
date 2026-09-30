package sudoku

import "slices"

func (board BoardSpace) Safe(pos, row, col int, num int) bool {
	return board.rowSafe(row, num) && board.colSafe(col, num) && board.regionSafe(pos, num)
}

func (board BoardSpace) rowSafe(row int, num int) bool {
	return !slices.Contains(board[row], num)
}

func (board BoardSpace) colSafe(col int, num int) bool {
	for row := range 9 {
		if board[row][col] == num {
			return false
		}
	}

	return true
}

func (board BoardSpace) regionSafe(pos, num int) bool {
	for _, region := range regions {
		if !slices.Contains(region, pos) {
			continue
		}

		for _, regionPos := range region {
			row, col := regionPos/9, regionPos%9
			if board[row][col] == num {
				return false
			}
		}
	}

	return true
}
