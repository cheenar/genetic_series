package expr

import (
	"math"
	"math/big"
	"testing"
)

const testPrec = 512

func bf(v float64) *big.Float {
	return new(big.Float).SetPrec(testPrec).SetFloat64(v)
}

func bfInt(v int64) *big.Float {
	return new(big.Float).SetPrec(testPrec).SetInt64(v)
}

func assertEval(t *testing.T, node ExprNode, n float64, expected float64, tol float64) {
	t.Helper()
	nf := bf(n)
	result, ok := node.Eval(nf, testPrec)
	if !ok {
		t.Fatalf("Eval returned ok=false for n=%v", n)
	}
	got, _ := result.Float64()
	if math.Abs(got-expected) > tol {
		t.Errorf("Eval(n=%v) = %v, want %v (tol=%v)", n, got, expected, tol)
	}
}

func TestVarNode(t *testing.T) {
	v := &VarNode{}
	assertEval(t, v, 5, 5, 0)
	assertEval(t, v, 0, 0, 0)

	if v.String() != "n" {
		t.Errorf("VarNode.String() = %q, want \"n\"", v.String())
	}
	if v.NodeCount() != 1 {
		t.Errorf("VarNode.NodeCount() = %d, want 1", v.NodeCount())
	}
}

func TestConstNode(t *testing.T) {
	c := &ConstNode{Val: 7}
	assertEval(t, c, 99, 7, 0)

	if c.String() != "7" {
		t.Errorf("ConstNode.String() = %q, want \"7\"", c.String())
	}
}

func TestFactorial(t *testing.T) {
	// 5! = 120
	node := &UnaryNode{Op: OpFactorial, Child: &ConstNode{Val: 5}}
	assertEval(t, node, 0, 120, 0)

	// 0! = 1
	node = &UnaryNode{Op: OpFactorial, Child: &ConstNode{Val: 0}}
	assertEval(t, node, 0, 1, 0)

	// Factorial of negative returns false
	node = &UnaryNode{Op: OpFactorial, Child: &ConstNode{Val: -1}}
	_, ok := node.Eval(bf(0), testPrec)
	if ok {
		t.Error("Factorial(-1) should return ok=false")
	}
}

func TestAltSign(t *testing.T) {
	node := &UnaryNode{Op: OpAltSign, Child: &VarNode{}}

	assertEval(t, node, 0, 1, 0)  // (-1)^0 = 1
	assertEval(t, node, 1, -1, 0) // (-1)^1 = -1
	assertEval(t, node, 2, 1, 0)  // (-1)^2 = 1
	assertEval(t, node, 3, -1, 0)
}

func TestBinaryOps(t *testing.T) {
	n := &VarNode{}
	two := &ConstNode{Val: 2}

	// n + 2
	add := &BinaryNode{Op: OpAdd, Left: n, Right: two}
	assertEval(t, add, 3, 5, 0)

	// n - 2
	sub := &BinaryNode{Op: OpSub, Left: n, Right: two}
	assertEval(t, sub, 5, 3, 0)

	// n * 2
	mul := &BinaryNode{Op: OpMul, Left: n, Right: two}
	assertEval(t, mul, 4, 8, 0)

	// n / 2
	div := &BinaryNode{Op: OpDiv, Left: n, Right: two}
	assertEval(t, div, 10, 5, 0)
}

func TestDivisionByZero(t *testing.T) {
	node := &BinaryNode{Op: OpDiv, Left: &ConstNode{Val: 1}, Right: &ConstNode{Val: 0}}
	_, ok := node.Eval(bf(0), testPrec)
	if ok {
		t.Error("Division by zero should return ok=false")
	}
}

func TestPow(t *testing.T) {
	// 2^3 = 8
	node := &BinaryNode{Op: OpPow, Left: &ConstNode{Val: 2}, Right: &ConstNode{Val: 3}}
	assertEval(t, node, 0, 8, 0)

	// 2^(-1) = 0.5
	node = &BinaryNode{Op: OpPow, Left: &ConstNode{Val: 2}, Right: &ConstNode{Val: -1}}
	assertEval(t, node, 0, 0.5, 1e-15)
}

func TestBinomial(t *testing.T) {
	// C(5, 2) = 10
	node := &BinaryNode{Op: OpBinomial, Left: &ConstNode{Val: 5}, Right: &ConstNode{Val: 2}}
	assertEval(t, node, 0, 10, 0)

	// C(10, 0) = 1
	node = &BinaryNode{Op: OpBinomial, Left: &ConstNode{Val: 10}, Right: &ConstNode{Val: 0}}
	assertEval(t, node, 0, 1, 0)
}

