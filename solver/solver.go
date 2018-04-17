package solver

import (
	"fmt"
	"math"
	"rn/parallel"
	"sort"
	"sync"

	mm "github.com/ianmcmahon/mastermind"
)

var initialMoves map[mm.GameSize]mm.Code
var initialMutex *sync.Mutex

const (
	maxPositions int  = 10
	maxColors    byte = 10
)

func init() {
	initialMutex = &sync.Mutex{}
	initialMoves = map[mm.GameSize]mm.Code{
		mm.GameSize{4, 6}: mm.Code{0, 0, 1, 1},
		mm.GameSize{5, 6}: mm.Code{0, 0, 1, 2, 3},
	}
}

type Solver struct {
	*mm.Game
	initialMove mm.Code
}

func NewSolver(g *mm.Game) *Solver {
	size := mm.GameSize{g.Positions(), g.Colors()}
	initialMutex.Lock()
	if _, ok := initialMoves[size]; !ok {
		fmt.Printf("calculating initial move for size %v\n", size)
		game := &Solver{mm.NewCustomGame(g.Positions(), g.Colors()), mm.Code{}}
		S, P := game.allPossibleCodes()

		guess := game.bestGuessOfSet(S, P)

		fmt.Printf("game of size %v, initial move: %s\n", size, guess)
		initialMoves[size] = guess
	}
	initialMutex.Unlock()
	g.Reset()
	return &Solver{
		g,
		initialMoves[size],
	}
}

func (g *Solver) MustScoredGuess(code mm.Code) mm.Result {
	r, err := g.ScoredGuess(code)
	if err != nil {
		panic(err)
	}
	return r
}

func (g *Solver) allPossibleCodes() (mm.CodeSet, mm.CodeSlice) {
	numPossibleCodes := int(math.Pow(float64(g.Colors()), float64(g.Positions())))
	set := make(mm.CodeSet, numPossibleCodes)
	slice := make(mm.CodeSlice, numPossibleCodes)

	for i := 0; i < numPossibleCodes; i++ {
		remainder := i
		code := g.EmptyCode()
		for pos := 0; pos < g.Positions(); pos++ {
			power := int(math.Pow(float64(g.Colors()), float64(g.Positions()-pos-1)))
			posVal := int(remainder / power)
			remainder -= posVal * power
			code[pos] = byte(posVal)
		}
		set[code.String()] = code
		slice[i] = code
	}

	return set, slice
}

func (g *Solver) possibleResults() []mm.Result {
	out := []mm.Result{}
	for black := 0; black <= g.Positions(); black++ {
		for white := g.Positions() - black; white >= 0; white-- {
			out = append(out, mm.Result{black, white})
		}
	}
	return out
}

type hitmap map[mm.Result]int

func (h hitmap) maxHits() (mm.Result, int) {
	bestScore := 0
	var bestResult mm.Result
	for result, count := range h {
		score := count
		if score > bestScore {
			bestScore = score
			bestResult = result
		}
	}
	return bestResult, bestScore
}

func (g *Solver) emptyHitMap() hitmap {
	results := g.possibleResults()
	hm := make(hitmap, len(results))
	for _, r := range results {
		hm[r] = 0
	}
	return hm
}

func (g *Solver) selectMovesWithResult(S mm.CodeSet, guess mm.Code, result mm.Result) mm.CodeSet {
	T := mm.CodeSet{}
	hitcounts := g.emptyHitMap()
	for k, s := range S {
		res2, err := mm.CheckCode(s, guess, g.Colors())
		if err != nil {
			panic(err)
		}

		hitcounts[res2]++

		if res2 == result {
			T[k] = s
		}
	}
	return T
}

func (g *Solver) countHits(S mm.CodeSet, code mm.Code) hitmap {
	hitCounts := g.emptyHitMap()
	for _, s := range S {
		result, err := mm.CheckCode(code, s, g.Colors())
		if err != nil {
			panic(err)
		}

		hitCounts[result]++
	}
	return hitCounts
}

// returns intersection of S and codes, unless that set has length 0
// in which case, returns S
func selectGuesses(S mm.CodeSet, codes mm.CodeSlice) mm.CodeSlice {
	inS := mm.CodeSlice{}
	notInS := mm.CodeSlice{}
	for _, g := range codes {
		if _, ok := S[g.String()]; ok {
			inS = append(inS, g)
		} else {
			notInS = append(notInS, g)
		}
	}
	if len(inS) == 0 {
		return notInS
	}
	return inS
}

