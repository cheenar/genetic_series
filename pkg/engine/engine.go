package engine

import (
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/wildfunctions/genetic_series/pkg/constants"
	"github.com/wildfunctions/genetic_series/pkg/pool"
	"github.com/wildfunctions/genetic_series/pkg/series"
	"github.com/wildfunctions/genetic_series/pkg/strategy"
)

// Engine runs the evolutionary search.
type Engine struct {
	cfg       Config
	pool      pool.Pool
	strategy  strategy.Strategy
	target    *big.Float
	targetF64 float64
	rng       *rand.Rand
}

// New creates a new engine from the given config.
func New(cfg Config) (*Engine, error) {
	p, err := pool.Get(cfg.Pool)
	if err != nil {
		return nil, err
	}
	s, err := strategy.Get(cfg.Strategy)
	if err != nil {
		return nil, err
	}

	// If a seed formula was provided, pass it to the strategy.
	if cfg.SeedFormula != "" {
		type seedable interface {
			SetSeedFormula(string) error
		}
		if ss, ok := s.(seedable); ok {
			if err := ss.SetSeedFormula(cfg.SeedFormula); err != nil {
				return nil, fmt.Errorf("invalid seed formula: %w", err)
			}
		} else {
			return nil, fmt.Errorf("strategy %q does not support -seed-formula", cfg.Strategy)
		}
	}

	c := constants.Get(cfg.Target)
	if c == nil {
		return nil, fmt.Errorf("unknown target constant: %s (available: %v)", cfg.Target, constants.Names())
	}

	seed := cfg.Seed
	if seed == 0 {
		seed = rand.Int63()
	}

	return &Engine{
		cfg:       cfg,
		pool:      p,
		strategy:  s,
		target:    c.Value,
		targetF64: c.Float64Value,
		rng:       rand.New(rand.NewSource(seed)),
	}, nil
}

