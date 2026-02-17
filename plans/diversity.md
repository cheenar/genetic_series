# Diversity Ideas: Richer Candidate Structures

## Current Shape

Every candidate is a single series:

```
S = sum_{n=start}^{inf} Numerator(n) / Denominator(n)
```

Numerator and Denominator are expression trees built from `{n, constants, +, -, *, /, ^, !, sqrt, trig, ...}`. This is expressive but structurally limited — many famous constants require sums of fractions, products, nested forms, or continued fractions that can't be reached by a single `Num/Den`.


## Idea 1: Sum of Two Series (Additive Composition)

Allow a candidate to be the **sum of two independent series**:

```
S = sum Num1(n)/Den1(n) + sum Num2(n)/Den2(n)
```

**Why it helps:** Many constants decompose into a "main series" plus a "correction series". For example, fast convergence formulas for pi often combine two Ramanujan-like sums. The search can discover each piece independently.

**Implementation sketch:**
- Add an optional second `(Numerator2, Denominator2, Start2)` triple to `Candidate`, or a `[]SeriesTerm` slice.
- Evaluate each sub-series independently, then sum partial sums.
- Crossover can swap entire sub-series between candidates.
- Mutation can: add/remove a sub-series, mutate one sub-series, or swap which sub-series is "primary".
- Fitness: score the combined sum.
- Start simple: at most 2 sub-series. Could generalize to N later.


## Idea 2: Outer Transform (Post-Processing the Sum)

Wrap the entire series result in an **outer function**:

```
S = f( sum Num(n)/Den(n) )
```

where `f` could be `sqrt`, `exp`, `ln`, `1/x`, `x^k`, etc.

**Why it helps:** Many constants are roots or logs of convergent series. For example, `pi = sqrt(6 * sum 1/n^2)` (Basel). The current engine can't discover this because the sqrt lives *outside* the sum.

**Implementation sketch:**
- Add an optional `OuterTransform ExprNode` to `Candidate`. When nil, no transform.
- After computing partial sum `S`, apply `OuterTransform.Eval(S)`.
- Mutation can: add/remove/change the outer transform.
- Pool provides a small set of outer transforms: `sqrt`, `cbrt`, `exp`, `ln`, `1/x`, `x^2`, `x^3`, `x * k` for small k.
- Keep it to depth-1 transforms to avoid bloat (e.g., `sqrt(6*x)` not `sqrt(ln(exp(x)))`).


## Idea 3: Product Series (Multiplicative Form)

Some constants are naturally expressed as **infinite products**:

```
P = prod_{n=start}^{inf} Expr(n)
```

For example, Wallis' product for pi/2: `prod (4n^2)/(4n^2 - 1)`.

**Why it helps:** Opens up an entirely different class of representations. Many classical formulas are products, not sums.

**Implementation sketch:**
- Add a `Mode` field to Candidate: `"sum"` (default) or `"product"`.
- In product mode, accumulate `result *= Num(n)/Den(n)` instead of `result += Num(n)/Den(n)`.
- Convergence detection: check that partial products stabilize (ratio of consecutive partial products approaches 1).
- Mutation can flip between sum and product mode (rare mutation, maybe 2% chance).
- Most of the expression tree machinery stays the same.


## Idea 4: Continued Fractions

Many constants have beautiful continued fraction representations:

```
a0 + b1 / (a1 + b2 / (a2 + b3 / (a3 + ...)))
```

**Why it helps:** Continued fractions often converge faster than series. Pi, e, sqrt(2), golden ratio all have elegant continued fraction forms.

**Implementation sketch:**
- New candidate type with `A(n)` and `B(n)` expression trees for the CF coefficients.
- Evaluate using the Lentz/Thompson algorithm (numerically stable).
- Could be a separate `Mode` or a separate candidate struct.
- This is the most invasive change but potentially the highest payoff for discovering new representations.


## Idea 5: Raising the Series to a Power

```
S = ( sum Num(n)/Den(n) ) ^ k
```

This is a special case of Idea 2 but worth calling out because `pi^2 = 6 * sum 1/n^2` (Basel problem) is the textbook example.

**Implementation sketch:**
- Could be folded into the OuterTransform idea (Idea 2) with `f(x) = x^k`.
- Or add a dedicated `Power int` field to Candidate (simpler, less general).
- The search would evolve `k` as a small integer (-3..3) or rational.


## Idea 6: Multiplicative Scaling Constant

```
S = k * sum Num(n)/Den(n)
```

**Why it helps:** Many formulas have a leading constant factor (e.g., `pi/4 = sum (-1)^n/(2n+1)`, so `pi = 4 * sum ...`). Currently the engine can bake `4` into the numerator, but that forces the tree to carry extra weight.

**Implementation sketch:**
- Add a `Scale *big.Float` (or small rational `p/q`) to Candidate.
- Multiply partial sum by scale before comparing to target.
- Mutate scale: perturb, invert, double, halve, set to small integers.
- This is the lightest-weight change and could be done first as a quick win.


## Idea 7: Alternating Series Flag

Instead of requiring `(-1)^n` inside the expression tree (costs 2 nodes), add a boolean flag:

```
S = sum (-1)^n * Num(n)/Den(n)    (when alternating=true)
```

**Why it helps:** Saves 2 nodes of complexity budget for every alternating series. The engine currently wastes tree capacity on `(-1)^n` nodes.

**Implementation sketch:**
- Add `Alternating bool` to Candidate.
- During evaluation, multiply each term by `(-1)^(n-start)`.
- Mutation can flip this flag (maybe 5% chance).
- Reduces complexity score for alternating series, letting the engine explore deeper numerator/denominator structures.


## Suggested Priority Order

| Priority | Idea | Effort | Expected Impact |
|----------|------|--------|-----------------|
| 1 | Idea 6: Scale constant | Small | Medium — quick win, unlocks `k * sum` forms |
| 2 | Idea 7: Alternating flag | Small | Medium — frees up node budget |
| 3 | Idea 2: Outer transform | Medium | High — unlocks Basel-type results |
| 4 | Idea 3: Product series | Medium | High — entirely new class of formulas |
| 5 | Idea 1: Sum of two series | Medium | Medium — compositional discovery |
| 6 | Idea 5: Power of series | Small | Medium — subset of Idea 2 |
| 7 | Idea 4: Continued fractions | Large | Very High — but significant new code |


## Compatibility Notes

- All ideas are **additive** — they extend `Candidate` without breaking existing functionality.
- Default values (no scale, no outer transform, sum mode, not alternating) reproduce current behavior.
- The float64 fast path would need matching extensions for each new feature.
- LaTeX rendering needs updates for each new form (e.g., rendering products with `\prod`, continued fractions, outer transforms).
- JSON serialization of `FinalReport` would need new fields.
