package mastermind

import (
	"fmt"
	"math"
	"rn/parallel"
	"sort"
)

type codeSlice []Code

func (s codeSlice) Less(i, j int) bool {
	return s[i].String() < s[j].String()
}

func (s codeSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s codeSlice) Len() int {
	return len(s)
}

type codeSet map[string]Code

func (g *Game) allPossibleCodes() codeSet {
	numPossibleCodes := int(math.Pow(float64(g.Colors()), float64(g.Positions())))
	out := make(codeSet, numPossibleCodes)

	for i := 0; i < numPossibleCodes; i++ {
		remainder := i
		code := g.EmptyCode()
		for pos := 0; pos < g.Positions(); pos++ {
			power := int(math.Pow(float64(g.Colors()), float64(g.Positions()-pos-1)))
			posVal := int(remainder / power)
			remainder -= posVal * power
			code[pos] = byte(posVal)
		}
		out[code.String()] = code
	}

	return out
}

func (game *Game) initialGuess() Code {
	code := game.EmptyCode()
	for i, _ := range code {
		if i < game.Positions()/2 {
			code[i] = 0
		} else {
			code[i] = 1
		}
	}
	return code
}

// a situation is a case where we know how many possible codes are remaining in S,
// and we know the result of each next move
// initial situation is 1296 codes in S, next guess is 1122

type situation struct {
	lastGuess  Code
	totalCodes int
	remaining  int
}

func (g *Game) initialSituation() situation {
	numPossibleCodes := int(math.Pow(float64(g.Colors()), float64(g.Positions())))
	return situation{
		totalCodes: numPossibleCodes,
	}
}

func (s situation) nextGuess(r Result, remaining int) Code {
	if remaining == s.totalCodes {
		return Code([]byte{0, 0, 1, 1})
	}

	for black := 0; black < r.correct; black++ {
		for white := 0; white < r.halfCorrect; white++ {

		}
	}

	return Code{}
}

func (g *Game) possibleResults() []Result {
	out := []Result{}
	for black := 0; black <= g.Positions(); black++ {
		for white := g.Positions() - black; white >= 0; white-- {
			out = append(out, Result{black, white})
		}
	}
	return out
}

func (g *Game) emptyHitMap() map[Result]int {
	results := g.possibleResults()
	hitmap := make(map[Result]int, len(results))
	for _, r := range results {
		hitmap[r] = 0
	}
	return hitmap
}

func Solve(game *Game) (Code, error) {
	// create set S of possible codes
	S := game.allPossibleCodes()

	// and the same set P of possible codes, which we won't remove from
	P := game.allPossibleCodes()

	usedCodes := map[string]Code{}

	guess := game.initialGuess()

	// while result of guess is not a win:
	pass := 0
	for {

		pass++
		usedCodes[guess.String()] = guess
		delete(S, guess.String())
		//fmt.Printf("*** guessing %s\n", guess)
		result, err := game.ScoredGuess(guess)
		if err != nil {
			return nil, fmt.Errorf("bad guess %s: %v", guess, err)
		}
		if game.IsWin(result) {
			return guess, nil
		}

		//  remove from S any code that has a different result than our guess
		hitcounts := game.emptyHitMap()
		for k, s := range S {
			res2, err := CheckCode(s, guess, game.Colors())
			if err != nil {
				return nil, err
			}

			hitcounts[res2]++

			if res2 != result {
				delete(S, k)
			}
		}

		if len(S) <= 2 {
			for _, s := range S {
				guess = s
			}
			continue
		}

		limiter := parallel.NewLimiter(100)

		guesses := map[int][]Code{}

		for pk, p := range P {
			// skip codes we've already tried
			if _, ok := usedCodes[pk]; ok {
				continue
			}

			p1 := p
			limiter.Go(func() error {
				// count the number of distinct results each guess would produce
				// for the remaining set S
				hitCounts := game.emptyHitMap()
				for _, s := range S {
					result, err := CheckCode(p1, s, game.Colors())
					if err != nil {
						return err
					}

					hitCounts[result]++
				}
				// score these as the number of possibilities removed from S for that guess
				// on the next pass, after guessing this code

				bestScore := 0
				var bestResult Result
				for result, count := range hitCounts {
					//fmt.Printf("code %s: any code resulting in %v will remove %d from S(%d)\n", p1, result, count, len(S))
					score := count
					if score > bestScore {
						bestScore = score
						bestResult = result
					}
				}

				_ = bestResult
				//fmt.Printf("best result for %s is %v, with a score of %d and %d in S(%d) producing same this result\n", p1, bestResult, bestScore, hitCounts[bestResult], len(S))

				limiter.Locked(func() error {
					if _, ok := guesses[bestScore]; !ok {
						guesses[bestScore] = []Code{}
					}
					guesses[bestScore] = append(guesses[bestScore], p1)
					return nil
				})
				return nil
			})
		}

		if err := limiter.Wait(); err != nil {
			return nil, err
		}

		bestScore := 0
		totalGuesses := 0
		var bestGuesses []Code
		for cnt, codes := range guesses {
			score := len(S) - cnt
			//fmt.Printf("score %d, guesses %v\n", score, codes)
			totalGuesses += len(codes)
			if score > bestScore {
				bestScore = score
				bestGuesses = codes
			}
		}

		//fmt.Printf("best score: %d, guesses: %d\n", bestScore, len(bestGuesses))

		// for next guess, default to the first 'best guess'
		// if any of the best guesses are in S, use the first of those instead
		inS := codeSlice{}
		notInS := codeSlice{}
		for _, g := range bestGuesses {
			if _, ok := S[g.String()]; ok {
				inS = append(inS, g)
			} else {
				notInS = append(notInS, g)
			}
		}
		sort.Sort(inS)
		sort.Sort(notInS)
		//fmt.Printf("%d guesses in S, ", len(inS))
		//fmt.Printf("%d guesses not in S\n", len(notInS))

		P1 := inS
		if len(inS) == 0 {
			P1 = notInS
		}
		// let's see if we can find a code that minimizes the set of possible next moves
		minMax := 1296
		codesForMax := map[int]codeSlice{}
		for _, p := range P1 {
			hitcount := game.emptyHitMap()
			for _, s := range S {
				res, _ := CheckCode(p, s, game.Colors())
				hitcount[res]++
			}
			sum := 0
			max := 0
			for _, r := range game.possibleResults() {
				sum += hitcount[r]
				if hitcount[r] > max {
					max = hitcount[r]
				}
			}
			if _, ok := codesForMax[max]; !ok {
				codesForMax[max] = codeSlice{}
			}
			codesForMax[max] = append(codesForMax[max], p)

			if max < minMax {
				minMax = max
			}
		}

		//fmt.Printf("%d codes have minMax %d nodes on their biggest tree:\n", len(codesForMax[minMax]), minMax)
		sort.Sort(codesForMax[minMax])
		//fmt.Printf("%v\n", codesForMax[minMax])

		guess = codesForMax[minMax][0]
	}

	return nil, nil
}
