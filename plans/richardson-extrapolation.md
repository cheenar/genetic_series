# Richardson Extrapolation for Slow-Converging Series

## The Problem

The engine is blind to slow-converging series. The Leibniz formula for pi:

```
pi = 4 * sum_{n=0}^{inf} (-1)^n / (2n+1)
```

is one of the simplest and most famous series in mathematics. The engine has all the
operators needed to represent it (OpAltSign, OpMul, OpAdd, constants). But it can never
*find* it, because the fitness function can't tell it apart from noise.

The Leibniz series converges as O(1/n). With the engine's default 512 maxterms:

| Terms   | Correct digits |
|---------|----------------|
| 512     | 3.2            |
| 10,000  | 4.5            |
| 100,000 | ~5.5           |

At 3.2 digits, Leibniz loses to random coincidences that hit 4-5 digits by luck. The
fitness function scores candidates by comparing the raw partial sum to the target. It
never asks "where is this series heading?" -- only "where is it right now?"

This is a fundamental gap. Any series that converges slower than ~1 digit per 50 terms
is invisible to the engine at 512 maxterms.


## The Idea: Extrapolate the Limit

We already collect partial sums at power-of-2 checkpoints: S_1, S_2, S_4, S_8, ...,
S_256, S_512. For Leibniz, these values oscillate around pi and close in on it. We can
use this sequence to *estimate* what the series converges to, without waiting for it to
actually get there.

The technique is called **Richardson extrapolation**. It works by assuming the error has
a known asymptotic form and canceling out the leading error terms using multiple partial
sums.


## Richardson Extrapolation Explained

### The Setup

Suppose a series has partial sum S_N that approaches some limit L, with error that
decreases as a power of 1/N:

```
S_N = L + c/N^p + O(1/N^{p+1})
```

where:
- L is the true limit (what we want)
- c is some unknown constant
- p is the order of convergence (p=1 for Leibniz)

We don't know L or c. But we have S_N at multiple values of N.

### One Level of Extrapolation

Take two checkpoints at N and 2N terms. We have two equations:

```
S_N   = L + c/N^p
S_2N  = L + c/(2N)^p = L + c/(2^p * N^p)
```

Two equations, two unknowns (L and c). Solve for L:

```
S_2N - S_N = c/N^p * (1/2^p - 1)
c/N^p = (S_2N - S_N) / (1/2^p - 1)
L = S_N - c/N^p = S_N - (S_2N - S_N) / (1/2^p - 1)
```

For p=1 (O(1/N) convergence, like Leibniz):

```
L = S_N - (S_2N - S_N) / (1/2 - 1) = S_N + 2*(S_2N - S_N) = 2*S_2N - S_N
```

### Concrete Example: Leibniz

With 512-term evaluation, our last two power-of-2 checkpoints are:

```
S_256 = 3.13766...   (error ≈ 0.004)
S_512 = 3.13964...   (error ≈ 0.002)
```

One level of Richardson extrapolation (p=1):

```
L ≈ 2 * S_512 - S_256 = 2 * 3.13964 - 3.13766 = 3.14162
```

That's ~4 correct digits of pi, up from 2.7 raw. We doubled the effective accuracy
without computing any additional terms.

### Two Levels of Extrapolation

After one level, we've eliminated the c/N term. The remaining error is O(1/N^2):

```
L_1(N)  = 2*S_2N - S_N         ≈ L + d/N^2
L_1(2N) = 2*S_4N - S_2N        ≈ L + d/(2N)^2 = L + d/(4N^2)
```

Apply Richardson again to cancel the d/N^2 term:

```
L_2 = (4 * L_1(2N) - L_1(N)) / 3
```

This gives an estimate accurate to O(1/N^3). With three checkpoint pairs, we can do
this. Each additional level roughly doubles the number of correct digits.

### General Formula (Romberg-style table)

Given checkpoints S_1, S_2, S_4, ..., S_{2^k}, build a triangular table:

```
Column 0 (raw):     S_1,  S_2,  S_4,  S_8,  ..., S_{2^k}
Column 1 (order 1): R_1,  R_2,  R_3,  ...,  R_{k}
Column 2 (order 2): R'_1, R'_2, ...,  R'_{k-1}
...
```

Where each entry is computed from the two entries to its left:

```
R[i][j] = (2^j * R[i][j-1] - R[i-1][j-1]) / (2^j - 1)
```

Wait -- that formula assumes O(1/N) convergence at each level, which uses powers of 2.
More precisely, for the standard Richardson table with halving step sizes:

