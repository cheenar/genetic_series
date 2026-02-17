package expr

import (
	"testing"
)

func TestParseExprLatexRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		node ExprNode
	}{
		// Leaves
		{"var", &VarNode{}},
		{"const", &ConstNode{Val: 42}},
		{"negative const", &ConstNode{Val: -7}},

		// All unary ops
		{"neg", &UnaryNode{Op: OpNeg, Child: &VarNode{}}},
		{"factorial", &UnaryNode{Op: OpFactorial, Child: &VarNode{}}},
		{"double factorial", &UnaryNode{Op: OpDoubleFactorial, Child: &VarNode{}}},
		{"altsign", &UnaryNode{Op: OpAltSign, Child: &VarNode{}}},
		{"fibonacci", &UnaryNode{Op: OpFibonacci, Child: &VarNode{}}},
		{"sin", &UnaryNode{Op: OpSin, Child: &VarNode{}}},
		{"cos", &UnaryNode{Op: OpCos, Child: &VarNode{}}},
		{"ln", &UnaryNode{Op: OpLn, Child: &VarNode{}}},
		{"floor", &UnaryNode{Op: OpFloor, Child: &VarNode{}}},
		{"ceil", &UnaryNode{Op: OpCeil, Child: &VarNode{}}},
		{"abs", &UnaryNode{Op: OpAbs, Child: &VarNode{}}},
		{"sqrt", &UnaryNode{Op: OpSqrt, Child: &VarNode{}}},

		// All binary ops
		{"add", &BinaryNode{Op: OpAdd, Left: &VarNode{}, Right: &ConstNode{Val: 3}}},
		{"sub", &BinaryNode{Op: OpSub, Left: &VarNode{}, Right: &ConstNode{Val: 3}}},
		{"mul", &BinaryNode{Op: OpMul, Left: &VarNode{}, Right: &ConstNode{Val: 3}}},
		{"div", &BinaryNode{Op: OpDiv, Left: &VarNode{}, Right: &ConstNode{Val: 3}}},
		{"pow", &BinaryNode{Op: OpPow, Left: &VarNode{}, Right: &ConstNode{Val: 3}}},
		{"binomial", &BinaryNode{Op: OpBinomial, Left: &VarNode{}, Right: &ConstNode{Val: 3}}},

		// Nested expressions (3+ levels)
		{"nested add-mul", &BinaryNode{
			Op: OpAdd,
			Left: &BinaryNode{
				Op:    OpMul,
				Left:  &ConstNode{Val: 20},
				Right: &VarNode{},
			},
			Right: &ConstNode{Val: 5},
		}},
		{"nested unary in binary", &BinaryNode{
			Op:    OpDiv,
			Left:  &UnaryNode{Op: OpFactorial, Child: &VarNode{}},
			Right: &BinaryNode{Op: OpPow, Left: &ConstNode{Val: 2}, Right: &VarNode{}},
		}},
		{"deep nesting", &UnaryNode{
			Op: OpNeg,
			Child: &BinaryNode{
				Op: OpAdd,
				Left: &UnaryNode{
					Op:    OpSqrt,
					Child: &VarNode{},
				},
				Right: &BinaryNode{
					Op:    OpMul,
					Left:  &ConstNode{Val: 3},
					Right: &ConstNode{Val: 7},
				},
			},
		}},

		// Factorial with complex child
		{"factorial of add", &UnaryNode{
			Op: OpFactorial,
			Child: &BinaryNode{
				Op:    OpAdd,
				Left:  &VarNode{},
				Right: &ConstNode{Val: 1},
			},
		}},

		// Power with complex exponent
		{"pow complex", &BinaryNode{
			Op:   OpPow,
			Left: &VarNode{},
			Right: &BinaryNode{
				Op:    OpAdd,
				Left:  &VarNode{},
				Right: &ConstNode{Val: 1},
			},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			latex := tt.node.LaTeX()
			parsed, err := ParseExprLatex(latex)
			if err != nil {
				t.Fatalf("ParseExprLatex(%q) error: %v", latex, err)
			}
			got := parsed.String()
			want := tt.node.String()
			if got != want {
				t.Errorf("round-trip failed:\n  LaTeX:  %q\n  got:    %s\n  want:   %s", latex, got, want)
			}
		})
	}
}

func TestParseExprLatexErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"malformed frac", `\frac{3`},
		{"missing brace", `{n`},
		{"unknown token", `@`},
		{"trailing junk", `n xyz`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseExprLatex(tt.input)
			if err == nil {
				t.Errorf("expected error for input %q, got nil", tt.input)
			}
		})
	}
}
