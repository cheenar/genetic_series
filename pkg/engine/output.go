package engine

import (
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/wildfunctions/genetic_series/pkg/series"
)

// GenerationReport summarizes one generation.
type GenerationReport struct {
	Generation    int            `json:"generation"`
	BestFitness   series.Fitness `json:"best_fitness"`
	BestCandidate string         `json:"best_candidate"`
	BestLaTeX     string         `json:"best_latex,omitempty"`
	AvgFitness    float64        `json:"avg_fitness"`
	BestPartialSum string        `json:"best_partial_sum,omitempty"`
}

// AttemptResult summarizes one restart attempt.
type AttemptResult struct {
	Attempt        int            `json:"attempt"`
	Generations    int            `json:"generations"`
	BestFoundAtGen int            `json:"best_found_at_gen"`
	BestCandidate  string         `json:"best_candidate"`
	BestLaTeX      string         `json:"best_latex"`
	BestFitness    series.Fitness `json:"best_fitness"`
	BestPartialSum string         `json:"best_partial_sum"`
	Timestamp      time.Time      `json:"timestamp"`
}

// FinalReport summarizes the entire run.
type FinalReport struct {
	Config        Config             `json:"config"`
	Generations   []GenerationReport `json:"generations,omitempty"`
	BestCandidate string             `json:"best_candidate"`
	BestLaTeX     string             `json:"best_latex"`
	BestFitness   series.Fitness     `json:"best_fitness"`
	BestPartialSum string            `json:"best_partial_sum"`
	Attempts      []AttemptResult    `json:"attempts,omitempty"`
}

// WriteTextReport writes a generation report in human-readable format.
func WriteTextReport(w io.Writer, r GenerationReport) {
	fmt.Fprintf(w, "Gen %4d | Best: %.4f (%.1f digits) | Avg: %.4f | %s\n",
		r.Generation, r.BestFitness.Combined, r.BestFitness.CorrectDigits,
		r.AvgFitness, r.BestCandidate)
}

// WriteAttemptSummary writes a single attempt result.
func WriteAttemptSummary(w io.Writer, a AttemptResult) {
	fmt.Fprintf(w, "Attempt %d: %d generations, %.1f digits | %s\n",
		a.Attempt, a.Generations, a.BestFitness.CorrectDigits, a.BestCandidate)
}

const maxHallOfFame = 100

// sortByDigits returns a copy of attempts sorted by CorrectDigits descending.
func sortByDigits(attempts []AttemptResult) []AttemptResult {
	sorted := make([]AttemptResult, len(attempts))
	copy(sorted, attempts)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].BestFitness.CorrectDigits != sorted[j].BestFitness.CorrectDigits {
			return sorted[i].BestFitness.CorrectDigits > sorted[j].BestFitness.CorrectDigits
		}
		return sorted[i].BestFitness.Combined > sorted[j].BestFitness.Combined
	})
	return sorted
}

// dedupAttempts removes duplicate candidates, keeping only the first (best) entry
// for each unique BestCandidate string. Input must already be sorted by quality descending.
func dedupAttempts(sorted []AttemptResult) []AttemptResult {
	seen := map[string]bool{}
	result := make([]AttemptResult, 0, len(sorted))
	for _, a := range sorted {
		if seen[a.BestCandidate] {
			continue
		}
		seen[a.BestCandidate] = true
		result = append(result, a)
	}
	return result
}

// WriteHallOfFame writes the sorted hall of fame across attempts.
func WriteHallOfFame(w io.Writer, attempts []AttemptResult) {
	sorted := sortByDigits(attempts)
	sorted = dedupAttempts(sorted)
	if len(sorted) > maxHallOfFame {
		sorted = sorted[:maxHallOfFame]
	}
	fmt.Fprintln(w, "\n--- Hall of Fame ---")
	for i, a := range sorted {
		fmt.Fprintf(w, "  #%d: [attempt %d, gen %d] %5.1f digits | %s\n",
			i+1, a.Attempt, a.BestFoundAtGen, a.BestFitness.CorrectDigits, a.BestCandidate)
	}
}

