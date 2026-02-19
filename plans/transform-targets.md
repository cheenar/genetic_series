# Transform Targets

## The Idea

Don't just search for pi. Simultaneously search for `1/pi`, `pi^2`, `pi^2/6`, `sqrt(pi)`,
`ln(pi)`, and other transforms. Many known identities target a transform of the constant,
not the constant itself:

- Ramanujan's famous series computes **1/pi**, not pi.
- The Basel problem gives **pi^2/6**, not pi.
- Stirling's approximation involves **sqrt(2*pi)**.
- Wallis's product gives **pi/2**.

If we only search for pi, we miss all of these. The engine might evolve a perfect series
for pi^2/6 and score it as "5.2 digits of pi" — close enough to survive a few generations
but not good enough to thrive. It would get outcompeted and die, even though it's a
genuine identity.


## The Fix

For each target constant, precompute a set of transforms and score each candidate against
all of them. The candidate's fitness is the **best score across all transforms**.

```
transforms(pi) = [pi, 1/pi, pi^2, pi^2/6, sqrt(pi), 2*pi, pi/2, pi/4, ln(pi)]
transforms(e)  = [e, 1/e, e^2, sqrt(e), ln(e)=1, 2*e, e/2]
transforms(gamma) = [gamma, 1/gamma, gamma^2, 2*gamma, pi*gamma]
...
```


## Which Transforms

### Universal transforms (apply to any constant C)

| Transform | Formula | Why |
|-----------|---------|-----|
| Reciprocal | 1/C | Ramanujan-type series compute 1/pi |
| Square | C^2 | Basel problem, zeta values |
| Square root | sqrt(C) | Stirling, Gaussian integrals |
| Double | 2*C | Wallis-type products |
| Half | C/2 | Common normalizations |
| Quarter | C/4 | Leibniz gives pi/4, not pi |
| Negation | -C | Series with flipped sign conventions |
| Natural log | ln(C) | Logarithmic identities |

### Constant-specific transforms

| Constant | Extra transforms | Why |
|----------|-----------------|-----|
| pi | pi^2/6, pi^2/8, pi^2/12, pi^4/90 | Zeta values: zeta(2)=pi^2/6, zeta(4)=pi^4/90 |
| pi | pi/4, pi/2, 3*pi/4 | Leibniz, Wallis, Gregory |
| pi | pi*sqrt(2), pi/sqrt(2) | Appears in elliptic integrals |
| e | e^pi, e^(-pi) | Appears in modular forms |
| ln(2) | ln(2)/2, 2*ln(2) | Common normalizations |
| zeta(3) | zeta(3)/2, 7*zeta(3)/8 | Known identity relations |


## Implementation

### Step 1: TransformSet type

```go
// pkg/series/transforms.go

type Transform struct {
    Name  string      // "pi^2/6", "1/pi", etc.
    Value *big.Float  // precomputed value at working precision
}

// TransformsFor returns the standard transforms for a target constant.
func TransformsFor(name string, value *big.Float, prec uint) []Transform {
    transforms := []Transform{
        {Name: name, Value: value},  // identity transform
    }

    // Universal transforms
    one := new(big.Float).SetPrec(prec).SetInt64(1)
    reciprocal := new(big.Float).SetPrec(prec).Quo(one, value)
    transforms = append(transforms, Transform{"1/" + name, reciprocal})

    squared := new(big.Float).SetPrec(prec).Mul(value, value)
    transforms = append(transforms, Transform{name + "^2", squared})

    // ... etc for each universal transform ...

    // Constant-specific transforms
    if name == "pi" {
        six := new(big.Float).SetPrec(prec).SetInt64(6)
        piSqOver6 := new(big.Float).SetPrec(prec).Quo(squared, six)
        transforms = append(transforms, Transform{"pi^2/6", piSqOver6})
        // ... etc ...
    }

    return transforms
}
```

### Step 2: Multi-target fitness scoring

```go
func ComputeFitnessMultiTarget(c *Candidate, result EvalResult,
    transforms []Transform, weights FitnessWeights) (Fitness, string) {

    bestFitness := WorstFitness
    bestName := ""

    for _, t := range transforms {
        f := ComputeFitness(c, result, t.Value, weights)
        if f.Combined > bestFitness.Combined {
            bestFitness = f
            bestName = t.Name
        }
    }

    return bestFitness, bestName
}
```

The `bestName` tells us which transform matched. If a candidate scores 15 digits of
`pi^2/6`, we know it's computing zeta(2).

### Step 3: Report which transform matched

Add `MatchedTransform string` to `AttemptResult` and `GenerationReport`. The hall of
fame should show not just the fitness but what the series is converging to:

```
#1: [attempt 5, gen 42] 15.3 digits of pi^2/6 | sum_{n=1}^{inf} 1/n^2
```


## Cost

Each candidate is evaluated once (producing one partial sum). Then we compare that
partial sum against each transform value. The comparison is just a subtraction and a
`countCorrectDigits` call — essentially free compared to the series evaluation.

With ~15 transforms per constant, the overhead is negligible. The evaluation dominates;
the multi-target comparison is O(n_transforms) big.Float subtractions.


## Interaction With PSLQ

Transform targets and PSLQ overlap in purpose but complement each other:

- **Transform targets** catch known, expected transforms (pi^2/6, 1/pi, etc.). They're
  fast and specific.
- **PSLQ** catches unexpected linear combinations (3*pi + 2*ln(2)). It's slower but
  more general.

Run transform targets during the search (cheap, helps fitness selection). Run PSLQ
after the search on the best candidates (more expensive, catches what transforms miss).


## Scope

This is a small, self-contained change:
1. New file `pkg/series/transforms.go` (~100 lines)
2. Modify `ComputeFitness` call site in engine.go to use multi-target version
3. Add `MatchedTransform` to reporting structs
4. Precompute transforms once at engine startup

No changes to genetic operators, candidate representation, or evaluation. This is
probably the cheapest of the three plans in terms of implementation effort, and it
could easily double our discovery surface.
