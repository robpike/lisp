// Copyright 2020 Rob Pike. All rights reserved.
// Use of this source code is governed by a BSD
// license that can be found in the LICENSE file.

// This file contains the definitions of the math elementary
// (builtin) functions.

package lisp1_5

import (
	"math/big"
)

// Arithmetic.

func (c *Context) mathFunc(expr *Expr, fn func(*big.Int, *big.Int) *big.Int) *Expr {
	return atomExpr(number(fn(c.getNumber(Car(expr)), c.getNumber(Car(Cdr(expr))))))
}

func (c *Context) getNumber(expr *Expr) *big.Int {
	if !expr.isNumber() {
		errorf("expect number; have %s", expr)
	}
	return expr.atom.num
}

func add(a, b *big.Int) *big.Int { return new(big.Int).Add(a, b) }
func div(a, b *big.Int) *big.Int {
	if b.Cmp(&zero) == 0 {
		errorf("division by zero")
	}
	return new(big.Int).Div(a, b)
}
func mul(a, b *big.Int) *big.Int { return new(big.Int).Mul(a, b) }
func rem(a, b *big.Int) *big.Int {
	if b.Cmp(&zero) == 0 {
		errorf("rem by zero")
	}
	return new(big.Int).Rem(a, b)
}
func sub(a, b *big.Int) *big.Int { return new(big.Int).Sub(a, b) }

func (c *Context) addFunc(name *token, expr *Expr) *Expr { return c.mathFunc(expr, add) }
func (c *Context) divFunc(name *token, expr *Expr) *Expr { return c.mathFunc(expr, div) }
func (c *Context) mulFunc(name *token, expr *Expr) *Expr { return c.mathFunc(expr, mul) }
func (c *Context) remFunc(name *token, expr *Expr) *Expr { return c.mathFunc(expr, rem) }
func (c *Context) subFunc(name *token, expr *Expr) *Expr { return c.mathFunc(expr, sub) }

// Comparison.

func (c *Context) boolFunc(expr *Expr, fn func(*big.Int, *big.Int) bool) *Expr {
	return truthExpr(fn(c.getNumber(Car(expr)), c.getNumber(Car(Cdr(expr)))))
}

func ge(a, b *big.Int) bool { return a.Cmp(b) >= 0 }
func gt(a, b *big.Int) bool { return a.Cmp(b) > 0 }
func le(a, b *big.Int) bool { return a.Cmp(b) <= 0 }
func lt(a, b *big.Int) bool { return a.Cmp(b) < 0 }
func ne(a, b *big.Int) bool { return a.Cmp(b) != 0 }

func (c *Context) geFunc(name *token, expr *Expr) *Expr { return c.boolFunc(expr, ge) }
func (c *Context) gtFunc(name *token, expr *Expr) *Expr { return c.boolFunc(expr, gt) }
func (c *Context) leFunc(name *token, expr *Expr) *Expr { return c.boolFunc(expr, le) }
func (c *Context) ltFunc(name *token, expr *Expr) *Expr { return c.boolFunc(expr, lt) }
func (c *Context) neFunc(name *token, expr *Expr) *Expr { return c.boolFunc(expr, ne) }

// Logic. These are implemented here because they are variadic.

func (c *Context) andFunc(name *token, expr *Expr) *Expr {
	if expr == nil {
		return truthExpr(true)
	}
	if !Car(expr).isTrue() {
		return truthExpr(false)
	}
	return c.andFunc(name, Cdr(expr))
}

func (c *Context) orFunc(name *token, expr *Expr) *Expr {
	if expr == nil {
		return truthExpr(false)
	}
	if Car(expr).isTrue() {
		return truthExpr(true)
	}
	return c.orFunc(name, Cdr(expr))
}
