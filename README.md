# Lisp 1.5

This is an implementation of the language defined, with sublime concision, in the first few pages of the LISP 1.5 Programmer's Manual by McCarthy, Abrahams, Edwards, Hart, and Levin, from MIT in 1962.

It is a pedagogical experiment to see just how well the interpreter (actually `EVALQUOTE/APPLY`) defined on page 13 of that book really works. The answer is: perfectly, of course.

This program was a joy to put together. Its purpose was fun and education, and in no way to create a modern or even realistic Lisp implementation. The goal was to turn that marvelous page 13 into a working interpreter using clean, direct Go code.

The program therefore has several profound shortcomings, even with respect to the Lisp 1.5 book:

- No `SET` or `SETQ`.
- No `PROG`. The intepreter, by calling `APPLY`, can evaluate only a single expression, a possibly recursive function invocation. But this is Lisp, and that's still a lot.
- No character handling.
- No I/O. Interactive only, although it can start by reading a file specified on the command line.

It is slow and of course the language is very, _very_ far from Common Lisp or Scheme.

Although I may add `PROG` and perhaps `SET[Q]` one day, that's about where I'd draw the line. I plan to use this as a teaching tool, not a practical programming environment.

It does do one important thing differently from the book, though: lexical scoping. There is no association list; instead a function has access only to the locals in its own frame, as well globals like `T`.

### Use

A few details about the interpreter.

A semicolon introduces a comment that extends to newline.

For convenience, `'A` is the familiar shorthand for `(QUOTE A)`

`T` and `F` are upper case, but all the other words (`car`, `nil`, and such) are lower case.

Numbers are implemented by Go's `big.Int`, so there is no floating point but numbers can be big.

Identifiers can be Unicode. Just for fun, `λ` is a synonym for `lambda`. (It's really the other way around, isn't it?)

Identifiers must be alphanumeric. The addition function is `add` not `+`.

### Built-in functions.

I never liked to type `DIFFERENCE` or `QUOTIENT`, so arithmetic uses the much shorter `add` `sub` `mul` `div` `rem`, and the comparision operators come from Fortran (why not?): `eq` `ne` `lt` `le` `gt` `ge`, as well as `and` and `or`.

Other builtin functions are: `apply` `atom`, `car`, `cdr`, `cond`, `cons`, `list`, `null`, and `quote`.

Function definition is done with the `defn` builtin:

	(defn (
		(add2 (λ (n) (add 2 n)))
		(add4 (λ (n) (add2 (add2 n))))
	))


### An example session.

Here is a typescript. There is a library in `lib.lisp`; passing it as an argument causes `lisp` to load it before reading standard input.

	% lisp lib.lisp
	(fac gcd ack equal not negate mapcar length opN member union intersection)
	> ; Funcs
	> (add 1 3)
	4
	> ; Define a lambda that adds 2.
	> (defn ((add2 (lambda (n) (add 2 n))) ))
	(add2)
	> (add2 3)
	5
	> ; Or add 1 twice.
	> (defn ((add2 (lambda (n) (add 1 (add 1 n)))) ))
	(add2)
	> (add2 3)
	5
	; There's a startup library in lisp.lib. Let's look at fac. It is recursive:
	> fac
	(lambda (n) (cond ((eq n 0) 1) (T (mul n (fac (sub n 1))))))
	> ; Breaking it down:
	> (car fac)
	lambda
	> (cadr fac)
	(n)
	> (caddr fac)
	(cond ((eq n 0) 1) (T (mul n (fac (sub n 1)))))
	> ; Let's run it.
	> (fac 1)
	1
	> (fac 10)
	3628800
	> ; We have big integers.
	> (fac 100)
	93326215443944152681699238856266700490715968264381621468592963895217599993229915608941463976156518286253697920827223758251185210916864000000000000000000000000
	> ; Equal in Lisp.
	> equal
	(lambda (x y) (cond ((eq x y) T) ((atom x) F) ((atom y) F) ((equal (car x) (car y)) (equal (cdr x) (cdr y))) (T F)))
	> (equal '(1 2 (3)) '(1 2 (3)))
	T
	> (equal '(1 2 (3)) '(1 2 (4)))
	F
	> ; Mapcar, a Lisp staple: Apply function to each list element.
	> mapcar
	(lambda (fn list) (cond ((null list) nil) (T (cons (fn (car list)) (mapcar fn (cdr list))))))
	> (mapcar fac '(1 2 3 4 5 6 7 8 9 10))
	(1 2 6 24 120 720 5040 40320 362880 3628800)
	> ; Using a lambda directly.
	> (mapcar '(lambda (n) (add 1 (add 1 n))) '(2 3 4))
	(4 5 6)
	> ; Let's build a function.
	> ; A helper: list makes it easier than using cons to build long lists. (Variadic)
	> (list 'a 'b '(c))
	(a b (c))
	> ; Easier than
	> (cons 'a (cons 'b (cons (cons 'c nil) nil)))
	(a b (c))
	> ; Remember how we built add2 above:
	> (defn ((add2 (lambda (n) (add 2 n))) ))
	(add2)
	> ; Now build a function that adds arbitrary N to its argument.
	> (defn( (addN (lambda (N) (list 'lambda '(a) (list 'add N 'a)))) ))
	(addN)
	> (addN 5)
	(lambda (a) (add 5 a))
	> ((addN 5) 3)
	cannot eval ((addN 5) 3)
	> ; A Lisp 1.5-ism: apply vs. eval.
	> (apply (addN 5) 3)
	8
	> (mapcar (addN 5) '(1 2 3 4))
	(6 7 8 9)
	> ; Why not program the op too?
	> (defn( (opN (lambda (op N) (list 'lambda '(a) (list op N 'a)))) ))
	(opN)
	> (mapcar (opN 'sub 5) '(1 2 3 4))
	(4 3 2 1)
	> ; One last piece of interest: The Ackermann function.
	> ; The first argument is a kind of level: constant, n, 2n, 2^n 2^2^n etc.
	> ack
	(lambda (m n) (cond ((eq m 0) (add n 1)) ((eq n 0) (ack (sub m 1) 1)) (T (ack (sub m 1) (ack m (sub n 1))))))
	> (ack 3 4) ; apply called 51535 times!
	125
	> ^D
	%
