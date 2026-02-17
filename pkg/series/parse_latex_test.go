package series

import (
	"testing"

	"github.com/wildfunctions/genetic_series/pkg/expr"
)

func TestParseCandidateLatexRoundTrip(t *testing.T) {
	tests := []struct {
		name      string
		candidate *Candidate
	}{
		{"simple", &Candidate{
			Numerator:   &expr.ConstNode{Val: 1},
			Denominator: &expr.UnaryNode{Op: expr.OpFactorial, Child: &expr.VarNode{}},
			Start:       0,
		}},
		{"start 1", &Candidate{
			Numerator:   &expr.UnaryNode{Op: expr.OpAltSign, Child: &expr.VarNode{}},
			Denominator: &expr.BinaryNode{Op: expr.OpMul, Left: &expr.ConstNode{Val: 2}, Right: &expr.VarNode{}},
			Start:       1,
		}},
		{"complex formula", &Candidate{
			Numerator: &expr.BinaryNode{
				Op:   expr.OpMul,
				Left: &expr.ConstNode{Val: 3},
				Right: &expr.BinaryNode{
					Op: expr.OpSub,
					Left: &expr.BinaryNode{
						Op:    expr.OpMul,
						Left:  &expr.ConstNode{Val: 20},
						Right: &expr.VarNode{},
					},
					Right: &expr.BinaryNode{
						Op: expr.OpAdd,
						Left: &expr.BinaryNode{
							Op:   expr.OpMul,
							Left: &expr.BinaryNode{Op: expr.OpMul, Left: &expr.VarNode{}, Right: &expr.VarNode{}},
							Right: &expr.UnaryNode{
								Op:    expr.OpFactorial,
								Child: &expr.VarNode{},
							},
						},
						Right: &expr.BinaryNode{
							Op: expr.OpSub,
							Left: &expr.BinaryNode{
								Op:    expr.OpMul,
								Left:  &expr.ConstNode{Val: 22},
								Right: &expr.VarNode{},
							},
							Right: &expr.VarNode{},
						},
					},
				},
			},
			Denominator: &expr.ConstNode{Val: 4},
			Start:       0,
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			latex := tt.candidate.LaTeX()
			parsed, err := ParseCandidateLatex(latex)
			if err != nil {
				t.Fatalf("ParseCandidateLatex(%q) error: %v", latex, err)
			}
			got := parsed.String()
			want := tt.candidate.String()
			if got != want {
				t.Errorf("round-trip failed:\n  LaTeX:  %q\n  got:    %s\n  want:   %s", latex, got, want)
			}
		})
	}
}

func TestParseCandidateLatexErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"missing sum", `\frac{1}{2}`},
		{"bad start", `\sum_{n=abc}^{\infty} n`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseCandidateLatex(tt.input)
			if err == nil {
				t.Errorf("expected error for input %q, got nil", tt.input)
			}
		})
	}
}