func TestFibonacci(t *testing.T) {
	cases := []struct {
		n    int64
		want float64
	}{
		{0, 0}, {1, 1}, {2, 1}, {3, 2}, {4, 3}, {5, 5}, {6, 8}, {10, 55},
	}
	for _, tc := range cases {
		node := &UnaryNode{Op: OpFibonacci, Child: &ConstNode{Val: tc.n}}
		assertEval(t, node, 0, tc.want, 0)
	}
}

func TestDoubleFactorial(t *testing.T) {
	// 5!! = 5*3*1 = 15
	node := &UnaryNode{Op: OpDoubleFactorial, Child: &ConstNode{Val: 5}}
	assertEval(t, node, 0, 15, 0)

	// 6!! = 6*4*2 = 48
	node = &UnaryNode{Op: OpDoubleFactorial, Child: &ConstNode{Val: 6}}
	assertEval(t, node, 0, 48, 0)
}

func TestClone(t *testing.T) {
	original := &BinaryNode{
		Op:   OpAdd,
		Left: &VarNode{},
		Right: &UnaryNode{
			Op:    OpFactorial,
			Child: &ConstNode{Val: 3},
		},
	}

	cloned := original.Clone()
	if cloned.String() != original.String() {
		t.Errorf("Clone mismatch: %q vs %q", cloned.String(), original.String())
	}

	// Modify clone, original should be unchanged
	cloned.(*BinaryNode).Right.(*UnaryNode).Child = &ConstNode{Val: 99}
	if original.String() == cloned.String() {
		t.Error("Clone is not a deep copy")
	}
}

func TestComplexity(t *testing.T) {
	leaf := &VarNode{}
	if leaf.NodeCount() != 1 {
		t.Errorf("VarNode.NodeCount() = %d, want 1", leaf.NodeCount())
	}

	tree := &BinaryNode{
		Op:   OpAdd,
		Left: &VarNode{},
		Right: &BinaryNode{
			Op:    OpMul,
			Left:  &ConstNode{Val: 2},
			Right: &VarNode{},
		},
	}
	if tree.NodeCount() != 5 {
		t.Errorf("tree.NodeCount() = %d, want 5", tree.NodeCount())
	}
	if tree.Depth() != 3 {
		t.Errorf("tree.Depth() = %d, want 3", tree.Depth())
	}
}

func TestString(t *testing.T) {
	// 1 / n!
	tree := &BinaryNode{
		Op:   OpDiv,
		Left: &ConstNode{Val: 1},
		Right: &UnaryNode{
			Op:    OpFactorial,
			Child: &VarNode{},
		},
	}
	s := tree.String()
	if s != "(1 / (n)!)" {
		t.Errorf("String() = %q", s)
	}
}

func TestLaTeX(t *testing.T) {
	tree := &BinaryNode{
		Op:   OpDiv,
		Left: &ConstNode{Val: 1},
		Right: &UnaryNode{
			Op:    OpFactorial,
			Child: &VarNode{},
		},
	}
	s := tree.LaTeX()
	if s != "\\frac{1}{{n}!}" {
		t.Errorf("LaTeX() = %q", s)
	}
}

// TestKnownSeries_EMinusOne: Sum_{n=0}^{inf} 1/n! = e
// We verify the partial sum of 1/n! for n=0..20 ≈ e with high precision.
func TestKnownSeries_EMinusOne(t *testing.T) {
	// 1/n!
	num := &ConstNode{Val: 1}
	den := &UnaryNode{Op: OpFactorial, Child: &VarNode{}}

	sum := new(big.Float).SetPrec(testPrec)
	for i := int64(0); i <= 20; i++ {
		n := bfInt(i)
		nv, ok := num.Eval(n, testPrec)
		if !ok {
			t.Fatalf("num eval failed at n=%d", i)
		}
		dv, ok := den.Eval(n, testPrec)
		if !ok {
			t.Fatalf("den eval failed at n=%d", i)
		}
		sum.Add(sum, new(big.Float).SetPrec(testPrec).Quo(nv, dv))
	}

	// Should be very close to e
	e, _ := new(big.Float).SetPrec(testPrec).SetString("2.71828182845904523536028747135266249775724709369995")
	diff := new(big.Float).Sub(sum, e)
	diff.Abs(diff)
	eps := new(big.Float).SetPrec(testPrec).SetFloat64(1e-18)
	if diff.Cmp(eps) > 0 {
		t.Errorf("partial sum of 1/n! = %s, want ≈ e", sum.Text('g', 30))
	}
}

