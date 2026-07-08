package annotation

import (
	"fmt"
	"strconv"

	"github.com/RomanAgaltsev/bigo/internal/bound"
)

type tokKind int

const (
	tEOF tokKind = iota
	tIdent
	tNum
	tCaret
	tPlus
	tStar
	tLParen
	tRParen
)

type token struct {
	kind tokKind
	text string
	num  int
}

func isIdentStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isIdentPart(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}

func lex(s string) ([]token, error) {
	var toks []token
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == ' ' || c == '\t':
			i++
		case c == '^':
			toks = append(toks, token{kind: tCaret})
			i++
		case c == '+':
			toks = append(toks, token{kind: tPlus})
			i++
		case c == '*':
			toks = append(toks, token{kind: tStar})
			i++
		case c == '(':
			toks = append(toks, token{kind: tLParen})
			i++
		case c == ')':
			toks = append(toks, token{kind: tRParen})
			i++
		case c >= '0' && c <= '9':
			j := i
			for j < len(s) && s[j] >= '0' && s[j] <= '9' {
				j++
			}
			n, _ := strconv.Atoi(s[i:j])
			toks = append(toks, token{kind: tNum, num: n})
			i = j
		case isIdentStart(c):
			j := i
			for j < len(s) && isIdentPart(s[j]) {
				j++
			}
			toks = append(toks, token{kind: tIdent, text: s[i:j]})
			i = j
		default:
			return nil, fmt.Errorf("unexpected character %q", string(c))
		}
	}
	return append(toks, token{kind: tEOF}), nil
}

type parser struct {
	toks []token
	pos  int
}

func (p *parser) cur() token { return p.toks[p.pos] }
func (p *parser) next()      { p.pos++ }

func (p *parser) accept(k tokKind) bool {
	if p.cur().kind == k {
		p.next()
		return true
	}
	return false
}

func (p *parser) acceptIdent(text string) bool {
	if p.cur().kind == tIdent && p.cur().text == text {
		p.next()
		return true
	}
	return false
}

// parseBigO parses a full "O(...)" expression into a bound.Bound.
func parseBigO(s string) (bound.Bound, error) {
	toks, err := lex(s)
	if err != nil {
		return bound.Bound{}, err
	}
	p := &parser{toks: toks}
	if !p.acceptIdent("O") {
		return bound.Bound{}, fmt.Errorf("expected 'O(' in %q", s)
	}
	if !p.accept(tLParen) {
		return bound.Bound{}, fmt.Errorf("expected '(' in %q", s)
	}
	b, err := p.parseSum()
	if err != nil {
		return bound.Bound{}, err
	}
	if !p.accept(tRParen) {
		return bound.Bound{}, fmt.Errorf("expected ')' in %q", s)
	}
	if p.cur().kind != tEOF {
		return bound.Bound{}, fmt.Errorf("trailing tokens after ')' in %q", s)
	}
	return b, nil
}

func (p *parser) parseSum() (bound.Bound, error) {
	m, err := p.parseTerm()
	if err != nil {
		return bound.Bound{}, err
	}
	b := bound.Of(m)
	for p.accept(tPlus) {
		m2, err := p.parseTerm()
		if err != nil {
			return bound.Bound{}, err
		}
		b = b.Join(bound.Of(m2))
	}
	return b, nil
}

func (p *parser) parseTerm() (bound.Monomial, error) {
	m, err := p.parseFactor()
	if err != nil {
		return bound.Monomial{}, err
	}
	for {
		if p.accept(tStar) {
			n, err := p.parseFactor()
			if err != nil {
				return bound.Monomial{}, err
			}
			m = m.Mul(n)
			continue
		}
		// implicit multiplication by adjacency (e.g. "n log n").
		if p.cur().kind == tIdent || p.cur().kind == tNum {
			n, err := p.parseFactor()
			if err != nil {
				return bound.Monomial{}, err
			}
			m = m.Mul(n)
			continue
		}
		break
	}
	return m, nil
}

func (p *parser) parseFactor() (bound.Monomial, error) {
	t := p.cur()
	switch t.kind {
	case tNum:
		if t.num == 1 {
			p.next()
			return bound.One(), nil
		}
		return bound.Monomial{}, fmt.Errorf("only the constant 1 is allowed, got %d", t.num)
	case tIdent:
		if t.text == "log" {
			return p.parseLog()
		}
		v := bound.Var(t.text)
		p.next()
		pow, err := p.optExponent()
		if err != nil {
			return bound.Monomial{}, err
		}
		return bound.Mono(v, pow, 0), nil
	default:
		return bound.Monomial{}, fmt.Errorf("unexpected token in expression")
	}
}

func (p *parser) parseLog() (bound.Monomial, error) {
	p.next() // consume 'log'
	var v string
	if p.accept(tLParen) {
		if p.cur().kind != tIdent {
			return bound.Monomial{}, fmt.Errorf("expected variable after 'log('")
		}
		v = p.cur().text
		p.next()
		if !p.accept(tRParen) {
			return bound.Monomial{}, fmt.Errorf("expected ')' after log argument")
		}
	} else if p.cur().kind == tIdent {
		v = p.cur().text
		p.next()
	} else {
		return bound.Monomial{}, fmt.Errorf("expected variable after 'log'")
	}
	logPow, err := p.optExponent()
	if err != nil {
		return bound.Monomial{}, err
	}
	return bound.Mono(bound.Var(v), 0, logPow), nil
}

// optExponent parses an optional "^int" suffix, defaulting to 1.
func (p *parser) optExponent() (int, error) {
	if !p.accept(tCaret) {
		return 1, nil
	}
	if p.cur().kind != tNum {
		return 0, fmt.Errorf("expected integer exponent after '^'")
	}
	n := p.cur().num
	p.next()
	return n, nil
}
