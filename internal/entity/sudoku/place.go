package sudoku

func (board BoardSpace) place(row, col, num int) {
	board[row][col] = num
}

func (board BoardSpace) remove(row, col int) {
	board[row][col] = 0
}

func (board BoardSpace) fill(pos int, nums Nums) bool {
	if pos == 81 {
		return true
	}

	row, col := pos/9, pos%9

	for _, num := range nums {
		if !board.Safe(pos, row, col, num) {
			continue
		}

		board.place(row, col, num)

		if board.fill(pos+1, nums.randomize()) {
			return true
		}

		board.remove(row, col)
	}

	return false
}

func (board PuzzleSpace) place(row, col, num int) {
	board[row][col] = num
}

func (board PuzzleSpace) remove(row, col int) {
	board[row][col] = 0
}

func (board PuzzleSpace) cellIsEmpty(row, col int) bool {
	return board[row][col] == 0
}
