package mastermind

import "testing"

func TestGuessLogic(t *testing.T) {
	game := NewGame()
	game.setSecretCode([]byte{5, 4, 3, 2})

	guesses := map[string]Result{
		"1111": Result{0, 0},
		"1234": Result{1, 2},
		"1235": Result{1, 2},
		"4321": Result{0, 3},
		"5321": Result{1, 2},
		"5431": Result{3, 0},
		"5432": Result{4, 0},
	}

	for guess, expected := range guesses {
		result, err := game.GuessString(guess)
		if err != nil {
			t.Error("guess %s generated error: %v", guess, err)
		}
		if result != expected {
			t.Error("for guess %s, got %s, expected %s", guess, result, expected)
		}
	}
}
