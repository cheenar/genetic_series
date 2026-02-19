# The Good, The Bad, and The Ugly

## A Generalized Candidate Structure for Euler-Mascheroni Type Constants


## The Insight

The Euler-Mascheroni constant is defined as:

```
γ = lim_{N→∞} [ H_N - ln(N) ]
```

where `H_N = sum_{n=1}^{N} 1/n` is the harmonic series (divergent) and `ln(N)` is also
divergent. Neither side converges on its own. But their *difference* converges to γ.

This is a fundamentally different shape from what the engine can currently represent. A
`Candidate` is a single convergent series `sum f(n)`. The engine rejects anything that
diverges. But γ is not a convergent series — it's the difference between two divergent
quantities that happen to cancel almost perfectly.

The user's generalization: we can add anything convergent to this pattern without breaking
it. So the most general form is:

```
constant = (convergent series) + (divergent series) - (divergent non-series correction)
         = The Good            + The Bad             - The Ugly
```

- **The Good**: A standard convergent series. Could be zero (absent).
- **The Bad**: A divergent series whose partial sum grows without bound.
- **The Ugly**: A non-series expression (like `ln(N)`, `sqrt(N)`, `N^k`) that also grows
  without bound, but whose growth matches The Bad's divergence.

The Bad and The Ugly diverge individually. But `Bad - Ugly → finite limit`.


## Why This Matters

### Constants that need this structure

| Constant | Value | Known representation |
|----------|-------|---------------------|
| Euler-Mascheroni γ | 0.5772156649... | `H_N - ln(N)` |
| Stieltjes constants γ_k | various | `lim_{N→∞} [sum_{n=1}^{N} (ln n)^k/n - (ln N)^{k+1}/(k+1)]` |
| Mertens' constant M | 0.2614972128... | `lim_{N→∞} [sum_{p≤N} 1/p - ln(ln(N))]` (sum over primes) |
| Gregory coefficients | various | Involve divergent harmonic-type sums |

The Euler-Mascheroni constant alone is one of the most important unsolved constants in
mathematics. Whether γ is rational or irrational is unknown. New series representations
would be genuinely interesting.

### What the engine currently misses

The engine rejects any series where `Converged == false`. This means:

1. It can never discover `H_N - ln(N) → γ`, because `H_N` diverges.
2. It can never discover regularized sums where a correction term is needed.
3. It misses an entire class of constants that are defined as limits of differences.

This is a structural blind spot, not a precision issue. No amount of Richardson
extrapolation or evaluation budget fixes it. The candidate *shape* must change.


## The Three Components

### The Good (convergent series) — optional

```
G = sum_{n=start}^{inf} g_num(n) / g_den(n)
```

A standard convergent series, same as the current `Candidate`. This part is optional — for
pure Euler-Mascheroni type limits, it's absent (zero). But including it lets the engine
search for representations like:

```
γ = (convergent correction series) + lim_{N→∞} [H_N - ln(N) - correction]
```

which could converge faster than the raw `H_N - ln(N)` definition.

### The Bad (divergent series) — required

```
B_N = sum_{n=start}^{N} b_num(n) / b_den(n)
```

A divergent series. This is evaluated as a **partial sum** up to N terms (not to infinity).
The key difference from The Good: we don't wait for convergence. We evaluate B_N at the
same checkpoint values as always (N = 1, 2, 4, 8, ..., 512) but keep the partial sum at
each checkpoint rather than testing for convergence.

For Euler-Mascheroni: `b_num(n) = 1, b_den(n) = n` → harmonic series.

### The Ugly (divergent correction) — required

```
U_N = ugly(N)
```

A single expression evaluated at N (not a sum over n). This is an `ExprNode` that takes N
as input and returns the correction value. It grows in a way that matches The Bad's
divergence.

For Euler-Mascheroni: `ugly(N) = ln(N)`.

The Ugly is NOT a series — it's a single function evaluation. This is what makes it
structurally different from The Bad. The Bad accumulates term by term; The Ugly is computed
directly from N.


## Evaluation

### The limit computation

At each checkpoint N, we compute:

```
L_N = G_N + B_N - U_N
```

