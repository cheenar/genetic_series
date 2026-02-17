package series

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/wildfunctions/genetic_series/pkg/expr"
)

// ParseCandidateLatex parses a LaTeX formula into a Candidate. Supported forms:
//
//	\sum_{n=0}^{\infty} \frac{NUM}{DEN}
//	\sum_{n=0}^{\infty} EXPR
//	\frac{A}{B} \sum_{n=0}^{\infty} \frac{NUM}{DEN}           (outer coefficient)
//	COEFF \sum_{n=0}^{\infty} \frac{C}{D} \frac{E}{F}         (multiple fracs)
//
// The summation variable can be any single letter (k, i, m, ...); it is
// normalized to n internally.
func ParseCandidateLatex(s string) (*Candidate, error) {
	// Normalize whitespace so newlines don't trip up the parser.
	s = strings.Join(strings.Fields(s), " ")

	// Find \sum_{ and extract the variable name.
	sumIdx := strings.Index(s, `\sum_{`)
	if sumIdx < 0 {
		return nil, fmt.Errorf("expected \\sum_{n=... in formula")
	}
	varPos := sumIdx + len(`\sum_{`)
	if varPos+2 > len(s) || s[varPos+1] != '=' || !unicode.IsLetter(rune(s[varPos])) {
		return nil, fmt.Errorf("expected \\sum_{VAR=... at pos %d", sumIdx)
	}
	varName := s[varPos]
	if varName != 'n' {
		// Replace the variable letter with n everywhere in the body (after \sum).
		body := strings.ReplaceAll(s[varPos+2:], string(varName), "n")
		s = s[:varPos] + "n=" + body
		sumIdx = strings.Index(s, `\sum_{`)
	}

	// Parse optional leading coefficient.
	var coeffNum, coeffDen expr.ExprNode
	if prefix := strings.TrimSpace(s[:sumIdx]); prefix != "" {
		coeff, err := expr.ParseExprLatex(prefix)
		if err != nil {
			return nil, fmt.Errorf("parsing coefficient: %w", err)
		}
		coeffNum, coeffDen = splitFraction(coeff)
	}

	// Parse \sum_{n=start}^{\infty}
	p := expr.NewLatexParser(s[sumIdx:])
	if err := p.Consume(`\sum_{n=`); err != nil {
		return nil, err
	}
	start, err := p.ParseInt()
	if err != nil {
		return nil, fmt.Errorf("parsing start index: %w", err)
	}
	if err := p.Consume(`}^{\infty}`); err != nil {
		return nil, err
	}
	p.SkipSpaces()

	// Parse the body as a full expression — handles \frac{}{}, \frac{}{}\frac{}{},
	// implicit multiplication, infix ops, etc.
	if p.Pos() >= p.Len() {
		return nil, fmt.Errorf("expected series body after \\sum")
	}
	body, err := p.ParseExpr()
	if err != nil {
		return nil, fmt.Errorf("parsing series body: %w", err)
	}
	p.SkipSpaces()
	if p.Pos() < p.Len() {
		return nil, fmt.Errorf("unexpected trailing input at pos %d: %q", p.Pos(), p.Remaining())
	}

	// Decompose body into numerator/denominator.
	num, den := splitFraction(body)

	// Fold coefficient in.
	if coeffNum != nil {
		num = maybeMul(coeffNum, num)
		den = maybeMul(coeffDen, den)
	}

	return &Candidate{Numerator: num, Denominator: den, Start: start}, nil
}

// splitFraction recursively decomposes an expression into (numerator, denominator).
//   - Div(a, b)       → (a, b)
//   - Mul(a, b)       → (a_num * b_num, a_den * b_den)
//   - anything else   → (expr, 1)
func splitFraction(node expr.ExprNode) (num, den expr.ExprNode) {
	if b, ok := node.(*expr.BinaryNode); ok {
		if b.Op == expr.OpDiv {
			return b.Left, b.Right
		}
		if b.Op == expr.OpMul {
			lNum, lDen := splitFraction(b.Left)
			rNum, rDen := splitFraction(b.Right)
			return maybeMul(lNum, rNum), maybeMul(lDen, rDen)
		}
	}
	return node, &expr.ConstNode{Val: 1}
}

// maybeMul multiplies two expressions, skipping if either is the constant 1.
func maybeMul(a, b expr.ExprNode) expr.ExprNode {
	if isOne(a) {
		return b
	}
	if isOne(b) {
		return a
	}
	return &expr.BinaryNode{Op: expr.OpMul, Left: a, Right: b}
}

func isOne(n expr.ExprNode) bool {
	c, ok := n.(*expr.ConstNode)
	return ok && c.Val == 1
}
