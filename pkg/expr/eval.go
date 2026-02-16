package expr

import (
	"math"
	"math/big"
	"sync"
)

var (
	bigZero = big.NewFloat(0)
	bigOne  = big.NewFloat(1)
)

func (v *VarNode) Eval(n *big.Float, prec uint) (*big.Float, bool) {
	return new(big.Float).SetPrec(prec).Copy(n), true
}

func (c *ConstNode) Eval(n *big.Float, prec uint) (*big.Float, bool) {
	return new(big.Float).SetPrec(prec).SetInt64(c.Val), true
}

func (u *UnaryNode) Eval(n *big.Float, prec uint) (*big.Float, bool) {
	child, ok := u.Child.Eval(n, prec)
	if !ok {
		return nil, false
	}

	switch u.Op {
	case OpNeg:
		return new(big.Float).SetPrec(prec).Neg(child), true

	case OpFactorial:
		return bigFactorial(child, prec)

	case OpAltSign:
		// (-1)^child — child must be a non-negative integer
		iv, ok := toInt64(child)
		if !ok || iv < 0 {
			return nil, false
		}
		if iv%2 == 0 {
			return new(big.Float).SetPrec(prec).SetInt64(1), true
		}
		return new(big.Float).SetPrec(prec).SetInt64(-1), true

	case OpDoubleFactorial:
		return bigDoubleFactorial(child, prec)

	case OpFibonacci:
		return bigFibonacci(child, prec)

	case OpSin:
		f, _ := child.Float64()
		if math.IsInf(f, 0) || math.IsNaN(f) {
			return nil, false
		}
		return new(big.Float).SetPrec(prec).SetFloat64(math.Sin(f)), true

	case OpCos:
		f, _ := child.Float64()
		if math.IsInf(f, 0) || math.IsNaN(f) {
			return nil, false
		}
		return new(big.Float).SetPrec(prec).SetFloat64(math.Cos(f)), true

	case OpLn:
		f, _ := child.Float64()
		if f <= 0 || math.IsInf(f, 0) || math.IsNaN(f) {
			return nil, false
		}
		return new(big.Float).SetPrec(prec).SetFloat64(math.Log(f)), true

	case OpFloor:
		return bigFloor(child, prec), true

	case OpCeil:
		return bigCeil(child, prec), true

	case OpAbs:
		return new(big.Float).SetPrec(prec).Abs(child), true

	case OpSqrt:
		if child.Sign() < 0 {
			return nil, false
		}
		f, _ := child.Float64()
		if math.IsInf(f, 0) || math.IsNaN(f) {
			return nil, false
		}
		return new(big.Float).SetPrec(prec).SetFloat64(math.Sqrt(f)), true

	default:
		return nil, false
	}
}

func (b *BinaryNode) Eval(n *big.Float, prec uint) (*big.Float, bool) {
	left, ok := b.Left.Eval(n, prec)
	if !ok {
		return nil, false
	}
	right, ok := b.Right.Eval(n, prec)
	if !ok {
		return nil, false
	}

	switch b.Op {
	case OpAdd:
		return new(big.Float).SetPrec(prec).Add(left, right), true

	case OpSub:
		return new(big.Float).SetPrec(prec).Sub(left, right), true

	case OpMul:
		return new(big.Float).SetPrec(prec).Mul(left, right), true

	case OpDiv:
		if right.Cmp(bigZero) == 0 {
			return nil, false
		}
		return new(big.Float).SetPrec(prec).Quo(left, right), true

	case OpPow:
		return bigPow(left, right, prec)

	case OpBinomial:
		return bigBinomial(left, right, prec)

	default:
		return nil, false
	}
}

// toInt64 converts a big.Float to int64 if it represents a whole number.
func toInt64(f *big.Float) (int64, bool) {
	if !f.IsInt() {
		return 0, false
	}
	i, acc := f.Int64()
	if acc != big.Exact {
		return 0, false
	}
	return i, true
}

const maxComputeInput = 1000

// Memoized lookup tables that grow on demand.
var (
	factorialCache = &mathCache{}
	dblFactCache   = &mathCache{}
	fibonacciCache = &mathCache{}
)

type mathCache struct {
	mu     sync.RWMutex
	values []*big.Int
}

func (c *mathCache) get(n int64) (*big.Int, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if n < int64(len(c.values)) {
		return c.values[n], true
	}
	return nil, false
}

func init() {
	// Seed factorial: 0!=1, 1!=1, ..., 20!=2432902008176640000
	facts := make([]*big.Int, 21)
	facts[0] = big.NewInt(1)
	for i := int64(1); i <= 20; i++ {
		facts[i] = new(big.Int).Mul(facts[i-1], big.NewInt(i))
	}
	factorialCache.values = facts

	// Seed double factorial
	dfacts := make([]*big.Int, 21)
	for i := int64(0); i <= 20; i++ {
		result := big.NewInt(1)
		for j := i; j >= 2; j -= 2 {
			result.Mul(result, big.NewInt(j))
		}
		dfacts[i] = result
	}
	dblFactCache.values = dfacts

	// Seed fibonacci
	fibs := make([]*big.Int, 21)
	fibs[0] = big.NewInt(0)
	fibs[1] = big.NewInt(1)
	for i := int64(2); i <= 20; i++ {
		fibs[i] = new(big.Int).Add(fibs[i-1], fibs[i-2])
	}
	fibonacciCache.values = fibs
}

