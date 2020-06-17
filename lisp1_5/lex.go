// Copyright 2020 Rob Pike. All rights reserved.
// Use of this source code is governed by a BSD
// license that can be found in the LICENSE file.

package lisp1_5

import (
	"bytes"
	"fmt"
	"io"
	"math/big"
	"os"
	"strings"
	"unicode"
)

//go:generate stringer -type TokType -trimprefix Tok
type TokType int

const (
	tokenError TokType = iota
	TokenEOF
	tokenAtom
	tokenConst
	tokenNumber
	tokenLpar
	tokenRpar
	tokenDot
	tokenChar
	tokenQuote
	tokenNewline
)

const EofRune rune = -1 // Returned by Parser.SkipSpace at EOF.

// A token is a Lisp atom, including a number.
type token struct {
	typ  TokType
	text string   // User's input text, empty for numbers.
	num  *big.Int // Nil for non-numbers.
}

func (t token) String() string {
	if t.typ == tokenNumber {
		return fmt.Sprint(t.num)
	}
	return t.text
}

func (t token) buildString(b *strings.Builder) {
	if t.typ == tokenNumber {
		b.WriteString(fmt.Sprint(t.num))
	} else {
		b.WriteString(t.text)
	}
}

type lexer struct {
	rd       io.RuneReader
	peeking  bool
	peekRune rune
	last     rune
	buf      bytes.Buffer
}

func newLexer(rd io.RuneReader) *lexer {
	return &lexer{
		rd: rd,
	}
}

var atoms = make(map[string]*token)

var zero big.Int

func mkToken(typ TokType, text string) *token {
	if typ == tokenNumber {
		var z big.Int
		num, ok := z.SetString(text, 0)
		if !ok {
			errorf("bad number syntax: %s", text)
		}
		return number(num)
	}
	tok := atoms[text]
	if tok == nil {
		tok = &token{typ, text, &zero}
		atoms[text] = tok
	}
	return tok
}

func number(num *big.Int) *token {
	return &token{tokenNumber, "", num}
}

func mkAtom(text string) *token {
	return mkToken(tokenAtom, text)
}

func (l *lexer) skipSpace() rune {
	comment := false
	for {
		r := l.read()
		if r == '\n' || r == EofRune {
			return r
		}
		if r == ';' {
			comment = true
			continue
		}
		if !comment && !isSpace(r) {
			l.back(r)
			return r
		}
	}
}

func (l *lexer) skipToNewline() {
	for l.last != '\n' && l.last != EofRune {
		l.nextRune()
	}
	l.peeking = false
}

func (l *lexer) next() *token {
	for {
		r := l.read()
		typ := tokenAtom
		switch {
		case isSpace(r):
		case r == ';':
			l.skipToNewline()
		case r == EofRune:
			return mkToken(TokenEOF, "EOF")
		case r == '\n':
			return mkToken(tokenNewline, "\n")
		case r == '(':
			return mkToken(tokenLpar, "(")
		case r == ')':
			return mkToken(tokenRpar, ")")
		case r == '.':
			return mkToken(tokenDot, ".")
		case r == '-' || r == '+':
			if !isNumber(l.peek()) {
				return mkToken(tokenChar, string(r))
			}
			fallthrough
		case isNumber(r):
			return l.number(r)
		case r == '\'':
			return mkToken(tokenQuote, "'")
		case r == '_' || unicode.IsLetter(r):
			return l.alphanum(typ, r)
		default:
			return mkToken(tokenChar, string(r))
		}
	}
}

func (l *lexer) read() rune {
	if l.peeking {
		l.peeking = false
		return l.peekRune
	}
	return l.nextRune()
}

func (l *lexer) nextRune() rune {
	r, _, err := l.rd.ReadRune()
	if err != nil {
		if err != io.EOF {
			fmt.Fprintln(os.Stderr)
		}
		r = EofRune
	}
	l.last = r
	return r
}

func (l *lexer) peek() rune {
	if l.peeking {
		return l.peekRune
	}
	r := l.read()
	l.peeking = true
	l.peekRune = r
	return r
}

func (l *lexer) back(r rune) {
	l.peeking = true
	l.peekRune = r
}

func (l *lexer) accum(r rune, valid func(rune) bool) {
	l.buf.Reset()
	for {
		l.buf.WriteRune(r)
		r = l.read()
		if r == EofRune {
			return
		}
		if !valid(r) {
			l.back(r)
			return
		}
	}
}

func isSpace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}

func isNumber(r rune) bool {
	return '0' <= r && r <= '9'
}

func isAlphanum(r rune) bool {
	return r == '_' || isNumber(r) || unicode.IsLetter(r)
}

func (l *lexer) number(r rune) *token {
	// Integer only for now.
	l.accum(r, isNumber)
	l.endToken()
	return mkToken(tokenNumber, l.buf.String())
}

func (l *lexer) alphanum(typ TokType, r rune) *token {
	// TODO: ASCII only for now.
	l.accum(r, isAlphanum)
	l.endToken()
	return mkToken(typ, l.buf.String())
}

// endToken guarantees that the following rune separates this token from the next.
func (l *lexer) endToken() {
	if r := l.peek(); isAlphanum(r) || !isSpace(r) && r != '(' && r != ')' && r != '.' && r != EofRune {
		errorf("invalid token after %s", &l.buf)
	}
}

var (
	// Pre-defined constants.
	tokF   = mkToken(tokenConst, "F")
	tokT   = mkToken(tokenConst, "T")
	tokNil = mkToken(tokenConst, "nil")

	// Pre-defined elementary functions and symbols.
	tokAdd         = mkAtom("add")
	tokAnd         = mkAtom("and")
	tokApply       = mkAtom("apply")
	tokAtom        = mkAtom("atom")
	tokCar         = mkAtom("car")
	tokCdr         = mkAtom("cdr")
	tokCond        = mkAtom("cond")
	tokCons        = mkAtom("cons")
	tokDefn        = mkAtom("defn")
	tokDiv         = mkAtom("div")
	tokEq          = mkAtom("eq")
	tokGe          = mkAtom("ge")
	tokASCIILambda = mkAtom("lambda")
	tokGt          = mkAtom("gt")
	tokLambda      = mkAtom("λ")
	tokLe          = mkAtom("le")
	tokList        = mkAtom("list")
	tokLt          = mkAtom("lt")
	tokMul         = mkAtom("mul")
	tokNe          = mkAtom("ne")
	tokOr          = mkAtom("or")
	tokNull        = mkAtom("null")
	tokQuote       = mkAtom("quote")
	tokRem         = mkAtom("rem")
	tokSub         = mkAtom("sub")
)
