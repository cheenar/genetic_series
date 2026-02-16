package series

import (
	"math/big"
	"time"
)

// EvalResult holds the result of evaluating a candidate's partial sum.
type EvalResult struct {
	PartialSum      *big.Float
	TermsComputed   int64
	Converged       bool
	ConvergenceRate float64 // average ratio of |S_{2N} - S_N| decrease per doubling
	OK              bool
}

// evalTimeout is the maximum time allowed for evaluating a single candidate.
const evalTimeout = 100 * time.Millisecond

// EvaluateCandidate computes the partial sum of a candidate series up to maxTerms,
// using checkpoints at powers of 2 for convergence detection.
func EvaluateCandidate(c *Candidate, maxTerms int64, prec uint) EvalResult {
	sum := new(big.Float).SetPrec(prec)
	n := new(big.Float).SetPrec(prec)

	// Track partial sums at checkpoints (powers of 2)
	var checkpoints []checkpoint
	nextCheckpoint := int64(1)

	var termsComputed int64
	deadline := time.Now().Add(evalTimeout)

	for i := c.Start; i < c.Start+maxTerms; i++ {
		if time.Now().After(deadline) {
			return EvalResult{OK: false}
		}

		n.SetInt64(i)

		num, ok := c.Numerator.Eval(n, prec)
		if !ok {
			break // term failed — use partial sum so far
		}

		den, ok := c.Denominator.Eval(n, prec)
		if !ok {
			break
		}

		if den.Sign() == 0 {
			break
		}

		term := new(big.Float).SetPrec(prec).Quo(num, den)
		sum.Add(sum, term)
		termsComputed++

		// Record checkpoint at powers of 2 (relative to start)
		offset := i - c.Start + 1
		if offset == nextCheckpoint {
			checkpoints = append(checkpoints, checkpoint{
				terms: offset,
				sum:   new(big.Float).SetPrec(prec).Copy(sum),
			})
			nextCheckpoint *= 2
		}
	}

	// Need at least a few terms for a meaningful result
	if termsComputed < 4 {
		return EvalResult{OK: false}
	}

	// Compute convergence rate from checkpoints
	converged, rate := analyzeConvergence(checkpoints, prec)

	return EvalResult{
		PartialSum:      sum,
		TermsComputed:   termsComputed,
		Converged:       converged,
		ConvergenceRate: rate,
		OK:              true,
	}
}

type checkpoint struct {
	terms int64
	sum   *big.Float
}

// analyzeConvergence checks if |S_{2N} - S_N| is decreasing by a consistent factor.
func analyzeConvergence(cps []checkpoint, prec uint) (bool, float64) {
	if len(cps) < 3 {
		return false, 0
	}

	var diffs []float64
	for i := 1; i < len(cps); i++ {
		diff := new(big.Float).SetPrec(prec).Sub(cps[i].sum, cps[i-1].sum)
		diff.Abs(diff)
		f, _ := diff.Float64()
		diffs = append(diffs, f)
	}

	// Check that differences are decreasing
	if len(diffs) < 2 {
		return false, 0
	}

	var totalRatio float64
	var validRatios int
	converging := true

	for i := 1; i < len(diffs); i++ {
		if diffs[i-1] == 0 {
			// Perfect convergence at this point
			continue
		}
		ratio := diffs[i] / diffs[i-1]
		if ratio >= 1.0 {
			converging = false
		}
		totalRatio += ratio
		validRatios++
	}

	if validRatios == 0 {
		return true, 1.0 // converged exactly
	}

	avgRatio := totalRatio / float64(validRatios)
	return converging && avgRatio < 0.99, avgRatio
}
