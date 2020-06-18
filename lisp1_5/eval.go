// Copyright 2020 Rob Pike. All rights reserved.
// Use of this source code is governed by a BSD
// license that can be found in the LICENSE file.

package lisp1_5 // import "github.com/robpike/lisp/lisp1_5"

import (
	"fmt"
	"strings"
)

type elemFunc func(*Context, *token, *Expr) *Expr
type funcMap map[*token]elemFunc
type frame map[*token]*Expr

// Types used to signal to the outside. Both are returned
// through panic, which the caller is expected to recover from.
type (
	Error string // Error on execution or parse.
	EOF   string // End of file on input.
)

var elementary funcMap
var constT, constF, constNIL *Expr

// A scope is effectively a stack frame.
type scope struct {
	vars frame  // The variables defined in this frame.
	fn   string // The name of the called function, for tracebacks.
	args *Expr  // The arguments of the called function, for tracebacks.
}

// A Context holds the state of an interpreter.
type Context struct {
	scope         []*scope // The stack of call frames.
	stackDepth    int      // Current stack depth.
	maxStackDepth int      // Stack limit.
}

// NewContext returns a Context ready to execute. The argument specifies
// the maximum stack depth to allow, with <=0 meaning unlimited.
func NewContext(depth int) *Context {
	evalInit()
	c := &Context{}
	c.maxStackDepth = depth
	c.push(top, nil) // Global variables go in scope[0].
	vars := c.scope[0].vars
	vars[tokT] = constT
	vars[tokF] = constF
	vars[tokNil] = constNIL
	return c
}

// isCadR reports whether the string represents a run of car and cdr calls.
func isCadR(s string) bool {
	if len(s) < 3 || s[0] != 'c' || s[len(s)-1] != 'r' {
		return false
	}
	for _, c := range s[1 : len(s)-1] {
		if c != 'a' && c != 'd' {
			return false
		}
	}
	return true
}

// lookupElementary returns the function tied to an elementary, or nil.
func lookupElementary(name *token) elemFunc {
	if fn, ok := elementary[name]; ok {
		return fn
	}
	if isCadR(name.text) {
		return (*Context).cadrFunc
	}
	return nil
}

// push pushes an execution frame onto the stack.
func (c *Context) push(fn string, args *Expr) {
	c.scope = append(c.scope, &scope{
		vars: make(frame),
		fn:   fn,
		args: args,
	})
}

// pop pops one frame of the execution stack.
func (c *Context) pop() {
	c.scope[len(c.scope)-1] = nil // Do not hold on to old frames.
	c.scope = c.scope[:len(c.scope)-1]
}

// PopStack resets the execution stack.
func (c *Context) PopStack() {
	c.stackDepth = 0
	for len(c.scope) > 1 {
		c.pop()
	}
}

// StackTrace returns a printout of the execution stack.
// The most recent call appears first. Long stacks are trimmed
// in the middle.
func (c *Context) StackTrace() string {
	if c.scope[len(c.scope)-1].fn == top {
		return ""
	}
	var b strings.Builder
	fmt.Fprintln(&b, "stack:")
	for i := len(c.scope) - 1; i > 0; i-- {
		if len(c.scope)-i > 20 && i > 20 { // Skip the middle bits.
			i = 20
			fmt.Fprintln(&b, "\t...")
			continue
		}
		s := c.scope[i]
		if s.fn != top {
			fmt.Fprintf(&b, "\t(%s %s)\n", s.fn, Car(s.args))
		}
	}
	return b.String()
}

// getScope returns the scope in which the token is set.
// If it is not set, it returns the innermost (deepest) scope.
func (c *Context) getScope(tok *token) *scope {
	var sc *scope
	for _, s := range c.scope {
		if _, ok := s.vars[tok]; ok {
			sc = s
		}
	}
	if sc == nil {
		return c.scope[len(c.scope)-1]
	}
	return sc
}

// nonConst guarantees that tok is not a constant.
func notConst(tok *token) {
	if tok.typ == tokenConst {
		errorf("cannot set constant %s", tok)
	}
}

// set binds the atom (token) to the expression. If the atom is already
// bound anywhere on the stack, the innermost instance is rebound.
func (c *Context) set(tok *token, expr *Expr) {
	notConst(tok)
	c.getScope(tok).vars[tok] = expr
}

// set binds the atom (token) to the expression in the innermost scope.
func (c *Context) setLocal(tok *token, expr *Expr) {
	notConst(tok)
	c.scope[len(c.scope)-1].vars[tok] = expr
}

// returns the bound value of the token. The value of a number is itself.
func (c *Context) get(tok *token) *Expr {
	if tok.typ == tokenNumber {
		return atomExpr(tok)
	}
	return c.getScope(tok).vars[tok]
}

// getAtom returns the atom (token) represented by the expression, or nil if
// expr is not an atom.
func (e *Expr) getAtom() *token {
	if e != nil && e.atom != nil {
		return e.atom
	}
	return nil
}

