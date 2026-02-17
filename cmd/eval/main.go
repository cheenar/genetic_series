package main

import (
	"flag"
	"fmt"
	"math"
	"math/big"
	"os"
	"strings"

	"github.com/wildfunctions/genetic_series/pkg/constants"
	"github.com/wildfunctions/genetic_series/pkg/series"
)

func main() {
	var (
		formula  string
		file     string
		target   string
		targetV  string
		maxTerms int64
		prec     uint
	)

	flag.StringVar(&formula, "formula", "", "LaTeX formula to evaluate")
	flag.StringVar(&file, "file", "", "file containing LaTeX formula")
	flag.StringVar(&target, "target", "", "named constant to compare ("+strings.Join(constants.Names(), ", ")+")")
	flag.StringVar(&targetV, "target-value", "", "explicit target value (decimal string)")
	flag.Int64Var(&maxTerms, "maxterms", 4096, "max terms to sum")
	flag.UintVar(&prec, "precision", 512, "precision in bits")
	flag.Parse()

	// Read formula from flag or file.
	if formula == "" && file != "" {
		data, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading %s: %v\n", file, err)
			os.Exit(1)
		}
		formula = strings.TrimSpace(string(data))
	}
	if formula == "" {
		fmt.Fprintln(os.Stderr, "usage: eval -formula '\\sum ...' [-target pi] [-maxterms 4096] [-precision 512]")
		fmt.Fprintln(os.Stderr, "       eval -file formula.txt [-target-value 3.14159...]")
		os.Exit(1)
	}

	// Parse the formula.
	cand, err := series.ParseCandidateLatex(formula)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Parsed: %s\n", cand.String())
	fmt.Fprintf(os.Stderr, "Evaluating up to %d terms at %d-bit precision...\n", maxTerms, prec)

	// Evaluate.
	result := series.EvaluateCandidate(cand, maxTerms, prec)
	if !result.OK {
		fmt.Fprintln(os.Stderr, "evaluation failed (not enough terms or timeout)")
		os.Exit(1)
	}

	fmt.Printf("Terms computed: %d\n", result.TermsComputed)
	fmt.Printf("Converged:     %v\n", result.Converged)
	fmt.Printf("Partial sum:   %s\n", result.PartialSum.Text('g', 50))

	// Compare against target if provided.
	var tv *big.Float
	if target != "" {
		c := constants.Get(target)
		if c == nil {
			fmt.Fprintf(os.Stderr, "unknown target: %s\n", target)
			os.Exit(1)
		}
		tv = c.Value
		fmt.Printf("Target (%s):   %s\n", target, tv.Text('g', 50))
	} else if targetV != "" {
		var ok bool
		tv, ok = new(big.Float).SetPrec(prec).SetString(targetV)
		if !ok {
			fmt.Fprintf(os.Stderr, "invalid target value: %s\n", targetV)
			os.Exit(1)
		}
		fmt.Printf("Target:        %s\n", tv.Text('g', 50))
	}

	if tv != nil {
		diff := new(big.Float).SetPrec(prec).Sub(result.PartialSum, tv)
		diff.Abs(diff)
		fmt.Printf("Error:         %s\n", diff.Text('e', 15))

		absTgt := new(big.Float).Abs(tv)
		if absTgt.Sign() > 0 {
			relErr := new(big.Float).SetPrec(prec).Quo(diff, absTgt)
			re, _ := relErr.Float64()
			if re > 0 {
				fmt.Printf("Correct digits: %.1f\n", -math.Log10(re))
			} else {
				fmt.Printf("Correct digits: 50+ (exact at this precision)\n")
			}
		}
	}
}
