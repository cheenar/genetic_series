package expr

import (
	"math"
	"math/big"
)

// maxRecurseDepth caps recursion in simplification to prevent stack overflow
// on pathologically deep trees produced by crossover.
const maxRecurseDepth = 100

// Simplify applies rewrite rules to reduce an expression tree.
// It repeatedly applies rules until no further changes occur.
func Simplify(node ExprNode) ExprNode {
	for i := 0; i < 20; i++ { // cap iterations
		next := simplifyD(node, 0)
		if next.String() == node.String() {
			return next
		}
		node = next
	}
	return node
}

func simplifyD(node ExprNode, depth int) ExprNode {
	if depth > maxRecurseDepth {
		return node
	}

	switch n := node.(type) {
	case *VarNode, *ConstNode:
		return node

	case *UnaryNode:
		child := simplifyD(n.Child, depth+1)

		// Double negation: -(-x) = x
		if n.Op == OpNeg {
			if inner, ok := child.(*UnaryNode); ok && inner.Op == OpNeg {
				return inner.Child
			}
		}

		// Neg of const: -(k) = -k (guard: -MinInt64 overflows)
		if n.Op == OpNeg {
			if c, ok := child.(*ConstNode); ok && (c.Val > math.MinInt64) {
				return &ConstNode{Val: -c.Val}
			}
		}

		// Factorial of small constants: fold entirely
		if n.Op == OpFactorial {
			if c, ok := child.(*ConstNode); ok && c.Val >= 0 && c.Val <= 20 {
				result := int64(1)
				for i := int64(2); i <= c.Val; i++ {
					result *= i
				}
				return &ConstNode{Val: result}
			}
		}

		// DoubleFactorial of small constants
		if n.Op == OpDoubleFactorial {
			if c, ok := child.(*ConstNode); ok && c.Val >= 0 && c.Val <= 20 {
				result := int64(1)
				for i := c.Val; i >= 2; i -= 2 {
					result *= i
				}
				return &ConstNode{Val: result}
			}
		}

		// AltSign constant folding
		if n.Op == OpAltSign {
			if c, ok := child.(*ConstNode); ok && c.Val >= 0 {
				if c.Val%2 == 0 {
					return &ConstNode{Val: 1}
				}
				return &ConstNode{Val: -1}
			}
		}

		// Abs of const
		if n.Op == OpAbs {
			if c, ok := child.(*ConstNode); ok {
				if c.Val < 0 {
					return &ConstNode{Val: -c.Val}
				}
				return c
			}
		}

		// Sqrt of perfect square constant: sqrt(k²) = k
		if n.Op == OpSqrt {
			if c, ok := child.(*ConstNode); ok && c.Val >= 0 {
				root := int64(math.Sqrt(float64(c.Val)))
				if root*root == c.Val {
					return &ConstNode{Val: root}
				}
			}
		}

		return &UnaryNode{Op: n.Op, Child: child}

	case *BinaryNode:
		left := simplifyD(n.Left, depth+1)
		right := simplifyD(n.Right, depth+1)

		lc, lok := left.(*ConstNode)
		rc, rok := right.(*ConstNode)

		// Constant folding for basic ops
		if lok && rok {
			if result, ok := foldConstants(n.Op, lc.Val, rc.Val); ok {
				return &ConstNode{Val: result}
			}
		}

		switch n.Op {
		case OpAdd:
			// x + 0 = x
			if rok && rc.Val == 0 {
				return left
			}
			// 0 + x = x
			if lok && lc.Val == 0 {
				return right
			}
			// x + (-k) = x - k (guard: -MinInt64 overflows back to negative)
			if rok && rc.Val < 0 && -rc.Val > 0 {
				return simplifyD(&BinaryNode{Op: OpSub, Left: left, Right: &ConstNode{Val: -rc.Val}}, depth+1)
			}
			// x + neg(y) = x - y
			if ru, ok := right.(*UnaryNode); ok && ru.Op == OpNeg {
				return simplifyD(&BinaryNode{Op: OpSub, Left: left, Right: ru.Child}, depth+1)
			}

		case OpSub:
			// x - 0 = x
			if rok && rc.Val == 0 {
				return left
			}
			// 0 - x = -x
			if lok && lc.Val == 0 {
				return simplifyD(&UnaryNode{Op: OpNeg, Child: right}, depth+1)
			}
			// x - (-k) = x + k (guard: -MinInt64 overflows back to negative)
			if rok && rc.Val < 0 && -rc.Val > 0 {
				return simplifyD(&BinaryNode{Op: OpAdd, Left: left, Right: &ConstNode{Val: -rc.Val}}, depth+1)
			}
			// x - neg(y) = x + y
			if ru, ok := right.(*UnaryNode); ok && ru.Op == OpNeg {
				return simplifyD(&BinaryNode{Op: OpAdd, Left: left, Right: ru.Child}, depth+1)
			}
			// x - x = 0 (structural equality)
			if left.String() == right.String() {
				return &ConstNode{Val: 0}
			}

		case OpMul:
			// x * 0 = 0
			if rok && rc.Val == 0 {
				return &ConstNode{Val: 0}
			}
			if lok && lc.Val == 0 {
				return &ConstNode{Val: 0}
			}
			// x * 1 = x
			if rok && rc.Val == 1 {
				return left
			}
			// 1 * x = x
			if lok && lc.Val == 1 {
				return right
			}
			// x * (-1) = -x
			if rok && rc.Val == -1 {
				return simplifyD(&UnaryNode{Op: OpNeg, Child: left}, depth+1)
			}
			// (-1) * x = -x
			if lok && lc.Val == -1 {
				return simplifyD(&UnaryNode{Op: OpNeg, Child: right}, depth+1)
			}

		case OpDiv:
			// x / 1 = x
			if rok && rc.Val == 1 {
				return left
			}
			// 0 / x = 0
			if lok && lc.Val == 0 {
				return &ConstNode{Val: 0}
			}
			// x / x = 1 (structural equality, non-zero)
			if left.String() == right.String() {
				return &ConstNode{Val: 1}
			}

		case OpPow:
			// x^0 = 1
			if rok && rc.Val == 0 {
				return &ConstNode{Val: 1}
			}
			// x^1 = x
			if rok && rc.Val == 1 {
				return left
			}
			// 0^x = 0 (for positive x)
			if lok && lc.Val == 0 {
				return &ConstNode{Val: 0}
			}
			// 1^x = 1
			if lok && lc.Val == 1 {
				return &ConstNode{Val: 1}
			}
		}

		return &BinaryNode{Op: n.Op, Left: left, Right: right}

	default:
		return node
	}
}