```
R[i][0] = S_{2^i}                                    (raw checkpoints)
R[i][j] = (4^j * R[i][j-1] - R[i-1][j-1]) / (4^j - 1)   for j >= 1
```

This is the Romberg method (applied to sequence limits rather than integrals, but the
math is identical). The bottom-right entry of the table is the best estimate.

**However**, the 4^j factor assumes the error expands in powers of 1/N^2 (even powers
only), which is true for the trapezoidal rule but NOT necessarily for general series.

For a general series with error c_1/N + c_2/N^2 + c_3/N^3 + ..., the correct
Richardson table is:

```
R[i][0] = S_{2^i}
R[i][j] = (2^j * R[i][j-1] - R[i-1][j-1]) / (2^j - 1)   for j >= 1
```

This cancels one power of 1/N at each level.


## What Convergence Orders Look Like

Different series have different convergence rates:

| Series type              | Error     | p   | Example                              |
|--------------------------|-----------|-----|--------------------------------------|
| Alternating, O(1/n)      | c/N       | 1   | Leibniz: 4*sum (-1)^n/(2n+1)        |
| Alternating, O(1/n^2)    | c/N^2     | 2   | sum (-1)^n/n^2 (Catalan-related)     |
| Geometric, ratio r       | c*r^N     | exp | Most engine-discovered series         |
| Hypergeometric           | c*r^N/N^a | exp | Ramanujan-type series                |

Richardson extrapolation helps most for polynomial convergence (p=1,2,3). For geometric
convergence (like Ramanujan), the raw partial sum is already excellent and extrapolation
adds little.

This is exactly the gap we need to fill: the engine already handles geometric convergence
well (those series score high with few terms). It's the polynomial-convergence series
that get lost.


## Detecting the Convergence Order

Before extrapolating, we need to estimate p. From three consecutive checkpoints at N,
2N, 4N:

```
d_1 = |S_2N - S_N|
d_2 = |S_4N - S_2N|
ratio = d_2 / d_1
```

For O(1/N^p) convergence: ratio ≈ 1/2^p.
For geometric convergence (ratio r): ratio ≈ r^N (decreases much faster).

```
if ratio ≈ 0.5:    p ≈ 1  (O(1/N) convergence)
if ratio ≈ 0.25:   p ≈ 2  (O(1/N^2) convergence)
if ratio ≈ 0.125:  p ≈ 3
if ratio < 0.01:   geometric convergence (extrapolation less useful)
```

We can estimate p as:

```
p = -log2(ratio) = -log(ratio) / log(2)
```

If p is close to a positive integer, use Richardson. If ratio is very small (geometric),
skip extrapolation and use the raw partial sum (which is already good).


## Implementation Plan

### Step 1: Add extrapolated limit to EvalResult

Currently `EvalResult` has `PartialSum` and `ConvergenceRate`. Add:

```go
type EvalResult struct {
    PartialSum        *big.Float
    ExtrapolatedLimit *big.Float  // NEW: Richardson-extrapolated estimate
    TermsComputed     int64
    Converged         bool
    ConvergenceRate   float64
    ConvergenceOrder  float64     // NEW: estimated p (1=O(1/N), 2=O(1/N^2), etc.)
    OK                bool
}
```

### Step 2: Implement Richardson extrapolation

After collecting checkpoints, build the Richardson table:

```go
func richardsonExtrapolate(checkpoints []checkpoint, prec uint) (*big.Float, float64) {
    n := len(checkpoints)
    if n < 3 {
        return nil, 0
    }

    // Estimate convergence order p from last 3 checkpoints.
    // d1 = |S_{2^k} - S_{2^{k-1}}|, d2 = |S_{2^{k-1}} - S_{2^{k-2}}|
    // ratio = d1/d2, p = -log2(ratio)
    d1 = |checkpoints[n-1].sum - checkpoints[n-2].sum|
    d2 = |checkpoints[n-2].sum - checkpoints[n-3].sum|
    ratio = d1 / d2
    p = -log2(ratio)

    // If geometric convergence (p > 5 or ratio < 0.01), skip extrapolation.
    if p > 5 || ratio < 0.01 {
        return checkpoints[n-1].sum, p
    }

    // Round p to nearest integer for Richardson (p=1, 2, or 3).
    pInt = round(p)

    // Build Richardson table using last few checkpoints.
    // R[i][0] = checkpoint sums
    // R[i][j] = (2^(j*pInt) * R[i][j-1] - R[i-1][j-1]) / (2^(j*pInt) - 1)
    // Return bottom-right entry.
    ...
}
```

### Step 3: Use extrapolated limit in fitness scoring

In `ComputeFitness`, if `ExtrapolatedLimit` is set, score against it:

```go
// In ComputeFitness:
value := result.PartialSum
if result.ExtrapolatedLimit != nil {
    value = result.ExtrapolatedLimit
}
correctDigits := countCorrectDigits(value, target)
```

This means Leibniz would be scored on its extrapolated limit (~4-6 digits at 512 terms)
rather than its raw partial sum (~3 digits). This makes it competitive in the fitness
tournament without changing the evaluation budget.

### Step 4: Also apply to the float64 fast path

The float64 evaluator (`EvaluateCandidateF64`) uses a ring buffer of 3 checkpoint sums.
This is exactly enough for one level of Richardson extrapolation:

```go
// In EvaluateCandidateF64, after collecting checkpoints:
// s0, s1, s2 are partial sums at N, 2N, 4N
d1 := math.Abs(s2 - s1)
d2 := math.Abs(s1 - s0)
if d2 > 0 {
    ratio := d1 / d2
    if ratio > 0.01 && ratio < 0.99 {
        // Polynomial convergence, apply Richardson
        extrapolated := 2*s2 - s1  // first-order extrapolation
    }
}
```


## Risks and Mitigations

### False positives

A divergent or oscillating series might produce checkpoint values that, when
extrapolated, accidentally land near the target. This could promote garbage formulas.

**Mitigation:** Only extrapolate if `Converged == true` (the existing convergence check
passes). Also require that the extrapolated value is within a reasonable neighborhood
of the raw partial sum -- if extrapolation moves the estimate by more than, say, 10x
the last checkpoint difference, it's probably unreliable.

### Wrong convergence order

If we estimate p=1 but the true convergence is p=2 (or vice versa), the extrapolation
will be less accurate but not catastrophically wrong. Richardson with the wrong p still
improves on the raw partial sum; it just doesn't improve as much as it could.

**Mitigation:** Use the estimated p but don't go beyond 2 levels of extrapolation. The
returns diminish quickly and the risk of instability grows.

### Geometric convergence (already fast)

For series that already converge geometrically (most engine-discovered series),
Richardson extrapolation adds almost nothing. The raw partial sum is already close to the
limit. Applying extrapolation shouldn't hurt (it would just return approximately the same
value), but it wastes a small amount of computation.

**Mitigation:** Skip extrapolation when the convergence ratio is below 0.01 (indicating
geometric or faster convergence).

### Numerical instability

Richardson extrapolation involves subtracting nearly-equal numbers, which can amplify
floating-point errors. At higher levels of the table, this becomes worse.

**Mitigation:** Limit to 2-3 levels of extrapolation. Use big.Float precision that
exceeds the expected accuracy of the extrapolated result. Don't extrapolate beyond what
the checkpoint data supports.


## Expected Impact

With Richardson extrapolation enabled, the engine should be able to:

1. **Find Leibniz-type series.** The extrapolated accuracy of ~4-6 digits (from 512
   terms) is enough to survive fitness selection and get refined over generations.

2. **Discover other slow-converging identities.** Many classical series for pi, ln(2),
   Catalan's constant, etc. have O(1/n) or O(1/n^2) convergence. Currently invisible;
   with extrapolation, they become detectable.

3. **Score existing fast series the same.** For Ramanujan-type series with geometric
   convergence, extrapolation is a no-op. No regression on the engine's current
   strengths.

The change is concentrated in `evaluate.go` and `fitness.go` -- no changes needed to
the genetic operators, selection strategies, or expression tree representation.


## Alternative Approaches Considered

### Increase maxterms

The brute-force approach: evaluate more terms. At 100,000 terms, Leibniz gives ~5.5
digits. But evaluating 100,000 terms per candidate with a population of 1000 would
multiply the engine's runtime by ~200x. Not practical for the genetic search.

### Euler transform

Apply the Euler transform to alternating series during evaluation. This converts a
slow-converging alternating series into a fast-converging one. For Leibniz specifically,
the Euler transform gives ~1 digit per term. But it only works for alternating series
and requires detecting the alternating pattern, which adds complexity. Richardson
extrapolation is more general.

### Aitken's delta-squared / Wynn's epsilon algorithm

These are general-purpose sequence acceleration methods that don't require knowing the
convergence order. They work on arbitrary convergent sequences. More powerful than
Richardson in theory, but also more complex to implement and harder to reason about
edge cases. Could be a future upgrade if Richardson proves insufficient.

### Separate slow-convergence pass

Run a second evaluation pass with 10x more terms for candidates that show slow but
steady convergence. This would be more accurate than extrapolation but slower. Could be
combined with Richardson: use extrapolation in the main loop, then do a detailed
evaluation for top candidates.
