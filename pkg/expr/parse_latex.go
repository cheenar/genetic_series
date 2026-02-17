package expr

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// ParseExprLatex parses a single LaTeX expression into an ExprNode.
func ParseExprLatex(s string) (ExprNode, error) {
	p := &LatexParser{src: s}
	node, err := p.ParseExpr()
	if err != nil {
		return nil, err
	}
	p.SkipSpaces()
	if p.pos < len(p.src) {
		return nil, fmt.Errorf("unexpected trailing input at pos %d: %q", p.pos, p.src[p.pos:])
	}
	return node, nil
}

// LatexParser is a recursive-descent parser for LaTeX math expressions.
// Handles both machine-generated (engine output) and human-written LaTeX.
//
// Precedence (low to high):
//  1. + - (additive)
//  2. implicit multiplication, \cdot (multiplicative)
//  3. unary minus
//  4. ! !! ^ (postfix)
//  5. primaries: numbers, n, \frac, \sqrt, (...), {...}, ...
type LatexParser struct {
	src string
	pos int
}

// NewLatexParser creates a parser for the given input string.
func NewLatexParser(s string) *LatexParser {
	return &LatexParser{src: s}
}

// Pos returns the current position.
func (p *LatexParser) Pos() int { return p.pos }

// Len returns the input length.
func (p *LatexParser) Len() int { return len(p.src) }

// Remaining returns the unconsumed input.
func (p *LatexParser) Remaining() string { return p.src[p.pos:] }

func (p *LatexParser) peek() byte {
	if p.pos >= len(p.src) {
		return 0
	}
	return p.src[p.pos]
}

// HasPrefix checks if the remaining input starts with s.
func (p *LatexParser) HasPrefix(s string) bool {
	return strings.HasPrefix(p.src[p.pos:], s)
}

// Consume expects and consumes a literal string; errors if mismatch.
func (p *LatexParser) Consume(s string) error {
	if !p.HasPrefix(s) {
		got := p.src[p.pos:]
		if len(got) > 20 {
			got = got[:20] + "..."
		}
		return fmt.Errorf("expected %q at pos %d, got %q", s, p.pos, got)
	}
	p.pos += len(s)
	return nil
}

// SkipSpaces skips whitespace and LaTeX spacing commands (\, \; \! \quad etc.).
func (p *LatexParser) SkipSpaces() {
	for p.pos < len(p.src) {
		if p.src[p.pos] == ' ' {
			p.pos++
			continue
		}
		// Skip \, \; \! \: \quad \qquad (LaTeX spacing)
		if p.src[p.pos] == '\\' && p.pos+1 < len(p.src) {
			next := p.src[p.pos+1]
			if next == ',' || next == ';' || next == '!' || next == ':' || next == ' ' {
				p.pos += 2
				continue
			}
			if strings.HasPrefix(p.src[p.pos:], `\quad`) {
				p.pos += 5
				continue
			}
			if strings.HasPrefix(p.src[p.pos:], `\qquad`) {
				p.pos += 6
				continue
			}
		}
		break
	}
}

// ParseExpr parses a full expression (entry point for each expression context).
func (p *LatexParser) ParseExpr() (ExprNode, error) {
	return p.parseAddSub()
}

// parseAddSub handles infix + and - (lowest precedence).
func (p *LatexParser) parseAddSub() (ExprNode, error) {
	left, err := p.parseMul()
	if err != nil {
		return nil, err
	}
	for {
		p.SkipSpaces()
		if p.peek() == '+' {
			p.pos++
			p.SkipSpaces()
			right, err := p.parseMul()
			if err != nil {
				return nil, err
			}
			left = &BinaryNode{Op: OpAdd, Left: left, Right: right}
			continue
		}
		if p.peek() == '-' {
			p.pos++
			p.SkipSpaces()
			right, err := p.parseMul()
			if err != nil {
				return nil, err
			}
			left = &BinaryNode{Op: OpSub, Left: left, Right: right}
			continue
		}
		break
	}
	return left, nil
}

// parseMul handles explicit (\cdot) and implicit multiplication.
func (p *LatexParser) parseMul() (ExprNode, error) {
	left, err := p.parseFactor()
	if err != nil {
		return nil, err
	}
	for {
		p.SkipSpaces()
		if p.HasPrefix(`\cdot`) {
			p.pos += 5
			p.SkipSpaces()
			right, err := p.parseFactor()
			if err != nil {
				return nil, err
			}
			left = &BinaryNode{Op: OpMul, Left: left, Right: right}
			continue
		}
		if p.canStartImplicitMul() {
			right, err := p.parseFactor()
			if err != nil {
				return nil, err
			}
			left = &BinaryNode{Op: OpMul, Left: left, Right: right}
			continue
		}
		break
	}
	return left, nil
}