func foldConstants(op BinaryOp, a, b int64) (int64, bool) {
	switch op {
	case OpAdd:
		return a + b, true
	case OpSub:
		return a - b, true
	case OpMul:
		// Check for overflow
		if a != 0 && b != 0 {
			result := a * b
			if result/a != b {
				return 0, false
			}
			return result, true
		}
		return 0, true
	case OpDiv:
		if b == 0 {
			return 0, false
		}
		if a%b != 0 {
			return 0, false // don't fold if not exact
		}
		return a / b, true
	case OpPow:
		if b < 0 {
			return 0, false
		}
		if b > 20 {
			return 0, false
		}
		result := int64(1)
		base := a
		for i := int64(0); i < b; i++ {
			result *= base
		}
		return result, true
	default:
		return 0, false
	}
}

// SimplifyBigFloat evaluates constant subtrees and replaces them with ConstNodes.
// This recursively finds subtrees with no VarNode and evaluates them.
func SimplifyBigFloat(node ExprNode, prec uint) ExprNode {
	node = Simplify(node)
	node = foldConstantSubtrees(node, prec)
	node = Simplify(node) // second pass to clean up after folding
	return node
}

func foldConstantSubtrees(node ExprNode, prec uint) ExprNode {
	return foldConstantSubtreesD(node, prec, 0)
}

func foldConstantSubtreesD(node ExprNode, prec uint, depth int) ExprNode {
	if depth > maxRecurseDepth {
		return node
	}

	if !containsVar(node) {
		dummyN := new(big.Float).SetPrec(prec).SetInt64(0)
		if val, ok := node.Eval(dummyN, prec); ok {
			if iv, ok := toInt64Approx(val); ok {
				return &ConstNode{Val: iv}
			}
			// Non-integer constant subtree (e.g. 1/(-13) + 9 ≈ 8.923):
			// round to nearest integer so the GA can work with a clean constant.
			// TODO: support rational constants (e.g. RatNode{Num, Den}) so we
			// can fold 1/3 + 1 to 4/3 instead of rounding to 1.
			if iv, ok := roundToInt64(val); ok {
				return &ConstNode{Val: iv}
			}
		}
		return node
	}

	switch n := node.(type) {
	case *UnaryNode:
		return &UnaryNode{Op: n.Op, Child: foldConstantSubtreesD(n.Child, prec, depth+1)}
	case *BinaryNode:
		return &BinaryNode{Op: n.Op,
			Left:  foldConstantSubtreesD(n.Left, prec, depth+1),
			Right: foldConstantSubtreesD(n.Right, prec, depth+1),
		}
	default:
		return node
	}
}

// ContainsVar reports whether the expression tree contains the variable n.
func ContainsVar(node ExprNode) bool {
	return containsVar(node)
}

func containsVar(node ExprNode) bool {
	return containsVarD(node, 0)
}

func containsVarD(node ExprNode, depth int) bool {
	if depth > maxRecurseDepth {
		return true // assume variable present to be safe (won't fold)
	}
	switch n := node.(type) {
	case *VarNode:
		return true
	case *ConstNode:
		return false
	case *UnaryNode:
		return containsVarD(n.Child, depth+1)
	case *BinaryNode:
		return containsVarD(n.Left, depth+1) || containsVarD(n.Right, depth+1)
	default:
		return false
	}
}

func toInt64Approx(f *big.Float) (int64, bool) {
	if !f.IsInt() {
		return 0, false
	}
	i, acc := f.Int64()
	return i, acc == big.Exact
}

// roundToInt64 rounds a big.Float to the nearest int64.
// Returns false for NaN, Inf, or values outside int64 range.
func roundToInt64(f *big.Float) (int64, bool) {
	if f.IsInf() {
		return 0, false
	}
	// Round to nearest integer
	rounded := new(big.Float).Copy(f)
	if f.Sign() >= 0 {
		rounded.Add(rounded, new(big.Float).SetFloat64(0.5))
	} else {
		rounded.Sub(rounded, new(big.Float).SetFloat64(0.5))
	}
	i, acc := rounded.Int64()
	if acc != big.Exact && acc != big.Below && acc != big.Above {
		return 0, false
	}
	// Reject zero — avoid introducing zero constants that could cause div-by-zero
	if i == 0 {
		return 0, false
	}
	// Reject if overflow (Int64 returns MinInt64/MaxInt64 for out-of-range)
	if i == math.MinInt64 || i == math.MaxInt64 {
		return 0, false
	}
	return i, true
}
