# Pi Series Consttune Experiments


## Experiment 1: Engine-discovered 11.8-digit formula

### Formula

```
Sum_{n=0}^{inf} ((n)! * ((2 * n))! * (26)) / ((((3 * n))! * (2)^(n)))
```

In LaTeX: `\sum_{n=0}^{\infty} \frac{26 \cdot n! \cdot (2n)!}{(3n)! \cdot 2^n}`

- **Target:** pi
- **Accuracy:** 11.8 digits
- **Terms needed:** ~15

### Consttune results

The constant 26 was perturbed across all integer values. Results showed smooth degradation:

| Constant | Digits |
|----------|--------|
| 26       | 11.8   |
| 25       | 10.2   |
| 27       | 10.2   |
| 24       | 9.3    |
| 28       | 9.3    |
| 23       | 8.7    |
| 29       | 8.7    |

Every step away from 26 lost ~1 digit, with symmetric falloff in both directions. No other constant in the structure could be tuned to recover accuracy.


## Experiment 2: 20.2-digit pi formula

### Formula

```
2 \sum_{k=0}^{\infty} \frac{k! \, (2k)! \, (25k - 3)}{(3k)! \, 2^{k}}
```

In engine notation:
```
Sum_{n=0}^{inf} (2 * n! * (2n)! * (25n - 3)) / ((3n)! * 2^n)
```

- **Target:** pi
- **Accuracy:** 20.2 digits (from 21 terms)
- **Convergence rate:** ~1 digit per term

### Consttune results (126 attempts)

The formula was seeded into consttune with kitchensink pool (population 1000, stagnation 500). Over 126 independent restart attempts, each running hundreds of generations:

**Hall of Fame top 10:**

| Rank | Digits | Formula |
|------|--------|---------|
| 1    | 20.2   | `2 * n! * (2n)! * (25n - 3) / ((3n)! * 2^n)` |
| 2    | 20.2   | (same formula, reordered multiplication) |
| 3    | 4.6    | `(99n + 3) * (n!)^2 / (6n)!` |
| 4    | 4.6    | (same as #3, negation variant) |
| 5    | 4.5    | `(14n + 3) * (n!)^2 / (5n)!` |
| 6    | 4.5    | (same as #5, negation variant) |
| 7    | 4.3    | `(-107n + 4) * (n!)^2 / (5n)!` |
| 8    | 3.4    | `(98n + 3) * (n!)^2 / (6n)!` |
| 9    | 3.3    | `(100n + 3) * (n!)^2 / (6n)!` |
| 10   | 3.1    | `(97n + 3) * (n!)^2 / (6n)!` |

There is a 15.6-digit gap between the original formula (20.2) and the best perturbation (4.6). In 126 attempts, not a single perturbation came close to the original.

### Secondary structure

The consttune runs revealed a secondary family of formulas with the simplified structure `(an + b) * (n!)^2 / (cn)!`:

- `(99n + 3) * (n!)^2 / (6n)!` at 4.6 digits
- `(14n + 3) * (n!)^2 / (5n)!` at 4.5 digits

These lose the `(2n)!` and `2^n` terms from the original, collapsing to a simpler form.


## Experiment 3: Modified Ramanujan series for 1/pi

### Formula

The user took the known Ramanujan series for 1/pi:

```
\frac{\sqrt{8}}{9801} \sum_{n=0}^{\infty} \frac{(4n)!}{(n!)^4} \frac{1103 + 26390n}{396^{4n}}
```

and deliberately introduced two small errors -- changing `(n!)^4` to `(n!)^5` and `26390` to `26391` -- as a blind test of consttune's diagnostic ability.

Modified (input) formula:
```
\frac{\sqrt{8}}{9801} \sum_{n=0}^{\infty} \frac{(4n)!}{(n!)^5} \frac{1103 + 26391n}{396^{4n}}
```

- **Target:** one_over_pi
- **Accuracy of input:** 12.1 digits (from 6 terms)

### Consttune results (~40 attempts)

Consttune immediately improved on the input by finding 26390 instead of 26391. The Hall of Fame showed:

| Rank | Digits | Key constants |
|------|--------|---------------|
| 1    | 16.5   | 26390, (n!)^5 |
| 2    | 16.2   | 26390, (n!)^4 |
| 3    | 16.1   | 26390, (n!)^6 |
| 4    | 16.0   | 26390, (n!)^7 |
| 5    | 15.9   | 26390, (n!)^8 |
| 6    | 15.9   | 26390, (n!)^9 |
| 7    | 15.9   | 26390, (n!)^10 |
| 8    | 15.6   | 26390, (n!)^3 |
| 9    | 15.2   | 26390, (n!)^2 |
| 10   | 14.8   | 26390, (n!)^1 |
| 11   | 14.2   | 26390, (n!)^-1 |
| 12+  | 12.1   | 26389 or 26391, any (n!)^k |

All formulas with the constant 1103 and base 396 held constant. The linear coefficient 26390 was the dominant factor. The exponent on `(n!)^k` barely mattered -- values from -1 to 10 all gave 14-16.5 digits.

This is a **broad plateau**, not a cliff. The true Ramanujan formula `(n!)^4, 26390` sits at 16.2 digits, surrounded by many nearby variants at comparable accuracy.


## Disproved hypothesis: cliff vs smooth as a diagnostic

### The hypothesis (Claude's)

After Experiments 1 and 2, I (Claude) proposed a framework for using consttune to distinguish "real" mathematical identities from numerical coincidences:

- **Smooth degradation** under perturbation (Experiment 1) = likely coincidence
- **Sharp cliff** under perturbation (Experiment 2) = likely genuine structure

The reasoning was: if a formula truly equals a constant, perturbing any integer should break the identity entirely, producing a catastrophic drop in accuracy. If it merely approximates a constant by luck, nearby integers should give nearby outputs and degrade gently.

### The disproof (user's)

Experiment 3 refutes this hypothesis. The Ramanujan series for 1/pi is a proven mathematical identity -- one of the most famous in mathematics. Yet under consttune it shows **smooth degradation**, not a cliff. Changing `(n!)^4` to `(n!)^5` or `(n!)^{10}` barely affects accuracy. The consttune profile looks indistinguishable from what the hypothesis would classify as "coincidence."

The user provided a clean explanation for why the hypothesis fails. Whether perturbation causes a large or small change depends on **how much weight each constant carries in the formula**, not on whether the formula is structurally valid:

- Consider `1 + 2 = 3`. Change either constant and you get a completely different result. But that's just because the entire output depends heavily on both constants.
- Consider `1/10 + 1/10 + 1/10 + ... (ten times) = 1`. Change one of the 10s to 11, and you get 0.991 -- still close to 1. It "degrades smoothly," but the underlying identity is just as valid.

In the Ramanujan formula, the `396^{4n}` term dominates convergence so overwhelmingly that the `(n!)^k` factor is nearly irrelevant for the first several terms. Changing the exponent on `n!` barely shifts the partial sum. This has nothing to do with whether the formula is a real identity -- it's just a property of the relative magnitudes of the terms.

### Conclusion

The consttune cliff/smooth pattern reflects the **sensitivity structure** of a formula -- which constants carry the most weight -- not whether the formula is mathematically genuine. It cannot be used as a diagnostic for distinguishing identities from coincidences.

What consttune IS good for: exploring the constant landscape around a formula, finding improved variants, and discovering related formula families. It found 26390 as an improvement over 26391, and it mapped out the secondary `(an+b)(n!)^2/(cn)!` family in Experiment 2. These are useful results even without the interpretive framework.