// parseFactor handles unary minus (binds tighter than +/- but looser than postfix).
func (p *LatexParser) parseFactor() (ExprNode, error) {
	if p.peek() == '-' {
		// Negative integer literal: let parsePrimary handle it.
		if p.pos+1 < len(p.src) && unicode.IsDigit(rune(p.src[p.pos+1])) {
			return p.parsePostfix()
		}
		// Unary minus.
		p.pos++
		p.SkipSpaces()
		child, err := p.parseFactor()
		if err != nil {
			return nil, err
		}
		return &UnaryNode{Op: OpNeg, Child: child}, nil
	}
	return p.parsePostfix()
}

// parsePostfix handles ! !! and ^ (highest precedence).
func (p *LatexParser) parsePostfix() (ExprNode, error) {
	node, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	for {
		if p.HasPrefix("!!") {
			p.pos += 2
			node = &UnaryNode{Op: OpDoubleFactorial, Child: node}
			continue
		}
		if p.peek() == '!' {
			p.pos++
			node = &UnaryNode{Op: OpFactorial, Child: node}
			continue
		}
		if p.peek() == '^' {
			p.pos++
			var exp ExprNode
			if p.peek() == '{' {
				p.pos++
				exp, err = p.ParseExpr()
				if err != nil {
					return nil, err
				}
				if err := p.Consume("}"); err != nil {
					return nil, err
				}
			} else {
				// Bare exponent: single primary (e.g. ^2, ^n)
				exp, err = p.parsePrimary()
				if err != nil {
					return nil, err
				}
			}
			node = &BinaryNode{Op: OpPow, Left: node, Right: exp}
			continue
		}
		break
	}
	return node, nil
}

// parsePrimary parses an atomic expression.
func (p *LatexParser) parsePrimary() (ExprNode, error) {
	if p.pos >= len(p.src) {
		return nil, fmt.Errorf("unexpected end of input at pos %d", p.pos)
	}

	// \frac{...}{...}
	if p.HasPrefix(`\frac{`) {
		p.pos += 6
		num, err := p.ParseExpr()
		if err != nil {
			return nil, err
		}
		if err := p.Consume("}{"); err != nil {
			return nil, err
		}
		den, err := p.ParseExpr()
		if err != nil {
			return nil, err
		}
		if err := p.Consume("}"); err != nil {
			return nil, err
		}
		return &BinaryNode{Op: OpDiv, Left: num, Right: den}, nil
	}

	// \binom{...}{...}
	if p.HasPrefix(`\binom{`) {
		p.pos += 7
		left, err := p.ParseExpr()
		if err != nil {
			return nil, err
		}
		if err := p.Consume("}{"); err != nil {
			return nil, err
		}
		right, err := p.ParseExpr()
		if err != nil {
			return nil, err
		}
		if err := p.Consume("}"); err != nil {
			return nil, err
		}
		return &BinaryNode{Op: OpBinomial, Left: left, Right: right}, nil
	}

	// \sqrt{...}
	if p.HasPrefix(`\sqrt{`) {
		p.pos += 6
		child, err := p.ParseExpr()
		if err != nil {
			return nil, err
		}
		if err := p.Consume("}"); err != nil {
			return nil, err
		}
		return &UnaryNode{Op: OpSqrt, Child: child}, nil
	}

	// \sin, \cos, \ln — accept both {(expr)} (engine) and (expr) (user)
	if p.HasPrefix(`\sin`) {
		p.pos += 4
		child, err := p.parseFuncArg()
		if err != nil {
			return nil, err
		}
		return &UnaryNode{Op: OpSin, Child: child}, nil
	}
	if p.HasPrefix(`\cos`) {
		p.pos += 4
		child, err := p.parseFuncArg()
		if err != nil {
			return nil, err
		}
		return &UnaryNode{Op: OpCos, Child: child}, nil
	}
	if p.HasPrefix(`\ln`) {
		p.pos += 3
		child, err := p.parseFuncArg()
		if err != nil {
			return nil, err
		}
		return &UnaryNode{Op: OpLn, Child: child}, nil
	}

	// \lfloor ... \rfloor
	if p.HasPrefix(`\lfloor`) {
		p.pos += 7
		p.SkipSpaces()
		child, err := p.ParseExpr()
		if err != nil {
			return nil, err
		}
		p.SkipSpaces()
		if err := p.Consume(`\rfloor`); err != nil {
			return nil, err
		}
		return &UnaryNode{Op: OpFloor, Child: child}, nil
	}

	// \lceil ... \rceil
	if p.HasPrefix(`\lceil`) {
		p.pos += 6
		p.SkipSpaces()
		child, err := p.ParseExpr()
		if err != nil {
			return nil, err
		}
		p.SkipSpaces()
		if err := p.Consume(`\rceil`); err != nil {
			return nil, err
		}
		return &UnaryNode{Op: OpCeil, Child: child}, nil
	}

	// |...| → OpAbs
	if p.peek() == '|' {
		p.pos++
		child, err := p.ParseExpr()
		if err != nil {
			return nil, err
		}
		if err := p.Consume("|"); err != nil {
			return nil, err
		}
		return &UnaryNode{Op: OpAbs, Child: child}, nil
	}

	// (-1)^{...} → OpAltSign (must come before general '(' handling)
	if p.HasPrefix(`(-1)^{`) {
		p.pos += 6
		child, err := p.ParseExpr()
		if err != nil {
			return nil, err
		}
		if err := p.Consume("}"); err != nil {
			return nil, err
		}
		return &UnaryNode{Op: OpAltSign, Child: child}, nil
	}

	// F_{...} → OpFibonacci
	if p.HasPrefix(`F_{`) {
		p.pos += 3
		child, err := p.ParseExpr()
		if err != nil {
			return nil, err
		}
		if err := p.Consume("}"); err != nil {
			return nil, err
		}
		return &UnaryNode{Op: OpFibonacci, Child: child}, nil
	}

	// {...} → brace grouping
	if p.peek() == '{' {
		p.pos++
		node, err := p.ParseExpr()
		if err != nil {
			return nil, err
		}
		if err := p.Consume("}"); err != nil {
			return nil, err
		}
		return node, nil
	}

	// (...) → paren grouping
	if p.peek() == '(' {
		p.pos++
		node, err := p.ParseExpr()
		if err != nil {
			return nil, err
		}
		if err := p.Consume(")"); err != nil {
			return nil, err
		}
		return node, nil
	}

	// n → VarNode
	if p.peek() == 'n' {
		p.pos++
		return &VarNode{}, nil
	}

	// Integer (possibly negative)
	if p.peek() == '-' || unicode.IsDigit(rune(p.peek())) {
		v, err := p.ParseInt()
		if err != nil {
			return nil, err
		}
		return &ConstNode{Val: v}, nil
	}

	got := p.src[p.pos:]
	if len(got) > 20 {
		got = got[:20] + "..."
	}
	return nil, fmt.Errorf("unexpected token at pos %d: %q", p.pos, got)
}