// checks every p in P (a complete set of possible codes)
// against each s in S, scoring p by the maximum codes represented by one unique Result.
// Returns a map, keyed on score, where score is the total number of codes remaining in S if p is the next guess
// and the value is the set of codes in P which produce that score across all combinations
func (g *Solver) score(S mm.CodeSet, P mm.CodeSlice) map[int]mm.CodeSlice {
	limiter := parallel.NewLimiter(100)
	guesses := map[int]mm.CodeSlice{}

	for _, p := range P {
		p1 := p
		limiter.Go(func() error {
			// count the number of distinct results each possible guess would produce for the remaining set S
			hitsForResult := g.countHits(S, p1)

			// score these as the number of possibilities remaining in S after guessing p1
			_, score := hitsForResult.maxHits()

			limiter.Locked(func() error {
				if _, ok := guesses[score]; !ok {
					guesses[score] = mm.CodeSlice{}
				}
				guesses[score] = append(guesses[score], p1)
				return nil
			})
			return nil
		})
	}

	limiter.Wait()

	return guesses
}

// S is our set of remaining possible solutions
// P is the set of codes that contain the optimal next moves
// Not all codes in P produce optimal solutions;
// the set of optimal next guesses is comprised of codes
// where the largest set possible in the next move is minimal.
// We count the possibilities for each possible result, track the
// maximum of these counts (the largest possible set after move p)
// and then select all codes where this maximum is as small as possible
// (to ensure the smallest set on the next pass)
// we then sort this optimal set and return the smallest code.
func (g *Solver) bestGuessOfSet(S mm.CodeSet, P mm.CodeSlice) mm.Code {
	// let's see if we can find a code that minimizes the set of possible next moves
	minMax := -1
	codesForMax := map[int]mm.CodeSlice{}
	for _, p := range P {
		hitcount := g.emptyHitMap()
		for _, s := range S {
			res, _ := mm.CheckCode(p, s, g.Colors())
			hitcount[res]++
		}
		sum := 0
		max := 0
		for _, r := range g.possibleResults() {
			sum += hitcount[r]
			if hitcount[r] > max {
				max = hitcount[r]
			}
		}
		if _, ok := codesForMax[max]; !ok {
			codesForMax[max] = mm.CodeSlice{}
		}
		codesForMax[max] = append(codesForMax[max], p)

		if minMax < 0 || max < minMax {
			minMax = max
		}
	}

	sort.Sort(codesForMax[minMax])

	return codesForMax[minMax][0]
}

func bestScore(scores map[int]mm.CodeSlice) mm.CodeSlice {
	best := -1
	// we want the minimum score, ie the smallest possible S after this move
	for score, _ := range scores {
		if best < 0 || score < best {
			best = score
		}
	}
	return scores[best]
}

func (game *Solver) Solve() (mm.Code, error) {
	// create set S of possible codes
	S, P := game.allPossibleCodes()

	guess := game.initialMove

	for {
		result := game.MustScoredGuess(guess)

		if game.IsWin(result) {
			return guess, nil
		}

		//  remove from S any code that has a different result than our guess
		S = game.selectMovesWithResult(S, guess, result)

		// if we're down to two possibilities, shortcut to either of them
		if len(S) <= 2 {
			for _, s := range S {
				guess = s
			}
			continue
		}

		// rank every code in complete set P by how many codes it would remove from S next pass
		scores := game.score(S, P)

		// choose the set of codes with the optimal (minimum) score.  Minimum score means
		// the fewest codes remaining in S after choosing any of these codes
		bestGuesses := bestScore(scores)

		// bestGuesses now contains all guesses which minimize S on the next move.
		// bestGuesses can be split into two sets, those contained in S, and those not.
		// if the set of guesses contained in S is empty, choose a best guess from the remainder.
		potentialGuesses := selectGuesses(S, bestGuesses)

		// even though every code in potentialGuesses will produce the same size S' next pass,
		// the distribution of codes in S' wrt Results on the next pass varies depending on which
		// of these codes we choose as our next guess.
		// Optimal solution involves choosing a code such that the maximum set of codes producing the same Result
		// is minimized.
		guess = game.bestGuessOfSet(S, potentialGuesses)
	}

	return nil, nil
}
