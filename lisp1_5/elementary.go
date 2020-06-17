// Copyright 2020 Rob Pike. All rights reserved.
// Use of this source code is governed by a BSD
// license that can be found in the LICENSE file.

// This file contains the definitions of the non-math elementary
// (builtin) functions.

package lisp1_5

func evalInit() {
	if elementary == nil {
		// Initialized here to avoid initialization loop.
		elementary = funcMap{
			tokAdd:   (*Context).addFunc,
			tokAnd:   (*Context).andFunc,
			tokApply: (*Context).applyFunc,
			tokAtom:  (*Context).atomFunc,
			tokCar:   (*Context).carFunc,
			tokCdr:   (*Context).cdrFunc,
			tokCons:  (*Context).consFunc,
			tokDefn:  (*Context).defnFunc,
			tokDiv:   (*Context).divFunc,
			tokEq:    (*Context).eqFunc,
			tokGe:    (*Context).geFunc,
			tokGt:    (*Context).gtFunc,
			tokLe:    (*Context).leFunc,
			tokList:  (*Context).listFunc,
			tokLt:    (*Context).ltFunc,
			tokMul:   (*Context).mulFunc,
			tokNe:    (*Context).neFunc,
			tokNull:  (*Context).nullFunc,
			tokOr:    (*Context).orFunc,
			tokRem:   (*Context).remFunc,
			tokSub:   (*Context).subFunc,
		}
	}
	constT = atomExpr(tokT)
	constF = atomExpr(tokF)
	constNIL = atomExpr(tokNil)
}

func (c *Context) applyFunc(name *token, expr *Expr) *Expr {
	return c.apply("applyFunc", Car(expr), Cdr(expr))
}

func (c *Context) defnFunc(name *token, expr *Expr) *Expr {
	var names []*Expr
	for expr = Car(expr); expr != nil; expr = Cdr(expr) {
		fn := Car(expr)
		if fn == nil {
			errorf("empty function in defn")
		}
		name := Car(fn)
		atom := name.getAtom()
		if atom == nil {
			errorf("malformed defn")
		}
		names = append(names, name)
		c.set(atom, Car(Cdr(fn)))
	}
	var result *Expr
	for i := len(names) - 1; i >= 0; i-- {
		result = Cons(names[i], result)
	}
	return result
}

func (c *Context) atomFunc(name *token, expr *Expr) *Expr {
	atom := Car(expr)
	return truthExpr(atom != nil && atom.atom != nil)
}

func (c *Context) carFunc(name *token, expr *Expr) *Expr {
	return Car(Car(expr))
}

func (c *Context) cdrFunc(name *token, expr *Expr) *Expr {
	return Cdr(Car(expr))
}

func (c *Context) cadrFunc(name *token, expr *Expr) *Expr {
	str := name.text
	if !isCadR(str) {
		return nil
	}
	expr = Car(expr)
	for i := len(str) - 2; expr != nil && i > 0; i-- {
		if str[i] == 'a' {
			expr = Car(expr)
		} else {
			expr = Cdr(expr)
		}
	}
	return expr
}

func (c *Context) consFunc(name *token, expr *Expr) *Expr {
	return Cons(Car(expr), Car(Cdr(expr)))
}

func (c *Context) eqFunc(name *token, expr *Expr) *Expr {
	a := Car(expr)
	b := Car(Cdr((expr)))
	return truthExpr(eq(a, b))
}

func eq(a, b *Expr) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	if a.atom == nil || b.atom == nil || a.atom.typ != b.atom.typ {
		return false
	}
	if a.atom.typ == tokenNumber {
		return a.atom.num.Cmp(b.atom.num) == 0
	}
	return a.atom == b.atom
}

func (c *Context) listFunc(name *token, expr *Expr) *Expr {
	if expr == nil {
		return nil
	}
	return Cons(Car(expr), Cdr(expr))
}

func (c *Context) nullFunc(name *token, expr *Expr) *Expr {
	return truthExpr(Car(expr) == nil)
}

func atomExpr(tok *token) *Expr {
	return &Expr{
		atom: tok,
	}
}

// truthExpr converts the boolean argument to the constant atom T or F.
func truthExpr(t bool) *Expr {
	if t {
		return constT
	}
	return constF
}

func (e *Expr) isNumber() bool {
	return e != nil && e.atom != nil && e.atom.typ == tokenNumber
}