where:
- `G_N` = partial sum of The Good up to N terms (or 0 if absent)
- `B_N` = partial sum of The Bad up to N terms
- `U_N` = The Ugly evaluated at N

If the candidate represents a real identity, `L_N` should converge as N grows, even though
`B_N` and `U_N` individually diverge.

### Convergence detection

We apply the same checkpoint-based convergence test to the *combined* sequence `L_1, L_2,
L_4, ..., L_512`. If the differences `|L_{2N} - L_N|` shrink, the combined expression
converges, even though the individual parts don't.

### Fitness scoring

Score against the target using `L_N` at the last checkpoint (or better: the Richardson-
extrapolated limit of the `L_N` sequence, combining both features).


## Proposed Candidate Structure

```go
// CandidateShape describes what kind of candidate this is.
type CandidateShape int

const (
    ShapeSeries    CandidateShape = iota // Standard: sum_{n}^{inf} f(n) (current)
    ShapeGoodBadUgly                     // Limit: G + B_N - U(N)
)

type Candidate struct {
    // --- Standard series (The Good, or the only series for ShapeSeries) ---
    Numerator   expr.ExprNode
    Denominator expr.ExprNode
    Start       int64

    // --- Good/Bad/Ugly fields (only used when Shape == ShapeGoodBadUgly) ---
    Shape       CandidateShape

    // The Bad: divergent series
    BadNumerator   expr.ExprNode
    BadDenominator expr.ExprNode
    BadStart       int64

    // The Ugly: correction term (single expression, not a series)
    Ugly expr.ExprNode
}
```

When `Shape == ShapeSeries` (default/zero value), the candidate behaves exactly as it does
today. The Bad/Ugly fields are nil and ignored. This preserves full backward compatibility.

When `Shape == ShapeGoodBadUgly`:
- Numerator/Denominator/Start = The Good (convergent series). If Numerator is nil, The Good
  is absent (zero).
- BadNumerator/BadDenominator/BadStart = The Bad (divergent series).
- Ugly = The Ugly (correction expression, evaluated at N).


## Evaluation Changes

### EvaluateCandidate

```go
func EvaluateCandidate(c *Candidate, maxTerms int64, prec uint) EvalResult {
    if c.Shape == ShapeGoodBadUgly {
        return evaluateGoodBadUgly(c, maxTerms, prec)
    }
    // ... existing evaluation logic for ShapeSeries ...
}

func evaluateGoodBadUgly(c *Candidate, maxTerms int64, prec uint) EvalResult {
    // Evaluate all three components in lockstep.
    // At each checkpoint N:
    //   G_N = partial sum of Good series (0 if Good is absent)
    //   B_N = partial sum of Bad series
    //   U_N = Ugly(N)
    //   L_N = G_N + B_N - U_N
    //
    // Check convergence of the L_N sequence.
    // Return L_N at last checkpoint as PartialSum.
}
```

The evaluation loop runs The Good and The Bad series in parallel (same loop counter), and
evaluates The Ugly at each checkpoint.

### Key difference from standard evaluation

For standard series, we check `Converged` on the partial sum of the series itself. For
GoodBadUgly, we check convergence of the *combined* `L_N` sequence. The Bad series is
*expected* to diverge — that's fine as long as `L_N` converges.

### Float64 fast path

Same idea: evaluate The Good, The Bad, and The Ugly using `EvalF64`, combine at
checkpoints, test convergence of the combined sequence.


## Fitness Changes

### Relaxed divergence checks

Currently, `ComputeFitness` rejects candidates where:
- Denominator doesn't depend on n (terms don't shrink → diverges)
- `!result.Converged`

For GoodBadUgly candidates, we need to relax these checks:
- The Bad's denominator may not shrink the terms (e.g., `1/n` diverges).
- The Bad alone won't converge. Only the combined `L_N` needs to converge.

The fitness function should route to a separate check for GoodBadUgly:

```go
if c.Shape == ShapeGoodBadUgly {
    // Check: combined L_N converges
    // Check: L_N is close to target
    // Don't reject Bad for diverging — that's the point
} else {
    // ... existing checks ...
}
```

### Complexity scoring

A GoodBadUgly candidate has more nodes (three components instead of one). The complexity
penalty should account for total node count across all components:

