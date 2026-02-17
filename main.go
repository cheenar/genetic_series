package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/wildfunctions/genetic_series/pkg/constants"
	"github.com/wildfunctions/genetic_series/pkg/engine"
	"github.com/wildfunctions/genetic_series/pkg/pool"
	"github.com/wildfunctions/genetic_series/pkg/strategy"
)

func main() {
	cfg := engine.DefaultConfig()
	outdir := "."

	flag.StringVar(&cfg.Target, "target", cfg.Target, "target constant ("+strings.Join(constants.Names(), ", ")+")")
	flag.UintVar(&cfg.Precision, "precision", cfg.Precision, "precision in bits")
	flag.StringVar(&cfg.Pool, "pool", cfg.Pool, "gene pool ("+strings.Join(pool.Names(), ", ")+")")
	flag.StringVar(&cfg.Strategy, "strategy", cfg.Strategy, "evolution strategy ("+strings.Join(strategy.Names(), ", ")+")")
	flag.IntVar(&cfg.Population, "population", cfg.Population, "population size")
	flag.IntVar(&cfg.Generations, "generations", cfg.Generations, "number of generations")
	flag.Int64Var(&cfg.MaxTerms, "maxterms", cfg.MaxTerms, "max terms to evaluate per series")
	flag.Int64Var(&cfg.Seed, "seed", cfg.Seed, "random seed (0 = random)")
	flag.StringVar(&cfg.Format, "format", cfg.Format, "output format (text, json)")
	flag.BoolVar(&cfg.Verbose, "verbose", cfg.Verbose, "verbose output per generation")
	flag.IntVar(&cfg.MaxDepth, "maxdepth", cfg.MaxDepth, "max tree depth")
	flag.IntVar(&cfg.Workers, "workers", cfg.Workers, "number of parallel workers")
	flag.IntVar(&cfg.StagnationLimit, "stagnation", cfg.StagnationLimit, "generations without improvement before restart")
	flag.Float64Var(&cfg.F64PromotionThreshold, "f64threshold", cfg.F64PromotionThreshold, "min float64 digits to promote to big.Float (0 = disabled)")
	flag.StringVar(&cfg.SeedFormula, "seed-formula", "", "LaTeX seed formula for constant-tuning strategy")
	flag.StringVar(&outdir, "outdir", outdir, "output directory for generated files")
	flag.Parse()

	// Create output directory and wire it into config so the engine can write during the run
	if err := os.MkdirAll(outdir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "error creating output dir: %v\n", err)
		os.Exit(1)
	}
	cfg.OutDir = outdir

	e, err := engine.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	report := e.Run()

	switch cfg.Format {
	case "json":
		if err := engine.WriteJSONFinal(os.Stdout, report); err != nil {
			fmt.Fprintf(os.Stderr, "error writing JSON: %v\n", err)
			os.Exit(1)
		}
	default:
		engine.WriteTextFinal(os.Stdout, report)
	}
}