const top = "<top>" // fn string used to identify the global, outermost scope.

// Eval returns value of the expression. The result depends on the expr:
// - for atoms, the value of the atom
// - for function definitions (defn ...), the list of defined functions
// - for general expressions, the value of executing apply[λ[;expr];nil],
// that is, a vacuous lambda with expr as its body and no arguments.
func (c *Context) Eval(expr *Expr) *Expr {
	// If the expression is an atom, print its value.
	if a := expr.getAtom(); a != nil {
		if lookupElementary(a) != nil {
			errorf("%s is elementary", a)
		}
		return c.get(a)
	}
	// Defn is very special.
	if a := Car(expr).getAtom(); a == tokDefn {
		return c.apply("defn", Car(expr), Cdr(expr))
	}
	// General expression, treat as a function invocation by
	// calling apply((lambda () expr), nil).
	lambda := Cons(atomExpr(tokLambda), Cons(nil, Cons(expr, nil)))
	return c.apply(top, lambda, nil)
}

// okToCall verifies the fn is defined and there is room on the stack.
func (c *Context) okToCall(name string, fn, x *Expr) {
	if fn == nil {
		errorf("undefined: %s", Cons(atomExpr(mkToken(tokenAtom, name)), x))
	}
	if c.maxStackDepth > 0 {
		c.stackDepth++
		if c.stackDepth > c.maxStackDepth {
			c.push(name, x) // Display this call at the top.
			errorf("stack too deep")
		}
	}
}

// apply applies fn to expr. The name is for debugging.
// This is on page 13 of the Lisp 1.5 book, but without the a-list.
// We do lexical scoping instead using c.push, c.set, etc.
func (c *Context) apply(name string, fn, x *Expr) *Expr {
	c.okToCall(name, fn, x)
	if fn.atom != nil {
		elem := lookupElementary(fn.atom)
		if elem != nil {
			return elem(c, fn.atom, x)
		}
		if fn.atom.typ != tokenAtom {
			errorf("%s is not a function", fn)
		}
		return c.apply(name, c.eval(fn), x)
	}
	if l := Car(fn).getAtom(); l == tokLambda || l == tokASCIILambda {
		args := x
		formals := Car(Cdr(fn))
		if args.length() != formals.length() {
			errorf("args mismatch for %s: %s %s", name, formals, args)
		}
		c.push(name, args)
		for args != nil {
			param := Car(formals)
			formals = Cdr(formals)
			atom := param.getAtom()
			if atom == nil {
				errorf("no atom")
			}
			c.setLocal(atom, Car(args))
			args = Cdr(args)
		}
		expr := c.eval(Car(Cdr(Cdr(fn))))
		c.pop()
		return expr
	}
	errorf("apply failed: %s", Cons(atomExpr(mkToken(tokenAtom, name)), x))
	return x
}

// eval evaluates the expression, as on page 13 of the Lisp 1.5 book.
func (c *Context) eval(e *Expr) *Expr {
	if e == nil {
		return nil
	}
	if atom := e.getAtom(); atom != nil {
		return c.get(atom)
	}
	if atom := Car(e).getAtom(); atom != nil {
		switch atom {
		case tokQuote:
			return Car(Cdr(e))
		case tokCond:
			return c.evcon(Cdr(e))
		}
		return c.apply(atom.text, Car(e), c.evlis(Cdr(e)))
	}
	errorf("cannot eval %s", e)
	return nil
}

// evcon evaluates a cond (sic) expression, as on page 13 of the Lisp 1.5 book.
func (c *Context) evcon(x *Expr) *Expr {
	if x == nil {
		errorf("no true case in cond")
	}
	if c.eval(Car(Car(x))).isTrue() {
		return c.eval(Car(Cdr(Car(x))))
	}
	return c.evcon(Cdr(x))
}

// evlis evaluates the list elementwise, as on page 13 of the Lisp 1.5 book.
func (c *Context) evlis(m *Expr) *Expr {
	if m == nil {
		return nil
	}
	return Cons(c.eval(Car(m)), c.evlis(Cdr(m)))
}

// Car implements the Lisp function CAR.
// Car and Cdr are functions not methods so (CADR X) is Car(Cdr(x)) not x.Cdr().Car().
func Car(e *Expr) *Expr {
	if e == nil || e.atom != nil {
		return nil
	}
	return e.car
}

// Cdr implements the Lisp function CDR.
func Cdr(e *Expr) *Expr {
	if e == nil || e.atom != nil {
		return nil
	}
	return e.cdr
}

// Cons implements the Lisp function CONS.
func Cons(car, cdr *Expr) *Expr {
	return &Expr{
		car: car,
		cdr: cdr,
	}
}

// isTrue reports whether the expression is the T atom.
func (e *Expr) isTrue() bool {
	return e != nil && e.atom == tokT
}

// length reports the number of items in the top level of the list.
// Used to check arguments match formals.
func (e *Expr) length() int {
	if e == nil {
		return 0
	}
	return 1 + Cdr(e).length()
}
