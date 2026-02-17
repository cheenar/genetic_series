# Captain's Log: Feasibility Check

**Stardate:** 2026-02-16
**Mood:** Cautiously optimistic
**Confidence:** 6/10

## Context

Several sessions in. Built the core engine, LaTeX parser, eval tool, and consttune
strategy. Ran experiments on pi and 1/pi formulas. Haven't discovered anything new yet.
User asked the hard question: is this project feasible, or are we wasting our time?

## What we've learned so far

1. **Consttune is useful but not diagnostic.** We initially hypothesized that sharp
   cliffs under constant perturbation indicate real identities while smooth degradation
   indicates coincidences. The user disproved this by feeding a slightly modified
   Ramanujan series (a proven identity) that showed smooth degradation. The cliff/smooth
   pattern just reflects how much weight each constant carries, not mathematical validity.

2. **The engine had evaluation blind spots.** The `sqrt` function was capped at float64
   precision (~15 digits), and `intPow` stopped at exponent 20. The correct Ramanujan
   formula for 1/pi was being evaluated at 16.2 digits instead of 116+. Fixed now.

3. **Slow-converging series are invisible.** The Leibniz formula for pi (simplest series
   in existence) gives only 3.2 digits at 512 terms. The engine can't distinguish it
   from noise. Richardson extrapolation is planned to fix this.

4. **The search space is enormous.** With all our operators, most randomly generated
   trees are garbage. The engine is searching for needles in a haystack.

## The case for continuing

- **Prior art works.** The Ramanujan Machine project (Technion, 2019, published in
  Nature) used computational search to discover genuinely new continued fraction
  representations of constants. The premise is proven.

- **We're building the right tools.** Even if the genetic search hasn't hit gold yet,
  the eval tool, consttune, and LaTeX parser make it easy to investigate and verify
  candidates. These are the picks and shovels.

- **Unbiased search is the point.** For well-known constants like pi, we could cheat
  by constraining the search to known hypergeometric families. But for lesser-known
  constants (Catalan's, Apery's, Euler-Mascheroni), nobody knows what structures work.
  An unbiased genetic search is arguably the right tool precisely because it doesn't
  assume structure.

- **We haven't really tried yet.** Most of our time has been building infrastructure.
  We haven't done long runs on unexplored constants. The engine hasn't had a real
  chance to search.

## The case for concern

- **The Ramanujan Machine used structured search (MITM), not random genetic search.**
  Their search space was much more constrained (continued fractions with integer
  parameters). Our search space is vastly larger.

- **No discoveries yet.** Everything we've found so far is either a known formula or
  a numerical coincidence.

- **Compute budget is unknown.** We don't know how many CPU hours it takes to find
  something real. Could be days, could be years.

## Plan going forward

1. Implement Richardson extrapolation (unlock slow-converging series)
2. Do long runs targeting lesser-known constants (Catalan's, Apery's)
3. Save all candidates above 8 digits -- look for structural patterns even if nothing
   hits 20+
4. Keep improving the fitness function and evaluation precision
5. Don't over-constrain the search space. The whole point is unbiased exploration.

## Gut feeling

This is a moonshot. The odds of discovering a genuinely new mathematical identity are
not high on any given run. But the tools we're building are sound, the approach has
precedent, and the constants we're targeting have room for discovery. If we keep
improving the engine and give it enough compute, something interesting will come out.

The worst case is that we build a great tool for verifying and exploring known series,
which has value on its own.