```go
func (c *Candidate) Complexity() int {
    if c.Shape == ShapeGoodBadUgly {
        total := nodeCount(c.BadNumerator) + nodeCount(c.BadDenominator)
        total += nodeCount(c.Ugly)
        if c.Numerator != nil {
            total += nodeCount(c.Numerator) + nodeCount(c.Denominator)
        }
        return total
    }
    return c.Numerator.NodeCount() + c.Denominator.NodeCount()
}
```


## Genetic Operators

### Initialization

Create random GoodBadUgly candidates alongside regular ones. The population should be a
mix — most candidates are standard series, but some fraction (e.g., 10-20%) are GoodBadUgly
shape.

For a random GoodBadUgly candidate:
- **The Good**: Either absent (nil) or a small random convergent series.
- **The Bad**: A random series (no convergence requirement). Bias toward simple divergent
  forms like `1/n^k`, `1/(n * f(n))`, etc.
- **The Ugly**: A random expression using `ln(N)`, `sqrt(N)`, `N^k`, etc. The operator pool
  for The Ugly should be biased toward smooth, slowly-growing functions.

### Mutation

Mutations on GoodBadUgly candidates can target any component:
1. Mutate The Good (same as standard series mutation)
2. Mutate The Bad (same, but no convergence pressure)
3. Mutate The Ugly (standard expression tree mutation)
4. Add/remove The Good component
5. Swap The Bad and The Good (try the other way around)

### Crossover

Options for crossing two GoodBadUgly candidates:
1. Swap The Good components
2. Swap The Bad components
3. Swap The Ugly components
4. Swap matched pairs (Bad+Ugly from one parent, Good from the other)

Cross-shape crossover (between a standard and a GoodBadUgly candidate):
- Take the standard candidate's series as The Good, keep the other parent's Bad and Ugly.

### Shape conversion

Allow candidates to change shape during evolution:
- A standard series that plateaus at low fitness could be promoted to GoodBadUgly by adding
  a random Bad+Ugly pair.
- A GoodBadUgly candidate whose Bad and Ugly are both near-zero could be demoted to a
  standard series (just The Good).


## The Ugly: Expression Pool

The Ugly is a single expression evaluated at N. It needs operators that produce the right
growth rates to cancel divergent series. Key building blocks:

| Expression | Growth | Cancels |
|-----------|--------|---------|
| `ln(N)` | O(log N) | Harmonic series `sum 1/n` |
| `ln(ln(N))` | O(log log N) | `sum 1/(n ln n)` |
| `sqrt(N)` | O(N^{1/2}) | `sum 1/(2 sqrt(n))` |
| `N^k` | O(N^k) | `sum n^{k-1}` type |
| `N * ln(N)` | O(N log N) | `sum ln(n)` |

The existing `OpLn`, `OpSqrt`, `OpPow`, `OpMul`, `OpAdd` operators are sufficient.
No new operators needed — just a different context (evaluated at N, not summed over n).


## Worked Example: Euler-Mascheroni

Target: `γ = 0.5772156649015328606...`

### The candidate

```
Shape: ShapeGoodBadUgly

The Good: absent (nil)
The Bad:  sum_{n=1}^{N} 1/n        (BadNumerator=1, BadDenominator=n)
The Ugly: ln(N)                     (Ugly = Ln(Var(n)))
```

### Evaluation at checkpoints

| N | B_N = H_N | U_N = ln(N) | L_N = B_N - U_N |
|---|-----------|-------------|------------------|
| 1 | 1.000 | 0.000 | 1.0000 |
| 2 | 1.500 | 0.693 | 0.8069 |
| 4 | 2.083 | 1.386 | 0.6971 |
| 8 | 2.718 | 2.079 | 0.6382 |
| 16 | 3.380 | 2.773 | 0.6077 |
| 32 | 4.058 | 3.466 | 0.5922 |
| 64 | 4.744 | 4.159 | 0.5849 |
| 128 | 5.437 | 4.852 | 0.5812 |
| 256 | 6.124 | 5.545 | 0.5793 |
| 512 | 6.811 | 6.238 | 0.5783 |

