(defn(
	; Nice examples.
	(fac (lambda (n) (cond
		((eq n 0) 1)
		(T (mul n (fac (sub n 1))))
	)))
	(gcd (λ (x y) (cond  ; Yes, you can use λ. It's prettier.
		((gt x y) (gcd y x))
		((eq (rem y x) 0) x)
		(T (gcd (rem y x) x))
	)))
	(ack (λ (m n) (cond
		((eq m 0) (add n 1))
		((eq n 0) (ack (sub m 1) 1))
		(T (ack (sub m 1) (ack m (sub n 1))))
	)))
	(equal (λ (x y) (cond
		((eq x y) T)
		((atom x) F)
		((atom y) F)
		((equal (car x) (car y)) (equal (cdr x) (cdr y)))
		(T F)
	)))

	; Helpers.
	(not (λ (m) (cond
		(m F)
		(T T)
	)))
	(negate (λ (n) (sub 0 n)))
	(mapcar (λ (fn list) (cond
		((null list) nil)
		(T (cons (fn (car list)) (mapcar fn (cdr list))))
	)))
	(length (λ (l) (cond
		((null l) 0)
		(T (add 1 (length (cdr l))))
	)))

	; Demo of building a function.
	(opN (λ (op N) (list 'λ '(a) (list op N 'a))))

	; From the book.
	(member (λ (x list) (cond
		((null list) F)
		((equal x (car list)) T)
		(T (member x (cdr list)))
	)))
	(union (λ (x y) (cond
		((null x) y)
		((member (car x) y) (union (cdr x) y))
		(T (cons (car x) (union (cdr x) y)))
	)))
	(intersection (λ (x y) (cond
		((null x) nil)
		((member (car x) y) (cons (car x) (intersection (cdr x) y)))
		(T (intersection (cdr x) y))
	)))
))
