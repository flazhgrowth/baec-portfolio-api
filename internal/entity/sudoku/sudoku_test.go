package sudoku

import "testing"

func TestGeneratedPuzzle(t *testing.T) {
	for _, diff := range []Difficulty{DIFF_EASY, DIFF_MEDIUM, DIFF_HARD} {
		board := NewBoard()
		board.Generate()
		puzzle := board.ToPuzzle()
		puzzle.Generate(diff)

		clues := 0
		for row := range 9 {
			for col := range 9 {
				if puzzle.Solution[row][col] == 0 {
					t.Fatalf("%s: solution has an empty cell at %d,%d", diff.Slug, row, col)
				}
				if puzzle.Puzzle[row][col] == 0 {
					continue
				}
				clues++
				if puzzle.Puzzle[row][col] != puzzle.Solution[row][col] {
					t.Fatalf("%s: clue at %d,%d differs from solution", diff.Slug, row, col)
				}
			}
		}
		// Generate may keep a few clues over target if carving gets stuck, but never fewer.
		if clues < diff.Clue {
			t.Fatalf("%s: want at least %d clues, got %d", diff.Slug, diff.Clue, clues)
		}
		if n := puzzle.Puzzle.countSolutions(0, 2); n != 1 {
			t.Fatalf("%s: puzzle has %d solutions, want exactly 1", diff.Slug, n)
		}
	}
}

func TestGridRoundTrip(t *testing.T) {
	board := NewBoard()
	board.Generate()

	grid, err := GridFromString(board.String())
	if err != nil {
		t.Fatal(err)
	}
	for row := range 9 {
		for col := range 9 {
			if grid[row][col] != board[row][col] {
				t.Fatalf("mismatch at %d,%d", row, col)
			}
		}
	}
	if _, err := GridFromString("123"); err == nil {
		t.Fatal("expected error for short grid")
	}
}
