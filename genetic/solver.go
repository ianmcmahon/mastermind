package genetic

import (
	"fmt"
	"math"
	"math/rand"
	"rn/parallel"
	"sort"

	mm "github.com/ianmcmahon/mastermind"
)

const (
	initialPopulationSize int     = 150
	maxGenerations        int     = 100
	maxSamplePopulation   int     = 60
	fitnessThreshold      float64 = 0.0
	spawnRate             float64 = 0.5
)

type Solver struct {
	*mm.Game
	move    int
	guesses []mm.Code
	results []mm.Result
}

func NewSolver(g *mm.Game) *Solver {
	s := &Solver{
		Game: g,
		move: 0,
	}
	maxGuesses := s.maxGuesses()
	s.results = make([]mm.Result, maxGuesses)
	s.guesses = make([]mm.Code, maxGuesses)
	return s
}

func (s *Solver) Solve() (mm.Code, error) {
	var err error

	guess := s.InitialGuess()

	for {
		if s.move >= 9 {
			return nil, fmt.Errorf("didn't find solution in %d moves", s.move)
		}
		s.move++
		s.guesses[s.move] = guess
		fmt.Printf("GUESS: %v\n", guess)
		s.results[s.move], err = s.ScoredGuess(guess)
		if err != nil {
			return nil, err
		}

		if s.IsWin(s.results[s.move]) {
			return guess, nil
		}

		Ei := make(Population, 0)
		population := s.InitializePopulation(initialPopulationSize)

		fmt.Printf("move %d: initial %d\n", s.move, len(population))

		for h := 0; h < maxGenerations; h++ {
			fmt.Printf("move %d generation %d\n", s.move, h)

			// add last move's Ei to this move's population
			for k, v := range Ei {
				population[k] = v
			}

			// Generate new population using crossover, mutation, inversion and permutation;
			population = s.Generate(population)

			for _, c := range population {
				f := s.fitness(c)
				if s.move > 1 {
					//fmt.Printf("move %d: second cull: %v - %.2f\n", s.move, c, f)
				}
				if f <= fitnessThreshold {
					Ei[c.Key()] = c
				}
			}
			if len(Ei) >= maxSamplePopulation {
				break
			}
		}
		fmt.Printf("move %d: population %d\n", s.move, len(population))
		fmt.Printf("move %d: Ei %d: %v\n", s.move, len(Ei), Ei)

		guess = s.BestCandidate(Ei).Code
	}
}

// theoretically this algorithm should be able to complete in O(n log log n)
// n^2 should be plenty big enough; maybe revisit and calculate a tighter
// set once the algorithm is optimal
func (s *Solver) maxGuesses() int {
	return int(math.Ceil(math.Pow(float64(s.Positions()), 2.0)))
}

func (s *Solver) InitialGuess() mm.Code {
	size := s.GameSize()
	switch size.Positions {
	case 4:
		return mm.Code{0, 0, 1, 2}
	case 5:
		return mm.Code{0, 0, 1, 2, 3}
	case 6:
		return mm.Code{0, 0, 1, 1, 2, 3}
	}
	return mm.Code{}
}

// Initialize population;
// A population of size 150 is used, which is initialized randomly,
// taking into account that every code in the population should be distinct.
func (s *Solver) InitializePopulation(size int) Population {
	set := make(Population, size)
	for i := 0; i < size; {
		code := s.RandomCode()
		if _, ok := set[code.String()]; !ok {
			set[code.String()] = Citizen{Code: code}
			i++
		}
	}
	return set
}

//  In order to compute the fitness value of a chromosome c, we compare it with
// every previous guess gq by determining the number of black pins Xq′ (c) and the
// number of white pins Yq′(c) that the code c would score if the previous guess gq
// were the secret code. The difference between Xq′ and Xq and between Yq′ and Yq
// is an indication of the quality of the code c; if these differences are zero for
// each previous guess gq then the code is eligible.
//
// {X'q(c), Y'q(c)} is the result produced for guess gq if c were the secret
// {Xq, Yq} is the actual result produced for the guess at move q
//
// f(c;i) = a(sum[q=1-i](|X'q(c) - Xq|) + sum[q=1-i](|Y'q(c) - Yq|) + bP(i-1)
//
// P is the number of positions in the game
// a and b are weights allowing us to balance the weight of black pins (corrects)
// against a constant proportional to P and the number of turns taken.
// initially, a = 2, b = 2
func (s *Solver) fitness(c Citizen) float64 {
	a := 2.0
	b := 2.0
	P := float64(s.Size.Positions)

	sumX := 0.0
	sumY := 0.0

	for q := 1; q <= s.move; q++ {
		gq := s.guesses[q]
		// resQ = {Xq,Yq}
		// resP = {X'q(c), Y'q(c)
		resQ := s.results[q]
		resP, _ := mm.CheckCode(c.Code, gq, s.Size.Colors)

		sumX += absi(resP.Correct - resQ.Correct)
		sumY += absi(resP.HalfCorrect - resQ.HalfCorrect)
	}

	fitness := (a * sumX) + sumY + (b * P * float64((s.move - 1)))

	return fitness
}

func absi(v int) float64 {
	return math.Abs(float64(v))
}

func (s *Solver) Fitness(pop Population) fitnessList {
	citizens := fitnessList{}

	limiter := parallel.NewLimiter(1)

	for _, citizen := range pop {
		c := citizen

		limiter.Go(func() error {
			f := s.fitness(c)
			c.fitness = f
			limiter.Locked(func() error {
				citizens = append(citizens, c)
				return nil
			})
			return nil
		})
	}

	limiter.Wait()

	// sort elders by fitness
	sort.Sort(citizens)

	return citizens
}

