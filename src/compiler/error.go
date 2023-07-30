package compiler

import (
	"fmt"
	"path/filepath"
	"runtime"
	"runtime/debug"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
)

// helper that might be extended later
// it is not intended for the end user to see these errors, as they are compiler bugs
// the errors in the ddp-code were already reported by the parser/typechecker/resolver
func (c *compiler) err(msg string, args ...any) {
	// retreive the file and line on which the error occured
	_, file, line, _ := runtime.Caller(1)
	stack_trace := debug.Stack()
	mod_path := "not found"
	if c.ddpModule != nil {
		mod_path = c.ddpModule.FileName
	}
	panic(&CompilerError{
		Err:        nil,
		Msg:        fmt.Sprintf("(%s, %d) %s", filepath.Base(file), line, fmt.Sprintf(msg, args...)),
		ModulePath: mod_path,
		Node:       c.currentNode,
		StackTrace: stack_trace,
	})
}

func compiler_panic_wrapper(c *compiler) {
	if err := recover(); err != nil {
		if err, ok := err.(*CompilerError); ok {
			panic(err)
		}

		stack_trace := debug.Stack()
		mod_path := "not found"
		if c.ddpModule != nil {
			mod_path = c.ddpModule.FileName
		}
		panic(&CompilerError{
			Err:        err,
			Msg:        "unknown compiler panic",
			ModulePath: mod_path,
			Node:       c.currentNode,
			StackTrace: stack_trace,
		})
	}
}

type CompilerError struct {
	Err        any      // additional data (probably an error, but maybe a string or something from a library), maybe nil
	Msg        string   // an additional message describing the error
	ModulePath string   // the module that was compiled when this error occured
	Node       ast.Node // the node where the error occurred or nil
	StackTrace []byte   // a stack trace
}

func (err *CompilerError) Error() string {
	rng := ""
	if err.Node != nil {
		rng = err.Node.GetRange().String()
	}
	return fmt.Sprintf("CompilerError(Mod: %s, Node: %v, Range: %s): %s\nWraps: %v\nStackTrace:\n%s", err.ModulePath, err.Node, rng, err.Msg, err.Err, string(err.StackTrace))
}

func (err *CompilerError) Unwrap() error {
	wrapped, ok := err.Err.(error)
	if ok {
		return wrapped
	}
	return nil
}
