# Hypergeometric Template Search

## The Idea

Almost every known fast-converging series for a mathematical constant is hypergeometric.
A series is hypergeometric when the ratio of consecutive terms is a rational function of n:

```
a_{n+1} / a_n = P(n) / Q(n)
```

where P and Q are polynomials in n with integer coefficients.

Instead of evolving arbitrary expression trees (huge search space, mostly garbage),
dedicate a subpopulation to searching over the integer parameters of hypergeometric
templates. This is a much smaller, more structured search space — and it's where the
gold is.


## What Hypergeometric Series Look Like

### The general form

```
sum_{n=0}^{inf} C * (a1*n+b1)(a2*n+b2)...(ap*n+bp) / ((c1*n+d1)(c2*n+d2)...(cq*n+dq)) * x^n / n!
```

Or equivalently, using Pochhammer symbols (rising factorials):

```
pFq(a1,...,ap; c1,...,cq; x) = sum_{n=0}^{inf} (a1)_n...(ap)_n / ((c1)_n...(cq)_n) * x^n / n!
```

where `(a)_n = a(a+1)(a+2)...(a+n-1)` is the rising factorial.

### Famous examples

**Ramanujan's 1/pi series:**
```
1/pi = (2*sqrt(2)/9801) * sum_{n=0}^{inf} (4n)! * (1103 + 26390*n) / ((n!)^4 * 396^{4n})
```

The term ratio here is a rational function of n times a geometric factor (1/396^4).

**Chudnovsky's formula (fastest known series for pi):**
```
1/pi = 12 * sum_{n=0}^{inf} (-1)^n * (6n)! * (13591409 + 545140134*n) /
       ((3n)! * (n!)^3 * 640320^{3n+3/2})
```

Same structure: factorials in the numerator, factorials and a geometric base in the
denominator, with a linear-in-n numerator factor.

**Apery's series for zeta(3):**
```
zeta(3) = 5/2 * sum_{n=1}^{inf} (-1)^{n-1} / (n^3 * C(2n, n))
```

**Series for Catalan's constant:**
```
G = pi/8 * ln(2 + sqrt(3)) + 3/8 * sum_{n=0}^{inf} 1/((2n+1)^2 * C(2n, n))
```

### The pattern

All of these have the form:

```
constant = K * sum_{n=0}^{inf} (linear_in_n) * (product_of_factorials) /
           (product_of_factorials) * (geometric_base)^n
```

The free parameters are:
- K: a scaling constant (often involving sqrt, pi, powers of small integers)
- The linear numerator factor: `alpha + beta*n`
- Which factorials appear: `(n!)`, `(2n)!`, `(3n)!`, `(4n)!`, `(6n)!`
- The geometric base: `x^n` where x is a rational number or reciprocal of a power
- Signs: `(-1)^n` or not


## The Template

Rather than evolving arbitrary trees, search over this parameterized template:

```
sum_{n=0}^{inf} (A + B*n) * (C1*n)!^e1 * (C2*n)!^e2 * ... / ((D1*n)!^f1 * (D2*n)!^f2 * ...) * R^n
```

Parameters to search over (all integers):
- `A, B`: linear numerator coefficients
- `C1, C2, ..., D1, D2, ...`: factorial multipliers (typically 1, 2, 3, 4, 6)
- `e1, e2, ..., f1, f2, ...`: factorial exponents (typically 1, 2, 3, 4)
- `R`: geometric ratio (as a fraction p/q, or as `1/base^k`)

### Concrete search space

Keep it small to start. Fix the template to:

```
sum_{n=0}^{inf} (A + B*n) * (a*n)!^p * (b*n)!^q / ((c*n)!^r * (d*n)!^s) * (1/R)^n
```

Where:
- `A` in `{-10000, ..., 10000}` (or search via consttune)
- `B` in `{-100000, ..., 100000}`
- `a, b, c, d` in `{1, 2, 3, 4, 6}`
- `p, q, r, s` in `{0, 1, 2, 3, 4}`
- `R` in `{k^m : k in {2,...,1000}, m in {1,...,6}}`