// Generate new population using crossover, mutation, inversion and permutation;
func (s *Solver) Generate(pop Population) Population {
	nextGen := make(Population, len(pop))

	elders := s.Fitness(pop)
	fmt.Printf("move %d: %d: %v\n", s.move, len(elders), elders)

	// take the first half of elders
	elders = elders[0 : len(elders)/2]

	// pair off top two elders and spawn until list is consumed
	for {
		if len(elders) < 2 {
			break
		}
		x, y := elders[0], elders[1]
		elders = elders[2:]

		// eligible parents go in next generation
		nextGen[x.Key()] = x
		nextGen[y.Key()] = y

		// spawn two inverse children
		a := s.Spawn(x, y)
		b := s.Spawn(y, x)

		a.fitness = s.fitness(a)
		b.fitness = s.fitness(b)

		// both go in next generation
		nextGen[a.Key()] = a
		nextGen[b.Key()] = b

		fmt.Printf("eligible parents %v and %v produced children %v and %v\n", x, y, a, b)
	}

	fmt.Printf("initial population %d, next generation %d\n", len(pop), len(nextGen))

	return nextGen
}

func (s *Solver) BestCandidate(p Population) Citizen {
	// naive way: take random one.
	for _, c := range p {
		return c
	}

	// whitepaper way:
	// algorithmically determine the code most like other codes
	fmt.Printf("WARN: Best Candidate didn't find a match, returning random code!\n")
	return Citizen{Code: s.RandomCode()}
}

// Subsequent generations of the population are created through 1-point or 2-point crossover
// from two parents of the previous generation. The crossover is followed by a mutation that
// has a chance to replace the color of one randomly chosen position by a random other color.
// Next, there is a chance of permutation, where the colors of two random positions are switched.
// Finally, there is a chance of inversion, in which case two positions are randomly picked,
// and the sequence of colors between these positions is inverted.
// When these procedures lead to a code that is already present in the population, it is replaced
// by a randomly composed code, in order to improve the diversity of the population.
func (s *Solver) Spawn(x, y Citizen) Citizen {
	child := s.crossover(x, y)
	s.mutate(child)
	s.permute(child)
	s.invert(child)
	return child
}

// 1-point crossover with probability 0.5
// 2-point crossover with probability 0.5
// attempts to divide the chromosome into as equal parts as possible
// currently always uses the same combinations; maybe the inverse should be possible?
func (s *Solver) crossover(x, y Citizen) Citizen {
	roll := rand.Float64()

	child := make(mm.Code, s.Positions())
	copy(child, x.Code)

	cp1, cp2 := 0, 0
	if roll < 0.5 {
		cp2 = int(s.Size.Positions / 2)
	} else {
		cp1 = int(s.Size.Positions / 3)
		cp2 = s.Size.Positions - cp1
	}

	for i := cp1; i < cp2; i++ {
		child[i] = y.Code[i]
	}

	return Citizen{Code: child}
}

//  With a probability of 0.03, a mutation replaces the color
// of one randomly chosen position by a random other color.
func (s *Solver) mutate(c Citizen) bool {
	roll := rand.Float64()

	if roll < 0.03 {
		pos := rand.Intn(s.Positions())
		for {
			col := byte(rand.Intn(int(s.Colors())))
			if c.Code[pos] != col {
				c.Code[pos] = col
				return true
			}
		}
	}

	return false
}

// 0.03 chance of permutation, where the colors of two random positions are switched.
func (s *Solver) permute(c Citizen) bool {
	roll := rand.Float64()

	if roll < 0.03 {
		p1, p2 := rand.Intn(s.Positions()), 0
		i := 0
		for {
			i++
			p2 = rand.Intn(s.Positions())
			if p1 == p2 {
				continue
			}
			if c.Code[p1] != c.Code[p2] {
				break
			}
			if i > 10 {
				break
			}
		}
		c.Code[p1], c.Code[p2] = c.Code[p2], c.Code[p1]
		return true
	}

	return false
}

// 0.02 chance of inversion, in which case two positions are randomly picked,
// and the sequence of colors between these positions is inverted.
func (s *Solver) invert(c Citizen) bool {
	roll := rand.Float64()

	if roll < 0.02 {
		p1, p2 := rand.Intn(s.Positions()), 0
		for {
			p2 = rand.Intn(s.Positions())
			if p1 != p2 {
				break
			}
		}
		p1, p2 = order(p1, p2)
		s := make([]byte, 1+p2-p1)
		copy(s, c.Code[p1:p2+1])
		for i, v := range s {
			c.Code[p2-i] = v
		}
		return true
	}
	return false
}

func order(x, y int) (int, int) {
	if y < x {
		return y, x
	}
	return x, y
}

type Population map[string]Citizen

type Citizen struct {
	mm.Code
	fitness float64
}

func (c Citizen) Key() string {
	return c.Code.String()
}

func (c Citizen) String() string {
	return fmt.Sprintf("{%s, %.2f}", c.Code.String(), c.fitness)
}

type fitnessList []Citizen

func (s fitnessList) Less(i, j int) bool {
	return s[i].fitness < s[j].fitness
}

func (s fitnessList) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s fitnessList) Len() int {
	return len(s)
}
