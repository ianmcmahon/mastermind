package solver

import (
	"fmt"
	"math"
	"testing"
	"time"

	mm "github.com/ianmcmahon/mastermind"
)

func (g *Solver) validCode(c mm.Code) bool {
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
	game := NewSolver(mm.NewGame())

	codes, _ := game.allPossibleCodes()

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
	var worstCaseCode mm.Code

	positions := 5
	colors := byte(6)

	numGames := 3
	for i := 0; i < numGames; i++ {
		solver := NewSolver(mm.NewCustomGame(positions, colors))

		winner, err := solver.Solve()
		if err != nil {
			t.Error(err)
			t.Fail()
		}
		sumDuration += solver.SolveTime

		if !solver.IsWinner(winner) {
			t.Error("Solution incorrect!")
		}
		if solver.TurnsTaken > worstCaseMoves {
			worstCaseMoves = solver.TurnsTaken
			worstCaseCode = winner
		}
	}
	avgTime := sumDuration / time.Duration(numGames)
	fmt.Printf("Solved worst case (%v) in %d moves\n", worstCaseCode, worstCaseMoves)
	fmt.Printf("Average solve time: %v\n", avgTime)
	if positions == 4 && colors == 6 && worstCaseMoves > 5 {
		t.Error(fmt.Errorf("Worst case took %d moves to solve, should be no more than 5", worstCaseMoves))
	}
}

func TestAllPossible(t *testing.T) {
	worstCaseMoves := 0
	sumDuration := 0 * time.Millisecond
	var worstCaseCode mm.Code

	codes, _ := NewSolver(mm.NewGame()).allPossibleCodes()
	numGames := len(codes)
	for _, code := range codes {
		solver := NewSolver(mm.NewCustomGameWithSecret(4, 6, code))

		winner, err := solver.Solve()
		if err != nil {
			t.Error(err)
			t.Fail()
		}
		sumDuration += solver.SolveTime

		if !solver.IsWinner(winner) {
			t.Error(fmt.Errorf("Solution for %s incorrect! Got %s", code, winner))
		}
		if solver.TurnsTaken > worstCaseMoves {
			worstCaseMoves = solver.TurnsTaken
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

func BenchmarkSolution(b *testing.B) {
	for i := 0; i < b.N; i++ {
		solver := NewSolver(mm.NewGame())
		solver.Solve()
	}
}
