package sudoku

type (
	Difficulty struct {
		Clue int
		Slug string
	}
)

var (
	DIFF_EASY Difficulty = Difficulty{
		Clue: 40,
		Slug: "easy",
	}
	DIFF_MEDIUM Difficulty = Difficulty{
		Clue: 32,
		Slug: "medium",
	}
	DIFF_HARD Difficulty = Difficulty{
		Clue: 26,
		Slug: "hard",
	}
)