The L_N sequence converges: 1.000 → 0.807 → 0.697 → ... → 0.578. At 512 terms, it's
about 1.1 digits of γ. Not great raw accuracy, but:

1. The convergence is clear and steady — passes the checkpoint test.
2. Richardson extrapolation would boost this significantly (this is O(1/N) convergence,
   exactly the case Richardson handles best).
3. Combined with Richardson, we'd get ~3-5 digits from 512 terms — enough to survive
   fitness selection.


## Implementation Phases

### Phase 1: Core structure (required)

1. Add `CandidateShape`, `BadNumerator`, `BadDenominator`, `BadStart`, `Ugly` to Candidate.
2. Add `evaluateGoodBadUgly` function in evaluate.go.
3. Route through EvaluateCandidate based on Shape.
4. Update `String()`, `LaTeX()`, `Clone()`, `Complexity()`, `NodeCount()`.
5. Update fitness to handle GoodBadUgly (relaxed divergence checks).

### Phase 2: Genetic operators (required)

1. Random GoodBadUgly candidate generation.
2. Mutation operators for each component.
3. Crossover operators (within-shape and cross-shape).
4. Population initialization with mixed shapes.

### Phase 3: Integration (required)

1. Update the engine to pass Shape information through the pipeline.
2. Update the hall of fame, reporting, and LaTeX output for GoodBadUgly candidates.
3. Add a config option for the fraction of GoodBadUgly candidates in the population.

### Phase 4: Refinement (optional, after testing)

1. Shape conversion during evolution (promote/demote candidates).
2. Specialized initialization heuristics for The Ugly (bias toward ln, sqrt, etc.).
3. Combine with Richardson extrapolation for maximum slow-convergence detection.
4. Multi-level nesting: The Good could itself be a GoodBadUgly candidate (probably not
   worth the complexity, but theoretically possible).


## Risks

### Explosion of the search space

Three components instead of one means roughly 3x the parameters. The search space grows
combinatorially. Most random GoodBadUgly candidates will be garbage — two divergent things
that don't cancel.

**Mitigation:** Keep GoodBadUgly candidates as a minority of the population (10-20%). The
standard series shape is still the workhorse. GoodBadUgly is a specialized tool for a
specific class of constants.

### Numerical cancellation issues

`B_N - U_N` involves subtracting two large, nearly-equal numbers. At N=512, both `H_512`
and `ln(512)` are around 6.2. The difference is ~0.58. This means we lose about 1 digit
of precision to cancellation.

**Mitigation:** Use sufficient big.Float precision (the existing 2048-bit default gives
~600 digits, so losing 1-2 is fine). For the float64 fast path, this is more concerning —
at N=512, we have about 14 usable digits out of 15-16 total. Enough for the screening
pass but just barely.

### The Ugly is too expressive

If The Ugly can be any expression tree, the search space is huge. Most random trees won't
produce growth rates that match any divergent series.

**Mitigation:** Constrain The Ugly's operator pool. Start with just `{ln, sqrt, pow, mul,
add, const, var}` — no factorial, no trig, no binomial. Keep trees small (depth 3-4 max).
The most useful Ugly expressions are simple: `ln(N)`, `ln(ln(N))`, `sqrt(N)`, `c * N^k`.


## Relation to Other Planned Features

### Richardson extrapolation (plans/richardson-extrapolation.md)

Richardson extrapolation and GoodBadUgly are complementary:
- Richardson helps with slow-converging *series* (like Leibniz).
- GoodBadUgly handles *limits of differences* (like Euler-Mascheroni).
- Combined: the `L_N` sequence from GoodBadUgly converges slowly (O(1/N) for γ), and
  Richardson can accelerate it.

Implement Richardson first — it's simpler and benefits standard series too. Then add
GoodBadUgly, which can immediately benefit from Richardson.

### Sum of Two Series (plans/diversity.md, idea #5)

GoodBadUgly subsumes "Sum of Two Series." If both series converge, set The Ugly = 0 and
you have The Good + The Bad = two convergent series summed. The GoodBadUgly structure is
strictly more general.

### Product series / continued fractions

These are orthogonal features. A product series or continued fraction could also have
GoodBadUgly structure in theory, but that's a much later concern.