This is still a large space, but it's vastly smaller than arbitrary expression trees.
And every point in this space is a well-formed hypergeometric series (no garbage).


## How to Search

### Option A: Genetic search over parameters (fits our architecture)

Represent a hypergeometric candidate as a vector of integers:

```go
type HyperCandidate struct {
    A, B           int64    // linear numerator: A + B*n
    NumFactors     []Factor // numerator factorials: (a*n)!^p
    DenFactors     []Factor // denominator factorials: (c*n)!^r
    BaseNum, BaseDen int64  // geometric ratio: (BaseNum/BaseDen)^n
}

type Factor struct {
    Mult int // factorial multiplier: (Mult*n)!
    Exp  int // exponent
}
```

Use the same genetic operators but adapted for integer vectors:
- **Mutation:** bump A or B by +/-1, change a factorial multiplier, change an exponent,
  change the base.
- **Crossover:** swap numerator factors from one parent, denominator from another.

Fitness: evaluate the series the same way we evaluate any candidate (partial sums at
checkpoints, compare to target).

### Option B: Exhaustive enumeration (like the Ramanujan Machine)

If the parameter space is small enough, enumerate all combinations and evaluate each one.
The Ramanujan Machine used meet-in-the-middle to speed this up, but for small parameter
ranges, brute force works.

Estimate: with 5 choices for each of 4 factorial multipliers, 5 choices for each of 4
exponents, and ~1000 geometric bases, that's `5^4 * 5^4 * 1000 ≈ 400M` combinations.
Too many for brute force, but amenable to genetic search or MITM.

### Recommendation: Option A

It fits our existing architecture. We already have genetic search infrastructure. We
just need a new candidate representation that maps to the hypergeometric template instead
of to expression trees.


## Implementation Plan

### Step 1: HyperCandidate struct

```go
// pkg/series/hyper.go

type HyperCandidate struct {
    A, B       int64
    NumFact    []HyperFactor  // numerator: product of (m*n)!^e
    DenFact    []HyperFactor  // denominator: product of (m*n)!^e
    GeoBase    *big.Rat       // geometric ratio per term
    AltSign    bool           // multiply by (-1)^n
}

type HyperFactor struct {
    Mult int  // (Mult * n)!
    Exp  int  // raised to this power
}
```

Add an `Evaluate` method that computes the partial sum directly (no expression tree —
just arithmetic on factorials and the geometric base). This is much faster than tree
evaluation because there's no tree traversal overhead.

### Step 2: Integration with the engine

Two options:

**Option A: Separate population.** Run a second population of HyperCandidates alongside
the main expression-tree population. They compete for the same fitness slots but evolve
independently.

**Option B: Encode as expression trees.** Convert the HyperCandidate template into an
ExprNode tree and use the existing evaluation pipeline. This is simpler architecturally
but loses the speed advantage.

Recommendation: **Option A**. The evaluation is fast enough that a separate population
is cheap, and it keeps the code clean.

### Step 3: Genetic operators for HyperCandidate

- **Random init:** random A, B in small range; 1-3 random factorial factors in num/den;
  random geometric base from a pool of common values (powers of small primes).
- **Mutate:** perturb one parameter (A+/-delta, change a factorial mult, etc.)
- **Crossover:** swap NumFact from parent 1, DenFact from parent 2, etc.

### Step 4: Scaling constant search

The linear numerator `A + B*n` handles part of the scaling, but many identities also
have an outer constant `K * sum(...)`. Options:
- Add K as a parameter and search over it.
- Use consttune to optimize K after finding a good series structure.
- Search for `1/constant` and `constant^2` etc. (see transform-targets plan).


## Relation to Existing Plans

This complements the expression-tree search rather than replacing it:
- **Expression trees** are good for discovering truly novel structures that nobody has
  thought of. Unbiased exploration.
- **Hypergeometric templates** are good for finding series within a known productive
  family. Biased but efficient.

Running both in parallel maximizes coverage. The hypergeometric search handles the
"low-hanging fruit" while the genetic tree search explores the unknown.

The PSLQ plan helps here too: if the hypergeometric search finds a series that converges
to *something* with high accuracy, PSLQ identifies what that something is.
