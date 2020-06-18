// Copyright 2020 Rob Pike. All rights reserved.
// Use of this source code is governed by a BSD
// license that can be found in the LICENSE file.

// Lisp is an implementation of the language defined, with sublime concision, in
// the first few pages of the LISP 1.5 Programmer's Manual by McCarthy, Abrahams,
// Edwards, Hart, and Levin, from MIT in 1962.
//
// It is a pedagogical experiment to see just how well the interpreter (actually
// EVALQUOTE/APPLY) defined on page 13 of that book really works. The answer is:
// perfectly, of course.
//
// This program's purpose was fun and education, and in no way to create a modern
// or even realistic Lisp implementation. The goal was to turn that marvelous page
// 13 into a working interpreter using clean, direct Go code.
//
// The program therefore has several profound shortcomings, even with respect to
// the Lisp 1.5 book:
//
//  - No `SET` or `SETQ`.
//  - No `PROG`. The interpreter, by calling `APPLY`, can evaluate only a single
//    expression, a possibly recursive function invocation. But this is Lisp, and
//    that's still a lot.
//  - No character handling.
//  - No I/O. Interactive only, although it can start by reading a file specified
//    on the command line.
//
// It is slow and of course the language is very, very far from Common Lisp or
// Scheme.
package main // import "github.com/robpike/lisp"

import (
	"bufio"
	"flag"
	"fmt"
	"os"

	"github.com/robpike/lisp/lisp1_5"
)

var (
	printSExpr = flag.Bool("sexpr", false, "always print S-expressions")
	doPrompt   = flag.Bool("doprompt", true, "show interactive prompt")
	prompt     = flag.String("prompt", "> ", "interactive prompt")
	stackDepth = flag.Int("depth", 1e5, "maximum call depth; 0 means no limit")
)

var loading bool

func main() {
	flag.Parse()
	lisp1_5.Config(*printSExpr)
	context := lisp1_5.NewContext(*stackDepth)
	loading = true
	for _, file := range flag.Args() {
		load(context, file)
	}
	loading = false
	parser := lisp1_5.NewParser(bufio.NewReader(os.Stdin))
	for {
		input(context, parser, *prompt)
	}
}

// load reads the named source file and parses it within the context.
func load(context *lisp1_5.Context, file string) {
	fd, err := os.Open(file)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer fd.Close()
	parser := lisp1_5.NewParser(bufio.NewReader(fd))
	input(context, parser, "")
}

// input runs the parser to EOF.
func input(context *lisp1_5.Context, parser *lisp1_5.Parser, prompt string) {
	defer handler(context, parser)
	for {
		if prompt != "" && *doPrompt {
			fmt.Print(prompt)
		}
		switch parser.SkipSpace() {
		case '\n':
			continue
		case lisp1_5.EofRune:
			if !loading {
				os.Exit(0)
			}
			return
		}
		expr := context.Eval(parser.List())
		fmt.Println(expr)
		parser.SkipSpace() // Grab the newline.
	}
}

// handler handles panics from the interpreter. These are part
// of normal operation, signaling parsing and execution errors.
func handler(context *lisp1_5.Context, parser *lisp1_5.Parser) {
	e := recover()
	if e != nil {
		switch e := e.(type) {
		case lisp1_5.EOF:
			os.Exit(0)
		case lisp1_5.Error:
			fmt.Fprintln(os.Stderr, e)
			parser.SkipToEndOfLine()
			fmt.Fprint(os.Stderr, context.StackTrace())
			context.PopStack()
		default:
			panic(e)
		}
	}
}