func bigFactorial(f *big.Float, prec uint) (*big.Float, bool) {
	iv, ok := toInt64(f)
	if !ok || iv < 0 || iv > maxComputeInput {
		return nil, false
	}
	if v, ok := factorialCache.get(iv); ok {
		return new(big.Float).SetPrec(prec).SetInt(v), true
	}
	// Extend cache up to iv
	factorialCache.mu.Lock()
	// Re-check after acquiring write lock
	if iv < int64(len(factorialCache.values)) {
		v := factorialCache.values[iv]
		factorialCache.mu.Unlock()
		return new(big.Float).SetPrec(prec).SetInt(v), true
	}
	cur := int64(len(factorialCache.values))
	for i := cur; i <= iv; i++ {
		next := new(big.Int).Mul(factorialCache.values[i-1], big.NewInt(i))
		factorialCache.values = append(factorialCache.values, next)
	}
	v := factorialCache.values[iv]
	factorialCache.mu.Unlock()
	return new(big.Float).SetPrec(prec).SetInt(v), true
}

func bigDoubleFactorial(f *big.Float, prec uint) (*big.Float, bool) {
	iv, ok := toInt64(f)
	if !ok || iv < 0 || iv > maxComputeInput {
		return nil, false
	}
	if v, ok := dblFactCache.get(iv); ok {
		return new(big.Float).SetPrec(prec).SetInt(v), true
	}
	dblFactCache.mu.Lock()
	if iv < int64(len(dblFactCache.values)) {
		v := dblFactCache.values[iv]
		dblFactCache.mu.Unlock()
		return new(big.Float).SetPrec(prec).SetInt(v), true
	}
	cur := int64(len(dblFactCache.values))
	for i := cur; i <= iv; i++ {
		var next *big.Int
		if i < 2 {
			next = big.NewInt(1)
		} else {
			next = new(big.Int).Mul(dblFactCache.values[i-2], big.NewInt(i))
		}
		dblFactCache.values = append(dblFactCache.values, next)
	}
	v := dblFactCache.values[iv]
	dblFactCache.mu.Unlock()
	return new(big.Float).SetPrec(prec).SetInt(v), true
}

func bigFibonacci(f *big.Float, prec uint) (*big.Float, bool) {
	iv, ok := toInt64(f)
	if !ok || iv < 0 || iv > maxComputeInput {
		return nil, false
	}
	if v, ok := fibonacciCache.get(iv); ok {
		return new(big.Float).SetPrec(prec).SetInt(v), true
	}
	fibonacciCache.mu.Lock()
	if iv < int64(len(fibonacciCache.values)) {
		v := fibonacciCache.values[iv]
		fibonacciCache.mu.Unlock()
		return new(big.Float).SetPrec(prec).SetInt(v), true
	}
	cur := int64(len(fibonacciCache.values))
	for i := cur; i <= iv; i++ {
		next := new(big.Int).Add(fibonacciCache.values[i-1], fibonacciCache.values[i-2])
		fibonacciCache.values = append(fibonacciCache.values, next)
	}
	v := fibonacciCache.values[iv]
	fibonacciCache.mu.Unlock()
	return new(big.Float).SetPrec(prec).SetInt(v), true
}

func bigPow(base, exp *big.Float, prec uint) (*big.Float, bool) {
	// For integer exponents, use repeated multiplication
	ei, ok := toInt64(exp)
	if ok {
		if ei < 0 {
			// base^(-|ei|) = 1 / base^|ei|
			if base.Cmp(bigZero) == 0 {
				return nil, false
			}
			pos, ok := intPow(base, -ei, prec)
			if !ok {
				return nil, false
			}
			return new(big.Float).SetPrec(prec).Quo(
				new(big.Float).SetPrec(prec).SetInt64(1), pos), true
		}
		return intPow(base, ei, prec)
	}
	// Fallback to float64 for non-integer exponents
	bf, _ := base.Float64()
	ef, _ := exp.Float64()
	if bf < 0 {
		return nil, false
	}
	result := math.Pow(bf, ef)
	if math.IsInf(result, 0) || math.IsNaN(result) {
		return nil, false
	}
	return new(big.Float).SetPrec(prec).SetFloat64(result), true
}

func intPow(base *big.Float, exp int64, prec uint) (*big.Float, bool) {
	if exp > 20 {
		return nil, false
	}
	result := new(big.Float).SetPrec(prec).SetInt64(1)
	b := new(big.Float).SetPrec(prec).Copy(base)
	for exp > 0 {
		if exp%2 == 1 {
			result.Mul(result, b)
		}
		b.Mul(b, b)
		exp /= 2
	}
	return result, true
}

func bigBinomial(nf, kf *big.Float, prec uint) (*big.Float, bool) {
	n, ok := toInt64(nf)
	if !ok || n < 0 || n > 1000 {
		return nil, false
	}
	k, ok := toInt64(kf)
	if !ok || k < 0 || k > n {
		return nil, false
	}
	if k > n-k {
		k = n - k
	}
	result := new(big.Int).SetInt64(1)
	for i := int64(0); i < k; i++ {
		result.Mul(result, big.NewInt(n-i))
		result.Div(result, big.NewInt(i+1))
	}
	return new(big.Float).SetPrec(prec).SetInt(result), true
}

func bigFloor(f *big.Float, prec uint) *big.Float {
	i, _ := f.Int(nil)
	result := new(big.Float).SetPrec(prec).SetInt(i)
	// If f was negative and not an integer, subtract 1
	if f.Sign() < 0 && result.Cmp(f) != 0 {
		result.Sub(result, new(big.Float).SetPrec(prec).SetInt64(1))
	}
	return result
}

func bigCeil(f *big.Float, prec uint) *big.Float {
	i, _ := f.Int(nil)
	result := new(big.Float).SetPrec(prec).SetInt(i)
	if f.Sign() > 0 && result.Cmp(f) != 0 {
		result.Add(result, new(big.Float).SetPrec(prec).SetInt64(1))
	}
	return result
}