// Run executes the evolutionary loop and returns the final report.
func (e *Engine) Run() FinalReport {
	runTimestamp := fmt.Sprintf("%d", time.Now().Unix())
	var hallOfFame []AttemptResult
	var genReports []GenerationReport
	totalGensUsed := 0
	attempt := 0
	tabuSet := map[string]bool{}

	// Track best across all attempts
	var globalBest *series.Candidate
	var globalBestFitness series.Fitness
	var globalBestResult series.EvalResult
	globalBestFitness.Combined = -1e18

	genBudget := "unlimited"
	if e.cfg.Generations > 0 {
		genBudget = fmt.Sprintf("%d", e.cfg.Generations)
	}
	fmt.Fprintf(os.Stderr, "Timestamp: [%s] Starting target %s, pool %s, strategy %s, population %d, %s gen budget, stagnation %d, workers %d, seed %d\n",
		runTimestamp, e.cfg.Target, e.cfg.Pool, e.cfg.Strategy, e.cfg.Population, genBudget, e.cfg.StagnationLimit, e.cfg.Workers, e.cfg.Seed)

	unlimited := e.cfg.Generations <= 0
	for unlimited || totalGensUsed < e.cfg.Generations {
		attempt++
		fmt.Fprintf(os.Stderr, "\n=== Attempt %d ===\n", attempt)

		population := e.strategy.Initialize(e.pool, e.rng, e.cfg.Population)

		var bestThisAttempt *series.Candidate
		var bestThisAttemptFitness series.Fitness
		var bestThisAttemptResult series.EvalResult
		bestThisAttemptFitness.Combined = -1e18
		gensSinceImprovement := 0
		bestFoundAtGen := 0
		attemptGens := 0

		for unlimited || totalGensUsed < e.cfg.Generations {
			fitnesses, results := e.evaluatePopulation(population, tabuSet)

			// Find best and second-best in this generation
			bestIdx, secondIdx := 0, -1
			var avgFit float64
			for i, f := range fitnesses {
				avgFit += f.Combined
				if f.Combined > fitnesses[bestIdx].Combined {
					secondIdx = bestIdx
					bestIdx = i
				} else if secondIdx == -1 || f.Combined > fitnesses[secondIdx].Combined {
					if i != bestIdx {
						secondIdx = i
					}
				}
			}
			avgFit /= float64(len(fitnesses))

			improved := fitnesses[bestIdx].Combined > bestThisAttemptFitness.Combined
			if improved {
				bestThisAttempt = population[bestIdx].Clone()
				bestThisAttemptFitness = fitnesses[bestIdx]
				bestThisAttemptResult = results[bestIdx]
				bestFoundAtGen = attemptGens
				gensSinceImprovement = 0
			} else {
				gensSinceImprovement++
			}

			report := GenerationReport{
				Generation:    attemptGens,
				BestFitness:   fitnesses[bestIdx],
				BestCandidate: population[bestIdx].String(),
				BestLaTeX:     population[bestIdx].LaTeX(),
				AvgFitness:    avgFit,
			}
			if results[bestIdx].OK && results[bestIdx].PartialSum != nil {
				report.BestPartialSum = results[bestIdx].PartialSum.Text('g', 20)
			}

			if e.cfg.Verbose {
				WriteTextReport(os.Stderr, report)
			} else if improved {
				fmt.Fprintf(os.Stderr, "[gen %d] NEW BEST %.1f digits | fitness %.4f\n",
					attemptGens, bestThisAttemptFitness.CorrectDigits, bestThisAttemptFitness.Combined)
				fmt.Fprintf(os.Stderr, "  #1: %s\n", bestThisAttempt.String())
				if secondIdx >= 0 && results[secondIdx].OK {
					fmt.Fprintf(os.Stderr, "  #2: %.1f digits | %s\n",
						fitnesses[secondIdx].CorrectDigits, population[secondIdx].String())
				}
			} else if attemptGens%20 == 0 {
				fmt.Fprintf(os.Stderr, "[gen %d]\n", attemptGens)
				if bestThisAttempt != nil {
					fmt.Fprintf(os.Stderr, "  #1: %.1f digits | %s\n",
						bestThisAttemptFitness.CorrectDigits, bestThisAttempt.String())
				}
				if secondIdx >= 0 && results[secondIdx].OK {
					fmt.Fprintf(os.Stderr, "  #2: %.1f digits | %s\n",
						fitnesses[secondIdx].CorrectDigits, population[secondIdx].String())
				}
			}
			genReports = append(genReports, report)

			totalGensUsed++
			attemptGens++

			// Hit the digit cap — nothing left to find, move on.
			if bestThisAttemptFitness.CorrectDigits >= float64(series.MaxDigits) {
				fmt.Fprintf(os.Stderr, "[gen %d] Hit %d digit cap, done\n",
					attemptGens, series.MaxDigits)
				break
			}

			// Check stagnation — patience scales with best digits found so far.
			// Low-digit matches get a short leash; high-digit matches get full patience.
			if e.cfg.StagnationLimit > 0 {
				digits := bestThisAttemptFitness.CorrectDigits
				scale := digits / 10.0
				if scale > 1.0 {
					scale = 1.0
				}
				effectiveLimit := int(float64(e.cfg.StagnationLimit) * scale)
				if effectiveLimit < 20 {
					effectiveLimit = 20
				}
				if gensSinceImprovement >= effectiveLimit {
					fmt.Fprintf(os.Stderr, "[gen %d] Stagnated after %d generations (%.1f digits, patience %d)\n",
						attemptGens, gensSinceImprovement, digits, effectiveLimit)
					break
				}
			}

			// Evolve
			population = e.strategy.Evolve(population, fitnesses, e.pool, e.rng)
		}

		// Save attempt result
		ar := AttemptResult{
			Attempt:        attempt,
			Generations:    attemptGens,
			BestFoundAtGen: bestFoundAtGen,
			Timestamp:      time.Now().UTC(),
		}
		if bestThisAttempt != nil {
			ar.BestCandidate = bestThisAttempt.String()
			ar.BestLaTeX = bestThisAttempt.LaTeX()
			ar.BestFitness = bestThisAttemptFitness
			if bestThisAttemptResult.OK && bestThisAttemptResult.PartialSum != nil {
				ar.BestPartialSum = bestThisAttemptResult.PartialSum.Text('g', 20)
			}
		}
		hallOfFame = append(hallOfFame, ar)

		// Add best candidate to tabu set so future restarts avoid it
		if bestThisAttempt != nil {
			s := bestThisAttempt.String()
			if !tabuSet[s] {
				tabuSet[s] = true
				fmt.Fprintf(os.Stderr, "Tabu: added %q\n", s)
			}
		}

		// Update global best
		if bestThisAttempt != nil && bestThisAttemptFitness.Combined > globalBestFitness.Combined {
			globalBest = bestThisAttempt
			globalBestFitness = bestThisAttemptFitness
			globalBestResult = bestThisAttemptResult
		}

		WriteHallOfFame(os.Stderr, hallOfFame)

		// Write LaTeX hall of fame after each attempt so it survives Ctrl+C
		if e.cfg.OutDir != "" {
			base := fmt.Sprintf("%s_%s_%s_%s", e.cfg.Target, e.cfg.Pool, e.cfg.Strategy, runTimestamp)
			tmpDir := os.TempDir()
			tmpTex := filepath.Join(tmpDir, base+".tex")

			f, createErr := os.Create(tmpTex)
			if createErr != nil {
				fmt.Fprintf(os.Stderr, "error creating %s: %v\n", tmpTex, createErr)
			} else {
				WriteHallOfFameLatex(f, hallOfFame, e.cfg, e.target)
				f.Close()

				// Compile to PDF if pdflatex is available
				pdflatex := os.Getenv("PDFLATEX_PATH")
				if pdflatex == "" {
					pdflatex, _ = exec.LookPath("pdflatex")
				}
				if pdflatex != "" {
					cmd := exec.Command(pdflatex, "-interaction=nonstopmode", base+".tex")
					cmd.Dir = tmpDir
					pdfOut, pdfErr := cmd.CombinedOutput()
					if pdfErr != nil {
						fmt.Fprintf(os.Stderr, "pdflatex failed: %v\n%s\n", pdfErr, pdfOut)
					}
				}

				// Copy outputs to outdir using absolute path
				absOut, _ := filepath.Abs(e.cfg.OutDir)
				for _, ext := range []string{".tex", ".pdf"} {
					src := filepath.Join(tmpDir, base+ext)
					if _, err := os.Stat(src); err == nil {
						dst := filepath.Join(absOut, base+ext)
						if err := copyFile(src, dst); err != nil {
							fmt.Fprintf(os.Stderr, "error writing %s: %v\n", dst, err)
						} else {
							fmt.Fprintf(os.Stderr, "Wrote %s\n", dst)
						}
					}
				}
				// Clean up all temp files
				for _, ext := range []string{".tex", ".aux", ".log", ".pdf"} {
					os.Remove(filepath.Join(tmpDir, base+ext))
				}
			}
		}

		// If global best hit the digit cap, no point restarting
		if globalBestFitness.CorrectDigits >= float64(series.MaxDigits) {
			fmt.Fprintf(os.Stderr, "Global best hit %d digit cap, stopping\n", series.MaxDigits)
			break
		}
	}

	// Dedup and cap attempts for the JSON report
	dedupedAttempts := dedupAttempts(sortByDigits(hallOfFame))
	if len(dedupedAttempts) > maxHallOfFame {
		dedupedAttempts = dedupedAttempts[:maxHallOfFame]
	}

	finalReport := FinalReport{
		Config:      e.cfg,
		BestFitness: globalBestFitness,
		Attempts:    dedupedAttempts,
	}

	if e.cfg.Verbose {
		finalReport.Generations = genReports
	}

	if globalBest != nil {
		finalReport.BestCandidate = globalBest.String()
		finalReport.BestLaTeX = globalBest.LaTeX()
		if globalBestResult.OK && globalBestResult.PartialSum != nil {
			finalReport.BestPartialSum = globalBestResult.PartialSum.Text('g', 20)
		}
	}

	return finalReport
}