// WriteTextFinal writes the final report in human-readable format.
func WriteTextFinal(w io.Writer, r FinalReport) {
	if len(r.Attempts) > 0 {
		WriteHallOfFame(w, r.Attempts)
	}
	fmt.Fprintln(w, "\n========== FINAL RESULT ==========")
	fmt.Fprintf(w, "Target:    %s\n", r.Config.Target)
	fmt.Fprintf(w, "Strategy:  %s\n", r.Config.Strategy)
	fmt.Fprintf(w, "Pool:      %s\n", r.Config.Pool)
	fmt.Fprintf(w, "Best:      %s\n", r.BestCandidate)
	fmt.Fprintf(w, "LaTeX:     %s\n", r.BestLaTeX)
	fmt.Fprintf(w, "Fitness:   %.4f\n", r.BestFitness.Combined)
	fmt.Fprintf(w, "Digits:    %.1f\n", r.BestFitness.CorrectDigits)
	fmt.Fprintf(w, "Partial:   %s\n", r.BestPartialSum)
	fmt.Fprintln(w, "==================================")
}

// WriteJSONFinal writes the final report as JSON.
func WriteJSONFinal(w io.Writer, r FinalReport) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// latexEscape escapes underscores and other special chars for LaTeX text mode.
func latexEscape(s string) string {
	return strings.ReplaceAll(s, "_", `\_`)
}

// WriteHallOfFameLatex writes a compilable LaTeX document of the hall of fame.
func WriteHallOfFameLatex(w io.Writer, attempts []AttemptResult, cfg Config, targetValue *big.Float) {
	sorted := sortByDigits(attempts)
	sorted = dedupAttempts(sorted)
	if len(sorted) > maxHallOfFame {
		sorted = sorted[:maxHallOfFame]
	}

	targetStr := targetValue.Text('g', 50)

	genBudget := "unlimited"
	if cfg.Generations > 0 {
		genBudget = fmt.Sprintf("%d", cfg.Generations)
	}

	fmt.Fprintln(w, `\documentclass{article}`)
	fmt.Fprintln(w, `\usepackage{amsmath}`)
	fmt.Fprintln(w, `\usepackage{geometry}`)
	fmt.Fprintln(w, `\geometry{margin=1in}`)
	fmt.Fprintf(w, "\\title{Hall of Fame --- Target: \\texttt{%s}}\n", latexEscape(cfg.Target))
	fmt.Fprintln(w, `\date{\today}`)
	fmt.Fprintln(w, `\begin{document}`)
	fmt.Fprintln(w, `\maketitle`)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "\\noindent Target: \\texttt{%s}, Pool: \\texttt{%s}, Strategy: \\texttt{%s}\\\\\n",
		latexEscape(cfg.Target), latexEscape(cfg.Pool), latexEscape(cfg.Strategy))
	fmt.Fprintf(w, "Population: %d, Gen budget: %s, Stagnation: %d, Workers: %d, Seed: %d\\\\\n",
		cfg.Population, genBudget, cfg.StagnationLimit, cfg.Workers, cfg.Seed)
	fmt.Fprintf(w, "Target value: \\verb|%s|\\ldots\n\n", targetStr)

	for i, a := range sorted {
		fmt.Fprintf(w, "\\subsection*{\\#%d --- %.1f digits (attempt %d, gen %d, %s)}\n",
			i+1, a.BestFitness.CorrectDigits, a.Attempt, a.BestFoundAtGen,
			a.Timestamp.Format("2006-01-02 15:04:05 UTC"))
		fmt.Fprintln(w, `\[`)
		fmt.Fprintf(w, "  %s\n", a.BestLaTeX)
		fmt.Fprintln(w, `\]`)
		if a.BestPartialSum != "" {
			// Compute error = |partial_sum - target|
			partialSum, _, err := big.ParseFloat(a.BestPartialSum, 10, targetValue.Prec(), big.ToNearestEven)
			if err == nil {
				diff := new(big.Float).Sub(partialSum, targetValue)
				diff.Abs(diff)
				fmt.Fprintf(w, "\\noindent Partial sum: \\verb|%s|\\\\\n", a.BestPartialSum)
				fmt.Fprintf(w, "Error: \\verb|%s|\n\n", diff.Text('e', 10))
			} else {
				fmt.Fprintf(w, "\\noindent Partial sum: \\verb|%s|\n\n", a.BestPartialSum)
			}
		}
	}

	fmt.Fprintln(w, `\end{document}`)
}
