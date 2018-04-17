package genetic

import (
	"fmt"
	"testing"
	"time"

	mm "github.com/ianmcmahon/mastermind"
)

// benchmark against large games, 6x9
func BenchmarkInitializePopulation(b *testing.B) {
	solver := NewSolver(mm.NewCustomGame(6, 9))
	solver.InitializePopulation(b.N)
}

func TestGeneticAlgorithm(t *testing.T) {
	worstCaseMoves := 0
	sumDuration := 0 * time.Millisecond
	var worstCaseCode mm.Code

	positions := 4
	colors := byte(6)

	numGames := 1
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
