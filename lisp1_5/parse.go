// Copyright 2020 Rob Pike. All rights reserved.
// Use of this source code is governed by a BSD
// license that can be found in the LICENSE file.

package lisp1_5

import (
	"fmt"
	"io"
	"log"
	"strings"
)

// printSExpr configures the interpreter to print S-Expressions.
var printSExpr bool

// Config configures the interpreter. The argument specifies whether output
// should be S-Expressions rather than lists.
func Config(alwaysPrintSExprs bool) {
	printSExpr = alwaysPrintSExprs
}

// Expr represents an arbitrary expression.
type Expr struct {
	// An Expr is either an atom, with atom set, or a list, with car and cdr set.
	// Car and cdr can be nil (empty) even if atom is nil.
	car  *Expr
	atom *token
	cdr  *Expr
}

// SExprString returns the expression as a formatted S-Expression.
func (e *Expr) SExprString() string {
	if e == nil {
		return "nil"
	}
	if e.atom != nil {
		return e.atom.String()
	}
	str := "("
	str += e.car.SExprString()
	str += " . "
	str += e.cdr.SExprString()
	str += ")"
	return str
}

// String returns the expression as a formatted list (unless printSExpr is set).
func (e *Expr) String() string {
	if printSExpr {
		return e.SExprString()
	}
	if e == nil {
		return "nil"
	}
	var b strings.Builder
	e.buildString(&b, true)
	return b.String()
}

// buildString is the internals of the String method. simplifyQuote
// specifies whether (quote expr) should be printed as 'expr.
func (e *Expr) buildString(b *strings.Builder, simplifyQuote bool) {
	if e == nil {
		b.WriteString("nil")
		return
	}
	if e.atom != nil {
		e.atom.buildString(b)
		return
	}
	// Simplify (quote a) to 'a.
	if simplifyQuote && Car(e).getAtom() == tokQuote {
		b.WriteByte('\'')
		Car(Cdr(e)).buildString(b, simplifyQuote)
		return
	}
	b.WriteByte('(')
	for {
		car, cdr := e.car, e.cdr
		car.buildString(b, simplifyQuote)
		if cdr == nil {
			break
		}
		if cdr.atom != nil {
			if cdr.atom.text == "nil" {
				break
			}
			b.WriteString(" . ")
			cdr.buildString(b, simplifyQuote)
			break
		}
		b.WriteByte(' ')
		e = cdr
	}
	b.WriteByte(')')
}

// Parser is the parser for lists.
type Parser struct {
	lex     *lexer
	peekTok *token
}

// NewParser returns a new parser that will read from the RuneReader.
// Parse errors cause panics of type Error that the caller must handle.
func NewParser(r io.RuneReader) *Parser {
	return &Parser{
		lex:     newLexer(r),
		peekTok: nil,
	}
}

// SkipSpace skips leading spaces, returning the rune that follows.
func (p *Parser) SkipSpace() rune {
	return p.lex.skipSpace()
}

// SkipToNewline advances the input past the next newline.
func (p *Parser) SkipToEndOfLine() {
	p.lex.skipToNewline()
}

func errorf(format string, args ...interface{}) {
	panic(Error(fmt.Sprintf(format, args...)))
}

func (p *Parser) next() *token {
	if tok := p.peekTok; tok != nil {
		p.peekTok = nil
		return tok
	}
	return p.lex.next()
}

func (p *Parser) back(tok *token) {
	p.peekTok = tok
}

// sExpr parses an S-Expression.
// SExpr:
//	Atom
//	Lpar SExpr Dot SExpr Rpar
func (p *Parser) SExpr() *Expr {
	tok := p.next()
	switch tok.typ {
	case TokenEOF:
		return nil
	case tokenQuote:
		return p.quote()
	case tokenAtom, tokenConst, tokenNumber:
		return atomExpr(tok)
	case tokenLpar:
		car := p.SExpr()
		dot := p.next()
		if dot.typ != tokenDot {
			log.Fatal("expected dot, found ", dot)
		}
		cdr := p.SExpr()
		rpar := p.next()
		if rpar.typ != tokenRpar {
			log.Fatal("expected rPar, found ", rpar)
		}
		return Cons(car, cdr)
	}
	errorf("bad token in SExpr: %q", tok)
	panic("not reached")
}

// quote parses a quoted expression. The leading quote has been consumed.
func (p *Parser) quote() *Expr {
	return Cons(atomExpr(tokQuote), Cons(p.List(), nil))
}

// List parses a list expression.
func (p *Parser) List() *Expr {
	tok := p.next()
	switch tok.typ {
	case TokenEOF:
		panic(EOF("eof"))
	case tokenQuote:
		return p.quote()
	case tokenAtom, tokenConst, tokenNumber:
		return atomExpr(tok)
	case tokenLpar:
		expr := p.lparList()
		tok = p.next()
		if tok.typ == tokenRpar {
			return expr
		}
	}
	errorf("bad token in list: %q", tok)
	panic("not reached")
}

// lparList parses the innards of a list, up to the closing paren.
// The opening paren has been consumed.
func (p *Parser) lparList() *Expr {
	tok := p.next()
	switch tok.typ {
	case tokenQuote:
		return Cons(p.quote(), p.lparList())
	case tokenAtom, tokenConst, tokenNumber:
		return Cons(atomExpr(tok), p.lparList())
	case tokenDot:
		return p.List()
	case tokenLpar:
		p.back(tok)
		return Cons(p.List(), p.lparList())
	case tokenRpar:
		p.back(tok)
		return nil
	}
	errorf("bad token parsing list: %q", tok)
	panic("not reached")
}
