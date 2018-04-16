package mastermind

import (
	"fmt"
	"math"
	"testing"
	"time"
)

func (g *Game) validCode(c Code) bool {
	if len(c) != g.Positions() {
		return false
	}
	for _, v := range c {
		if v < 0 || v > g.Colors()-1 {
			return false
		}
	}
	return true
}

func TestAllPossibleCodes(t *testing.T) {
	game := NewGame()

	codes := game.allPossibleCodes()

	for k, v := range codes {
		// assure valid
		if k != v.String() {
			t.Error("map entry %s contains code %s", k, v)
		}
		if !game.validCode(v) {
			t.Error("Invalid code: %s", v)
		}
		// code is a map, assures no duplicates
	}
	// assure correct number
	expected := int(math.Pow(float64(game.Colors()), float64(game.Positions())))
	if len(codes) != expected {
		t.Error("Should be %d (%d^%d) possible codes, only %d codes returned",
			expected, game.Colors(), game.Positions(), len(codes))
	}
}

func TestSolver(t *testing.T) {
	worstCaseMoves := 0
	sumDuration := 0 * time.Millisecond
	var worstCaseCode Code

	codes := NewGame().allPossibleCodes()
	numGames := len(codes)
	for _, code := range codes {
		game := NewGame()
		game.setSecretCode(code)

		winner, err := Solve(game)
		if err != nil {
			t.Error(err)
			t.Fail()
		}
		sumDuration += game.SolveTime

		if !game.isWinner(winner) {
			t.Error(fmt.Errorf("Solution incorrect! got %s, expected %s", winner, game.secretCode))
		}
		if game.TurnsTaken > worstCaseMoves {
			worstCaseMoves = game.TurnsTaken
			worstCaseCode = winner
		}
	}
	avgTime := sumDuration / time.Duration(numGames)
	fmt.Printf("Solved worst case (%v) in %d moves\n", worstCaseCode, worstCaseMoves)
	fmt.Printf("Average solve time: %v\n", avgTime)
	if worstCaseMoves > 5 {
		t.Error(fmt.Errorf("Worst case took %d moves to solve, should be no more than 5", worstCaseMoves))
	}
}

func TestWorstCase(t *testing.T) {
	game := NewGame()
	code, err := game.Code("2521")
	if err != nil {
		t.Fail()
	}
	game.setSecretCode(code)
	Solve(game)
	if game.TurnsTaken > 5 {
		t.Error("worst case took more than 5 moves to solve")
	}
}

func BenchmarkSolver(b *testing.B) {
	for i := 0; i < b.N; i++ {
		game := NewGame()
		Solve(game)
	}
}
