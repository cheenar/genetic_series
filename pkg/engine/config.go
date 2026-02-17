package engine

import (
	"runtime"

	"github.com/wildfunctions/genetic_series/pkg/constants"
	"github.com/wildfunctions/genetic_series/pkg/series"
)

// Config holds all parameters for an evolutionary run.
type Config struct {
	Target      string
	Pool        string
	Strategy    string
	Population  int
	Generations int
	MaxTerms    int64
	MaxDepth    int
	Precision   uint
	Seed        int64
	Format      string // "text" or "json"
	Verbose     bool
	Workers     int
	Weights               series.FitnessWeights
	StagnationLimit       int
	OutDir                string
	F64PromotionThreshold float64 // min float64 digits to promote to big.Float (0 = disabled)
	SeedFormula           string  // LaTeX formula for constant-tuning (empty = normal init)
}

// DefaultConfig returns a config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Target:      "e",
		Pool:        "conservative",
		Strategy:    "hillclimb",
		Population:  200,
		Generations: 1000,
		MaxTerms:    1024,
		MaxDepth:    4,
		Precision:   constants.DefaultPrecision,
		Seed:        0, // 0 = random
		Format:      "text",
		Verbose:     false,
		Workers:     runtime.NumCPU(),
		Weights:               series.DefaultWeights(),
		StagnationLimit:       200,
		F64PromotionThreshold: 4.0,
	}
}
