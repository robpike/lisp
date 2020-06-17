// Copyright 2020 Rob Pike. All rights reserved.
// Use of this source code is governed by a BSD
// license that can be found in the LICENSE file.

package lisp1_5

import (
	"strings"
	"testing"
)

var consTests = []struct {
	a, b string
	c    string
}{
	{"a", "b", "(a . b)"},
	{"(a . b)", "c", "((a . b) . c)"},
}

func TestCons(t *testing.T) {
	for _, test := range consTests {
		a := NewParser(strings.NewReader(test.a)).SExpr()
		b := NewParser(strings.NewReader(test.b)).SExpr()
		c := Cons(a, b)
		str := c.SExprString()
		if str != test.c {
			t.Errorf("cons(%s, %s) = %s, expected %s", test.a, test.b, str, test.c)
		}
	}
}

func strEval(str string) string {
	p := NewParser(strings.NewReader(str))
	return NewContext(0).Eval(p.List()).String()
}

var consEvalTests = []struct {
	in  string
	out string
}{
	// A nice little set from
	// https://medium.com/@aleksandrasays/my-other-car-is-a-cdr-3058e6743c15
	{"(cons 1 2)", "(1 . 2)"},
	{"(cons 'a (cons 'b (cons 'c '())))", "(a b c)"},
	{"(list 'a 'b 'c)", "(a b c)"}, // Original has c, not 'c, can't be right.
	{"(cons 1 '(2 3 4))", "(1 2 3 4)"},
	{"(cons '(a b c) ())", "((a b c))"},
	{"(cons '(a b c) '(d))", "((a b c) d)"},
}

func TestConsEval(t *testing.T) {
	for _, test := range consEvalTests {
		if got := strEval(test.in); got != test.out {
			t.Errorf("%s = %s, expected %s", test.in, got, test.out)
		}
	}
}

func TestApply(t *testing.T) {
	l := "(λ (x y) (cons (car x) y))"
	lambda := NewParser(strings.NewReader(l)).List()
	a := "((a b) (c d))"
	args := NewParser(strings.NewReader(a)).List()
	c := NewContext(0)
	expr := c.apply(l, lambda, args)
	const want = "(a c d)"
	if expr.String() != want {
		t.Fatal(expr)
	}

	l = "(lambda (x y) (cons (car x) y))" // Be sure to use both forms of lambda.
	lambda = NewParser(strings.NewReader(l)).List()
	a = "((a b) (c d))"
	args = NewParser(strings.NewReader(a)).List()
	c = NewContext(0)
	example := mkAtom("example")
	c.set(example, lambda)
	expr = c.apply(l, atomExpr(example), args)
	if expr.String() != want {
		t.Fatal(expr)
	}
}

var examples = []struct {
	name string
	fn   string
	in   string
	out  string
}{
	{
		"(fac)",
		`(defn(
			(fac (lambda (n) (cond
				((eq n 0) 1)
				(T (mul n (fac (sub n 1))))
			)))
		))`,
		"(fac 100)",
		"93326215443944152681699238856266700490715968264381621468592963895217599993229915608941463976156518286253697920827223758251185210916864000000000000000000000000",
	},
	{
		"(gcd)",
		`(defn(
			(gcd (lambda (x y) (cond
				((gt x y) (gcd y x))
				((eq (rem y x) 0) x)
				(T (gcd (rem y x) x))
			)))
		))`,
		"(gcd 144 64)",
		"16",
	},
	{
		"(ack)",
		`(defn(
			(ack (lambda (m n) (cond
				((eq m 0) (add n 1))
				((eq n 0) (ack (sub m 1) 1))
				(T (ack (sub m 1) (ack m (sub n 1))))
			)))
		))`,
		"(ack 3 4)", // apply called 51535 times!
		"125",
	},
	{
		"(one two three)",
		`(defn(
			(one (lambda (x y) (cons (car x) y)))
			(two (lambda (x y) (one x y)))
			(three (lambda (x y) (two x y)))
		))`,
		"(three '(a b) '(c d))",
		"(a c d)",
	},
	{
		"(testcaaaddr)",
		`(defn(
			(testcaaaddr (lambda (x) (caaddr x)))
		))`,
		"(caaaddr '((1 2) (3 4) ((5 6)) (7 8)))",
		"5",
	},
}

func TestExamples(t *testing.T) {
	for _, test := range examples {
		c := NewContext(0)
		p := NewParser(strings.NewReader(test.fn))
		if got := c.Eval(p.List()).String(); got != test.name {
			t.Errorf("%s = %s, expected %s", test.fn, got, test.name)
		}
		p = NewParser(strings.NewReader(test.in))
		if got := c.Eval(p.List()).String(); got != test.out {
			t.Errorf("%s = %s, expected %s", test.in, got, test.out)
		}
	}
}

func TestAnd(t *testing.T) {
	c := NewContext(0)
	const text = "(and T T T F)"
	p := NewParser(strings.NewReader(text))
	if got := c.Eval(p.List()).String(); got != "F" {
		t.Errorf("%s = %s, expected %s", text, got, "F")
	}
}

func TestOr(t *testing.T) {
	c := NewContext(0)
	const text = "(or F F F T)"
	p := NewParser(strings.NewReader(text))
	if got := c.Eval(p.List()).String(); got != "T" {
		t.Errorf("%s = %s, expected %s", text, got, "T")
	}
}

func TestStackTrace(t *testing.T) {
	const prog = `(defn(
		(error (lambda (x) (cond
			((eq x 0) (div 0 0))
			(T (error (sub x 1)))
		)))
	))`
	const crash = `(error 5)`
	c := NewContext(0)
	p := NewParser(strings.NewReader(prog))
	if got := c.Eval(p.List()).String(); got != "(error)" {
		t.Fatal("did not declare error")
	}
	p = NewParser(strings.NewReader(crash))
	defer func() {
		e := recover()
		_, ok := e.(Error)
		if !ok {
			t.Fatal("no error")
		}
		const expect = "stack: (error 0) (error 1) (error 2) (error 3) (error 4) (error 5)"
		stack := c.StackTrace()
		if strings.Join(strings.Fields(stack), " ") != expect {
			t.Fatal(stack)
		}
	}()
	c.Eval(p.List())
	t.Fatal("did not crash")
}