// evaluatePopulation evaluates all candidates in parallel, using a two-phase
// float64 fast path when F64PromotionThreshold > 0. Phase 1 evaluates all
// candidates at float64 speed. Phase 2 promotes only candidates that cleared
// the digit threshold to the expensive big.Float path.
func (e *Engine) evaluatePopulation(pop []*series.Candidate, tabuSet map[string]bool) ([]series.Fitness, []series.EvalResult) {
	n := len(pop)
	fitnesses := make([]series.Fitness, n)
	results := make([]series.EvalResult, n)

	// Pre-compute string representations once for tabu lookups.
	strs := make([]string, n)
	for i, c := range pop {
		strs[i] = c.String()
	}

	threshold := e.cfg.F64PromotionThreshold
	if threshold <= 0 {
		// Disabled — fall through to big.Float for everyone.
		e.evaluateBigFloat(pop, fitnesses, results, nil, tabuSet, strs)
		return fitnesses, results
	}

	workers := e.cfg.Workers
	if workers <= 0 {
		workers = 1
	}

	// Phase 1: float64 eval for ALL candidates.
	promote := make([]bool, n)

	type job struct {
		idx       int
		candidate *series.Candidate
		str       string
	}

	jobs := make(chan job, n)
	var wg sync.WaitGroup

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				if tabuSet[j.str] {
					fitnesses[j.idx] = series.WorstFitness()
					continue
				}
				r64 := series.EvaluateCandidateF64(j.candidate, e.cfg.MaxTerms)
				f64 := series.ComputeFitnessF64(j.candidate, r64, e.targetF64, e.cfg.Weights)
				fitnesses[j.idx] = f64
				if f64.CorrectDigits >= threshold {
					promote[j.idx] = true
				}
			}
		}()
	}

	for i, c := range pop {
		jobs <- job{idx: i, candidate: c, str: strs[i]}
	}
	close(jobs)
	wg.Wait()

	// Phase 2: big.Float eval for promoted candidates only.
	e.evaluateBigFloat(pop, fitnesses, results, promote, tabuSet, strs)

	return fitnesses, results
}

