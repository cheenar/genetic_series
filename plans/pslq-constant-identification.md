# PSLQ Constant Identification

## The Idea

When the engine finds a candidate that matches 8+ digits of *something*, don't just check
it against the target constant. Use the PSLQ algorithm to ask: is this value a linear
combination of known constants?

PSLQ takes a vector of real numbers `[x, 1, pi, e, ln(2), gamma, zeta(3), catalan]` and
finds integer coefficients `a_i` such that:

```
a_0 * x + a_1 * 1 + a_2 * pi + a_3 * e + ... = 0
```

If it finds a solution, we know what the series converges to:

```
x = -(a_1 + a_2*pi + a_3*e + ...) / a_0
```

This flips the search from "find a series for pi" to "find a series for anything
interesting, then figure out what it is."


## Why This Matters

1. **Serendipitous discovery.** A series targeting pi might stumble onto a formula for
   `(pi + e) / 2` or `3*ln(2)` by accident. Currently we'd score that as a near-miss and
   discard it. With PSLQ, we'd identify it.

2. **Multi-constant relations.** Some of the most interesting identities relate multiple
   constants. PSLQ can find `x = a*pi + b*zeta(3)` type relations.

3. **The Ramanujan Machine did exactly this.** Their pipeline: generate candidate values
   numerically, then use PSLQ to identify them against a basis of known constants.


## The Basis Vector

Use the constants we already have in `pkg/series/constants.go`:

```
basis = [candidate_value, 1, pi, e, ln(2), euler_gamma, zeta(3), catalan]
```

That's 8 elements. Drop `one_over_pi` since it's redundant with `pi` (PSLQ finds the
relation either way).

At our 512-bit precision (~154 decimal digits), we can search for coefficients up to
about 10^18 with 8 basis elements. In practice, coefficients up to 10^6 are more than
enough — most known identities have small integer coefficients.


## Precision Requirements

PSLQ needs `n * log10(M)` digits of precision, where n = basis size and M = max
coefficient. For our setup:

| Basis size | Max coefficient | Digits needed | Our precision |
|-----------|----------------|---------------|---------------|
| 8 | 1,000 | 24 | 154 (plenty) |
| 8 | 1,000,000 | 48 | 154 (plenty) |
| 12 | 1,000,000 | 72 | 154 (ok) |

We have headroom. The constraint is the candidate's partial sum precision, which depends
on how many correct digits the series gives us. A candidate with 8 correct digits can
only support PSLQ with ~8 usable digits — enough for small-coefficient relations.


## When to Run PSLQ

Not on every candidate — that's too expensive. Run it when:

1. **A candidate scores 8+ digits against any target.** This means the partial sum is
   accurate enough to be meaningful.

2. **The candidate doesn't match the primary target well.** If it already matches pi to
   20 digits, we know what it is. PSLQ is for the near-misses.

3. **Post-run analysis.** After a search completes, run PSLQ on all hall-of-fame entries
   to check for hidden relations.

The right place is probably a post-processing step, not in the hot evaluation loop.


## Go Implementation

There's a Go PSLQ library: `github.com/ncw/pslq`. It uses `math/big.Float`, which is
what we already use for constants. The API:

```go
import "github.com/ncw/pslq"

solver := pslq.New(512) // 512-bit precision
solver.SetMaxCoeff(new(big.Int).SetInt64(1000000))
solver.SetMaxSteps(10000)

// Build input vector: [candidate, 1, pi, e, ln2, gamma, zeta3, catalan]
input := []big.Float{candidateValue, one, pi, e, ln2, gamma, zeta3, catalan}

coeffs, err := solver.Run(input)
// coeffs[0]*candidate + coeffs[1]*1 + coeffs[2]*pi + ... = 0
```

If `err == nil`, we found a relation. Extract:

```go
// candidate = -(coeffs[1] + coeffs[2]*pi + ...) / coeffs[0]
```


## False Positive Detection

A PSLQ result isn't automatically real. Validate:

1. **Residual check.** Recompute the relation at higher precision (e.g., 1024 bits). If
   the residual stays tiny, it's likely real. If it grows, it's a precision artifact.

2. **Coefficient complexity.** Simple relations (small coefficients) are more likely real.
   If PSLQ returns coefficients like `[834729, -293847, 0, 0, 0, 0, 472918, 0]`, it's
   probably a coincidence. Something like `[1, 0, -1, 0, 0, 0, 0, 0]` (candidate = pi)
   is obviously real.

3. **RoI metric (from Ramanujan Machine).** Compute:
   ```
   information_bits = sum(log2(|coeff_i| + 1))
   matched_bits = -log2(residual)
   RoI = matched_bits / information_bits
   ```
   RoI > 22 strongly indicates a genuine relation.


## Integration Points

1. **New file: `pkg/series/pslq.go`** — wraps the ncw/pslq library, provides
   `IdentifyConstant(value *big.Float) (string, bool)` that returns a human-readable
   description like `"3*pi + 2*ln(2)"` if a relation is found.

2. **Post-run hook in engine.go** — after the search completes, run PSLQ on top hall-of-
   fame entries and annotate the report.

3. **Hall of fame enhancement** — add a `IdentifiedAs` field to `AttemptResult` for PSLQ
   results.

4. **Eval tool** — add a `--identify` flag that runs PSLQ on the evaluated partial sum.


## Scope

This is a post-processing tool, not a core engine change. It doesn't affect the search
algorithm, fitness function, or genetic operators. It just helps us understand what the
engine finds. Implementation is small — mostly wrapping the ncw/pslq library and wiring
it into reporting.
