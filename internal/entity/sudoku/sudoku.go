package sudoku

import (
	"fmt"
	"math/rand/v2"
	"strings"
)

type (
	Nums []int
)

func (nums Nums) randomize() Nums {
	checker, res := map[int]struct{}{}, Nums{}

	for len(checker) < 9 {
		ind := rand.IntN(9)
		val := nums[ind]
		if _, found := checker[val]; !found {
			checker[val] = struct{}{}
			res = append(res, val)
		}
	}

	return res
}

type (
	BoardSpace  map[int][]int
	PuzzleSpace BoardSpace
	Puzzle      struct {
		Solution BoardSpace
		Puzzle   PuzzleSpace
	}
)

func NewBoard() BoardSpace {
	return BoardSpace{
		0: []int{0, 0, 0, 0, 0, 0, 0, 0, 0},
		1: []int{0, 0, 0, 0, 0, 0, 0, 0, 0},
		2: []int{0, 0, 0, 0, 0, 0, 0, 0, 0},

		3: []int{0, 0, 0, 0, 0, 0, 0, 0, 0},
		4: []int{0, 0, 0, 0, 0, 0, 0, 0, 0},
		5: []int{0, 0, 0, 0, 0, 0, 0, 0, 0},

		6: []int{0, 0, 0, 0, 0, 0, 0, 0, 0},
		7: []int{0, 0, 0, 0, 0, 0, 0, 0, 0},
		8: []int{0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
}

var (
	regions map[int][]int = map[int][]int{
		1: {0, 1, 2, 9, 10, 11, 18, 19, 20},
		2: {3, 4, 5, 12, 13, 14, 21, 22, 23},
		3: {6, 7, 8, 15, 16, 17, 24, 25, 26},
		4: {27, 28, 29, 36, 37, 38, 45, 46, 47},
		5: {30, 31, 32, 39, 40, 41, 48, 49, 50},
		6: {33, 34, 35, 42, 43, 44, 51, 52, 53},
		7: {54, 55, 56, 63, 64, 65, 72, 73, 74},
		8: {57, 58, 59, 66, 67, 68, 75, 76, 77},
		9: {60, 61, 62, 69, 70, 71, 78, 79, 80},
	}
)

func (board BoardSpace) Generate() {
	nums := Nums{1, 2, 3, 4, 5, 6, 7, 8, 9}
	for pos := range 81 {
		board.fill(pos, nums.randomize())
	}
}

func (board BoardSpace) PrintSolution() {
	for row := range 9 {
		currRow := []string{}
		for col := range 9 {
			currRow = append(currRow, fmt.Sprintf("%d", board[row][col]))
		}
		fmt.Printf("|%s|\n", strings.Join(currRow, "|"))
	}
}
