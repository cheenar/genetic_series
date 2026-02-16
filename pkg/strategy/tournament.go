package strategy

import (
	"math/rand"
	"sort"

	"github.com/wildfunctions/genetic_series/pkg/expr"
	"github.com/wildfunctions/genetic_series/pkg/pool"
	"github.com/wildfunctions/genetic_series/pkg/series"
)

const (
	tournamentMaxDepth      = 4
	tournamentSize          = 5
	eliteRate               = 0.05 // top 5% carried over
	mutationRate            = 0.8  // probability of mutation after crossover
	tournamentInjectionRate = 0.05 // fraction of non-elite slots replaced with random each gen
)

func init() {
	Register("tournament", func() Strategy { return &TournamentStrategy{} })
}

// TournamentStrategy implements tournament selection with crossover and mutation.
type TournamentStrategy struct{}

func (s *TournamentStrategy) Name() string { return "tournament" }

func (s *TournamentStrategy) Initialize(p pool.Pool, rng *rand.Rand, popSize int) []*series.Candidate {
	pop := make([]*series.Candidate, popSize)
	for i := range pop {
		pop[i] = randomCandidate(p, rng, tournamentMaxDepth)
	}
	return pop
}

func (s *TournamentStrategy) Evolve(
	population []*series.Candidate,
	fitnesses []series.Fitness,
	p pool.Pool,
	rng *rand.Rand,
) []*series.Candidate {
	n := len(population)
	next := make([]*series.Candidate, 0, n)

	// Sort indices by fitness (descending)
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}
	sort.Slice(indices, func(a, b int) bool {
		return fitnesses[indices[a]].Combined > fitnesses[indices[b]].Combined
	})

	// Elitism: carry over top candidates
	eliteCount := int(float64(n) * eliteRate)
	if eliteCount < 1 {
		eliteCount = 1
	}
	for i := 0; i < eliteCount; i++ {
		next = append(next, population[indices[i]].Clone())
	}

	// Fill rest via tournament selection + crossover + mutation
	for len(next) < n {
		p1 := tournamentSelect(population, fitnesses, rng)
		p2 := tournamentSelect(population, fitnesses, rng)

		c1, c2 := CrossoverCandidates(p1, p2, rng)

		// Mutation + simplification
		if rng.Float64() < mutationRate {
			MutateCandidate(c1, p, rng)
		}
		c1.Numerator = expr.SimplifyBigFloat(c1.Numerator, 128)
		c1.Denominator = expr.SimplifyBigFloat(c1.Denominator, 128)

		if rng.Float64() < mutationRate {
			MutateCandidate(c2, p, rng)
		}
		c2.Numerator = expr.SimplifyBigFloat(c2.Numerator, 128)
		c2.Denominator = expr.SimplifyBigFloat(c2.Denominator, 128)

		// Reject overly deep trees
		if candidateOK(c1) {
			next = append(next, c1)
		} else {
			next = append(next, randomCandidate(p, rng, tournamentMaxDepth))
		}
		if len(next) < n {
			if candidateOK(c2) {
				next = append(next, c2)
			} else {
				next = append(next, randomCandidate(p, rng, tournamentMaxDepth))
			}
		}
	}

	// Random injection: replace some non-elite slots with fresh candidates
	injectionCount := int(float64(n) * tournamentInjectionRate)
	if injectionCount < 1 {
		injectionCount = 1
	}
	for i := 0; i < injectionCount; i++ {
		idx := eliteCount + rng.Intn(n-eliteCount)
		next[idx] = randomCandidate(p, rng, tournamentMaxDepth)
	}

	return next[:n]
}

func tournamentSelect(pop []*series.Candidate, fitnesses []series.Fitness, rng *rand.Rand) *series.Candidate {
	bestIdx := rng.Intn(len(pop))
	bestFit := fitnesses[bestIdx].Combined

	for i := 1; i < tournamentSize; i++ {
		idx := rng.Intn(len(pop))
		if fitnesses[idx].Combined > bestFit {
			bestIdx = idx
			bestFit = fitnesses[idx].Combined
		}
	}

	return pop[bestIdx].Clone()
}