// TestKnownSeries_PiOver4: Sum_{n=0}^{inf} (-1)^n / (2n+1) = π/4 (Leibniz)
func TestKnownSeries_PiOver4(t *testing.T) {
	// (-1)^n / (2n+1)
	num := &UnaryNode{Op: OpAltSign, Child: &VarNode{}}
	den := &BinaryNode{
		Op:    OpAdd,
		Left:  &BinaryNode{Op: OpMul, Left: &ConstNode{Val: 2}, Right: &VarNode{}},
		Right: &ConstNode{Val: 1},
	}

	sum := new(big.Float).SetPrec(testPrec)
	// This converges slowly, use many terms
	for i := int64(0); i <= 100000; i++ {
		n := bfInt(i)
		nv, ok := num.Eval(n, testPrec)
		if !ok {
			t.Fatalf("num eval failed at n=%d", i)
		}
		dv, ok := den.Eval(n, testPrec)
		if !ok {
			t.Fatalf("den eval failed at n=%d", i)
		}
		sum.Add(sum, new(big.Float).SetPrec(testPrec).Quo(nv, dv))
	}

	piOver4, _ := new(big.Float).SetPrec(testPrec).SetString("0.7853981633974483")
	diff := new(big.Float).Sub(sum, piOver4)
	diff.Abs(diff)
	eps := new(big.Float).SetPrec(testPrec).SetFloat64(1e-5)
	if diff.Cmp(eps) > 0 {
		t.Errorf("Leibniz series partial sum = %s, want ≈ π/4 = 0.7854", sum.Text('g', 20))
	}
}

func TestSimplify(t *testing.T) {
	tests := []struct {
		name string
		node ExprNode
		want string
	}{
		{
			"x + 0 = x",
			&BinaryNode{Op: OpAdd, Left: &VarNode{}, Right: &ConstNode{Val: 0}},
			"n",
		},
		{
			"x * 1 = x",
			&BinaryNode{Op: OpMul, Left: &VarNode{}, Right: &ConstNode{Val: 1}},
			"n",
		},
		{
			"x * 0 = 0",
			&BinaryNode{Op: OpMul, Left: &VarNode{}, Right: &ConstNode{Val: 0}},
			"0",
		},
		{
			"const fold 2+3",
			&BinaryNode{Op: OpAdd, Left: &ConstNode{Val: 2}, Right: &ConstNode{Val: 3}},
			"5",
		},
		{
			"double negation",
			&UnaryNode{Op: OpNeg, Child: &UnaryNode{Op: OpNeg, Child: &VarNode{}}},
			"n",
		},
		{
			"x^1 = x",
			&BinaryNode{Op: OpPow, Left: &VarNode{}, Right: &ConstNode{Val: 1}},
			"n",
		},
		{
			"x^0 = 1",
			&BinaryNode{Op: OpPow, Left: &VarNode{}, Right: &ConstNode{Val: 0}},
			"1",
		},
		{
			"x/1 = x",
			&BinaryNode{Op: OpDiv, Left: &VarNode{}, Right: &ConstNode{Val: 1}},
			"n",
		},
		{
			"x - (-5) = x + 5",
			&BinaryNode{Op: OpSub, Left: &VarNode{}, Right: &ConstNode{Val: -5}},
			"(n + 5)",
		},
		{
			"x + (-3) = x - 3",
			&BinaryNode{Op: OpAdd, Left: &VarNode{}, Right: &ConstNode{Val: -3}},
			"(n - 3)",
		},
		{
			"4! = 24",
			&UnaryNode{Op: OpFactorial, Child: &ConstNode{Val: 4}},
			"24",
		},
		{
			"5!! = 15",
			&UnaryNode{Op: OpDoubleFactorial, Child: &ConstNode{Val: 5}},
			"15",
		},
		{
			"MinInt64 add no crash",
			&BinaryNode{Op: OpAdd, Left: &VarNode{}, Right: &ConstNode{Val: math.MinInt64}},
			"(n + -9223372036854775808)",
		},
		{
			"MinInt64 sub no crash",
			&BinaryNode{Op: OpSub, Left: &VarNode{}, Right: &ConstNode{Val: math.MinInt64}},
			"(n - -9223372036854775808)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Simplify(tc.node)
			if got.String() != tc.want {
				t.Errorf("Simplify(%s) = %s, want %s", tc.node.String(), got.String(), tc.want)
			}
		})
	}
}

func TestFloorCeil(t *testing.T) {
	// floor(3.7) = 3
	node := &UnaryNode{Op: OpFloor, Child: &BinaryNode{
		Op: OpDiv, Left: &ConstNode{Val: 37}, Right: &ConstNode{Val: 10},
	}}
	assertEval(t, node, 0, 3, 0)

	// ceil(3.2) = 4
	node = &UnaryNode{Op: OpCeil, Child: &BinaryNode{
		Op: OpDiv, Left: &ConstNode{Val: 32}, Right: &ConstNode{Val: 10},
	}}
	assertEval(t, node, 0, 4, 0)
}
