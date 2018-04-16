package mastermind

import (
	"fmt"
	"math/rand"
	"time"
)

const (
	defaultPositions = 4
	defaultColors    = 6
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type Code []byte

func (c Code) String() string {
	return fmt.Sprintf("%v", []byte(c))
}

type Result struct {
	correct     int
	halfCorrect int
}

func (r Result) String() string {
	return fmt.Sprintf("%d-%d", r.correct, r.halfCorrect)
}

type Game struct {
	TurnsTaken int
	positions  int
	colors     byte
	secretCode Code
	startTime  time.Time
	SolveTime  time.Duration
}

func NewGame() *Game {
	return NewCustomGame(defaultPositions, defaultColors)
}

func NewCustomGame(positions int, colors byte) *Game {
	g := &Game{
		TurnsTaken: 0,
		positions:  positions,
		colors:     colors,
		secretCode: make([]byte, positions),
		startTime:  time.Now(),
	}

	for i := 0; i < positions; i++ {
		g.secretCode[i] = byte(rand.Intn(int(colors)))
	}

	return g
}

func (g *Game) Positions() int {
	return g.positions
}

func (g *Game) Colors() byte {
	return g.colors
}

func (g *Game) EmptyCode() Code {
	return make(Code, g.Positions())
}

func (g *Game) Code(code string) (Code, error) {
	if len(code) != g.positions {
		return nil, fmt.Errorf("code must have %d positions", g.positions)
	}
	out := Code(make([]byte, g.positions))
	for i, c := range code {
		v := byte(c - '0')
		if v < 0 || v >= g.colors {
			return nil, fmt.Errorf("code must use only colors 0 - %d", g.colors-1)
		}
		out[i] = v
	}
	return out, nil
}

func (g *Game) setSecretCode(c Code) {
	g.secretCode = c
}

func (g *Game) IsWin(r Result) bool {
	return r.correct == g.Positions() && r.halfCorrect == 0
}

func (g *Game) isWinner(c Code) bool {
	return c.String() == g.secretCode.String()
}

func (g *Game) isCorrect(code Code, position int) bool {
	return code[position] == g.secretCode[position]
}

func countColors(code Code, color byte) int {
	count := 0
	for _, v := range code {
		if v == color {
			count++
		}
	}
	return count
}

func min(x, y int) int {
	if y < x {
		return y
	}
	return x
}

func (game *Game) GuessString(guess string) (Result, error) {
	code, err := game.Code(guess)
	if err != nil {
		return Result{}, err
	}
	return game.ScoredGuess(code)
}

func (game *Game) ScoredGuess(code Code) (Result, error) {
	game.TurnsTaken++
	result, err := CheckCode(code, game.secretCode, game.Colors())
	if err != nil {
		return result, err
	}

	if game.IsWin(result) && game.isWinner(code) {
		game.SolveTime = time.Now().Sub(game.startTime)
		fmt.Printf("%s is correct; solved in %d moves (%v)\n", code, game.TurnsTaken, game.SolveTime)
		return result, nil
	}

	/*
		fmt.Printf("Move %d: %s is incorrect.  %d correct in position, %d correct but out of position\n",
			game.TurnsTaken, code, result.correct, result.halfCorrect)
	*/

	return result, err
}

func CheckCode(guess, actual Code, colors byte) (Result, error) {
	if len(guess) != len(actual) {
		return Result{}, fmt.Errorf("codes are not equal length")
	}

	// for each possible color, how many exist in the guess? how many in the secret?
	// the minimum of these two numbers is the sum of the correct and half-correct
	// counts for that color.

	// correct counts are easy; for each position, is it correct? sum these.

	// half-correct counts are the total quasi-correct counts minus the full correct count

	correct := 0
	halfCorrect := 0

	for i, _ := range guess {
		if guess[i] == actual[i] {
			correct++
		}
	}

	for i := byte(0); i < colors; i++ {
		x := countColors(guess, i)
		y := countColors(actual, i)
		halfCorrect += min(x, y)
	}

	halfCorrect -= correct

	return Result{correct, halfCorrect}, nil
}
