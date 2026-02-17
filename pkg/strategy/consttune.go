package strategy

import (
	"fmt"
	"math/rand"
	"sort"

	"github.com/wildfunctions/genetic_series/pkg/expr"
	"github.com/wildfunctions/genetic_series/pkg/pool"
	"github.com/wildfunctions/genetic_series/pkg/series"
)

const (
	constTuneEliteRate      = 0.10 // top 10% carried over (higher than tournament to preserve good combos)
	constTuneTournamentSize = 5
	constTuneWideRate       = 0.10 // fraction of non-elite that get wide exploration
)

func init() {
	Register("consttune", func() Strategy { return &ConstantTuneStrategy{} })
}

// ConstantTuneStrategy freezes the expression tree structure and only varies
// integer constants, using hill-climbing with tournament selection.
type ConstantTuneStrategy struct {
	seed *series.Candidate
}

func (s *ConstantTuneStrategy) Name() string { return "consttune" }

// SetSeedFormula parses a LaTeX formula and stores it as the seed candidate.
func (s *ConstantTuneStrategy) SetSeedFormula(latex string) error {
	c, err := series.ParseCandidateLatex(latex)
	if err != nil {
		return fmt.Errorf("parsing seed formula: %w", err)
	}
	s.seed = c
	return nil
}

func (s *ConstantTuneStrategy) Initialize(_ pool.Pool, rng *rand.Rand, popSize int) []*series.Candidate {
	pop := make([]*series.Candidate, popSize)

	// Clone 0 is the unmodified original (baseline).
	pop[0] = s.seed.Clone()

	// Remaining clones get 1-3 constant perturbations of ±1 to ±5.
	for i := 1; i < popSize; i++ {
		c := s.seed.Clone()
		nPerturbs := rng.Intn(3) + 1 // 1, 2, or 3
		for j := 0; j < nPerturbs; j++ {
			perturbConstWide(c, rng, 5)
		}
		pop[i] = c
	}

	return pop
}

func (s *ConstantTuneStrategy) Evolve(
	population []*series.Candidate,
	fitnesses []series.Fitness,
	_ pool.Pool,
	rng *rand.Rand,
) []*series.Candidate {
	n := len(population)
	next := make([]*series.Candidate, 0, n)

	// Sort indices by fitness (descending).
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}
	sort.Slice(indices, func(a, b int) bool {
		return fitnesses[indices[a]].Combined > fitnesses[indices[b]].Combined
	})

	// Elitism: carry over top 10%.
	eliteCount := int(float64(n) * constTuneEliteRate)
	if eliteCount < 1 {
		eliteCount = 1
	}
	for i := 0; i < eliteCount; i++ {
		next = append(next, population[indices[i]].Clone())
	}

	// Fill rest via tournament selection + const perturbation.
	wideCount := int(float64(n-eliteCount) * constTuneWideRate)
	nonEliteFilled := 0

	for len(next) < n {
		parent := constTuneSelect(population, fitnesses, rng)
		child := parent.Clone()

		if nonEliteFilled < wideCount {
			// Wide exploration: replace a random constant with a value in [-100, 100].
			replaceRandomConst(child, rng, 100)
		} else {
			// Normal hill-climb: 1-2 small perturbations.
			nPerturbs := rng.Intn(2) + 1
			for j := 0; j < nPerturbs; j++ {
				constPerturb(child.Numerator, rng)
				if rng.Float64() < 0.5 {
					constPerturb(child.Denominator, rng)
				}
			}
		}

		// Simplify (constant folding may collapse sub-expressions).
		child.Numerator = expr.SimplifyBigFloat(child.Numerator, 128)
		child.Denominator = expr.SimplifyBigFloat(child.Denominator, 128)

		next = append(next, child)
		nonEliteFilled++
	}

	return next[:n]
}

// constTuneSelect performs tournament selection for constant tuning.
func constTuneSelect(pop []*series.Candidate, fitnesses []series.Fitness, rng *rand.Rand) *series.Candidate {
	bestIdx := rng.Intn(len(pop))
	bestFit := fitnesses[bestIdx].Combined

	for i := 1; i < constTuneTournamentSize; i++ {
		idx := rng.Intn(len(pop))
		if fitnesses[idx].Combined > bestFit {
			bestIdx = idx
			bestFit = fitnesses[idx].Combined
		}
	}

	return pop[bestIdx]
}

// perturbConstWide perturbs a random constant in the candidate by ±1 to ±maxDelta.
func perturbConstWide(c *series.Candidate, rng *rand.Rand, maxDelta int) {
	// Pick numerator or denominator.
	tree := c.Numerator
	if rng.Float64() < 0.5 {
		tree = c.Denominator
	}

	consts := collectConsts(tree)
	if len(consts) == 0 {
		return
	}
	target := consts[rng.Intn(len(consts))]
	delta := int64(rng.Intn(maxDelta) + 1)
	if rng.Float64() < 0.5 {
		delta = -delta
	}
	target.Val += delta
	if target.Val == 0 {
		target.Val = 1
	}
}

// replaceRandomConst replaces a random constant in the candidate with a new value in [-maxVal, maxVal].
func replaceRandomConst(c *series.Candidate, rng *rand.Rand, maxVal int) {
	tree := c.Numerator
	if rng.Float64() < 0.5 {
		tree = c.Denominator
	}

	consts := collectConsts(tree)
	if len(consts) == 0 {
		return
	}
	target := consts[rng.Intn(len(consts))]
	newVal := int64(rng.Intn(2*maxVal+1)) - int64(maxVal)
	if newVal == 0 {
		newVal = 1
	}
	target.Val = newVal
}