// parseFuncArg parses a function argument in {(expr)}, (expr), or {expr} form.
func (p *LatexParser) parseFuncArg() (ExprNode, error) {
	// Engine format: {(expr)}
	if p.HasPrefix("{(") {
		p.pos += 2
		child, err := p.ParseExpr()
		if err != nil {
			return nil, err
		}
		if err := p.Consume(")}"); err != nil {
			return nil, err
		}
		return child, nil
	}
	// User format: (expr)
	if p.peek() == '(' {
		p.pos++
		child, err := p.ParseExpr()
		if err != nil {
			return nil, err
		}
		if err := p.Consume(")"); err != nil {
			return nil, err
		}
		return child, nil
	}
	// Bare brace: {expr}
	if p.peek() == '{' {
		p.pos++
		child, err := p.ParseExpr()
		if err != nil {
			return nil, err
		}
		if err := p.Consume("}"); err != nil {
			return nil, err
		}
		return child, nil
	}
	return nil, fmt.Errorf("expected function argument at pos %d", p.pos)
}

// canStartImplicitMul checks if the current position could begin a new primary
// expression, triggering implicit multiplication.
func (p *LatexParser) canStartImplicitMul() bool {
	if p.pos >= len(p.src) {
		return false
	}
	c := p.src[p.pos]
	if unicode.IsDigit(rune(c)) || c == 'n' || c == '(' || c == '{' {
		return true
	}
	if c == 'F' && p.pos+1 < len(p.src) && p.src[p.pos+1] == '_' {
		return true
	}
	if c == '\\' {
		rest := p.src[p.pos:]
		return strings.HasPrefix(rest, `\frac`) ||
			strings.HasPrefix(rest, `\binom`) ||
			strings.HasPrefix(rest, `\sqrt`) ||
			strings.HasPrefix(rest, `\sin`) ||
			strings.HasPrefix(rest, `\cos`) ||
			strings.HasPrefix(rest, `\ln`) ||
			strings.HasPrefix(rest, `\lfloor`) ||
			strings.HasPrefix(rest, `\lceil`)
	}
	return false
}

// ParseInt parses a (possibly negative) integer.
func (p *LatexParser) ParseInt() (int64, error) {
	start := p.pos
	if p.pos < len(p.src) && p.src[p.pos] == '-' {
		p.pos++
	}
	if p.pos >= len(p.src) || !unicode.IsDigit(rune(p.src[p.pos])) {
		return 0, fmt.Errorf("expected integer at pos %d", start)
	}
	for p.pos < len(p.src) && unicode.IsDigit(rune(p.src[p.pos])) {
		p.pos++
	}
	v, err := strconv.ParseInt(p.src[start:p.pos], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid integer %q: %w", p.src[start:p.pos], err)
	}
	return v, nil
}
