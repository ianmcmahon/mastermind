package mastermind

import (
	"bytes"
	"fmt"
	"math"
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
	buf := new(bytes.Buffer)
	for _, r := range c {
		buf.WriteRune(rune(r) + '0')
	}
	return buf.String()
}

type CodeSet map[string]Code

type CodeSlice []Code

func (s CodeSlice) Less(i, j int) bool {
	return s[i].String() < s[j].String()
}

func (s CodeSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s CodeSlice) Len() int {
	return len(s)
}

type Result struct {
	Correct     int
	HalfCorrect int
}

func (r Result) String() string {
	return fmt.Sprintf("%d-%d", r.Correct, r.HalfCorrect)
}

type GameSize struct {
	Positions int
	Colors    byte
}

type Game struct {
	TurnsTaken int
	Size       GameSize
	secretCode Code
	startTime  time.Time
	SolveTime  time.Duration
}

func NewGame() *Game {
	return NewCustomGame(defaultPositions, defaultColors)
}

func randomCode(p int, c byte) Code {
	code := make(Code, p)
	for i := 0; i < p; i++ {
		code[i] = byte(rand.Intn(int(c)))
	}
	return code
}

func (g *Game) RandomCode() Code {
	return randomCode(g.Size.Positions, g.Size.Colors)
}

func NewCustomGame(positions int, colors byte) *Game {
	return NewCustomGameWithSecret(positions, colors, randomCode(positions, colors))
}

func NewCustomGameWithSecret(positions int, colors byte, secret Code) *Game {
	posSqr := math.Pow(float64(positions), 2.0)
	if float64(colors) > posSqr {
		fmt.Printf("Limiting colors to positions^2 (%d)\n", colors)
		colors = byte(posSqr)
	}
	g := &Game{
		TurnsTaken: 0,
		Size: GameSize{
			Positions: positions,
			Colors:    colors,
		},
		secretCode: secret,
		startTime:  time.Now(),
	}
	return g
}

func (g *Game) GameSize() GameSize {
	return g.Size
}

func (g *Game) Reset() {
	g.TurnsTaken = 0
	g.startTime = time.Now()
}

func (g *Game) Positions() int {
	return g.Size.Positions
}

func (g *Game) Colors() byte {
	return g.Size.Colors
}

func (g *Game) EmptyCode() Code {
	return make(Code, g.Positions())
}

func (g *Game) Code(code string) (Code, error) {
	if len(code) != g.Size.Positions {
		return nil, fmt.Errorf("code must have %d positions", g.Size.Positions)
	}
	out := Code(make([]byte, g.Size.Positions))
	for i, c := range code {
		v := byte(c - '0')
		if v < 0 || v >= g.Size.Colors {
			return nil, fmt.Errorf("code must use only colors 0 - %d", g.Size.Colors-1)
		}
		out[i] = v
	}
	return out, nil
}

func (g *Game) setSecretCode(c Code) {
	g.secretCode = c
}

func (g *Game) IsWin(r Result) bool {
	return r.Correct == g.Positions() && r.HalfCorrect == 0
}

func (g *Game) IsWinner(c Code) bool {
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

	if game.IsWin(result) && game.IsWinner(code) {
		game.SolveTime = time.Now().Sub(game.startTime)
		fmt.Printf("%s is a winner; solved in %d moves (%v)\n", code, game.TurnsTaken, game.SolveTime)
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