// evaluateBigFloat runs big.Float evaluation on selected candidates.
// If promote is nil, all candidates are evaluated. Otherwise only promote[i]==true.
// strs contains pre-computed String() representations for tabu lookups.
func (e *Engine) evaluateBigFloat(pop []*series.Candidate, fitnesses []series.Fitness, results []series.EvalResult, promote []bool, tabuSet map[string]bool, strs []string) {
	workers := e.cfg.Workers
	if workers <= 0 {
		workers = 1
	}

	type job struct {
		idx       int
		candidate *series.Candidate
		str       string
	}

	jobs := make(chan job, len(pop))
	var wg sync.WaitGroup

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				if tabuSet[j.str] {
					fitnesses[j.idx] = series.WorstFitness()
					continue
				}
				result := series.EvaluateCandidate(j.candidate, e.cfg.MaxTerms, e.cfg.Precision)
				fitness := series.ComputeFitness(j.candidate, result, e.target, e.cfg.Weights)
				results[j.idx] = result
				fitnesses[j.idx] = fitness
			}
		}()
	}

	for i, c := range pop {
		if promote == nil || promote[i] {
			jobs <- job{idx: i, candidate: c, str: strs[i]}
		}
	}
	close(jobs)
	wg.Wait()
}

// copyFile copies src to dst, creating or overwriting dst.
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}
