package parser

import (
	"fmt"
	"path/filepath"
	"runtime"
	"runtime/debug"
)

func collectLocationInfo(p *parser) string {
	if p == nil {
		return "no location info available"
	}

	mod_path := "not found"
	if p.module != nil {
		mod_path = p.module.FileName
	}
	tok := p.previous()
	return fmt.Sprintf("Mod: %s, Tok: %s", mod_path, tok.StringVerbose())
}

// helper that might be extended later
// it is not intended for the end user to see these errors, as they are compiler bugs
// the errors in the ddp-code were already reported by the parser/typechecker/resolver
func (p *parser) panic(msg string, args ...any) {
	// retreive the file and line on which the error occured
	_, file, line, _ := runtime.Caller(1)
	stack_trace := debug.Stack()
	locationInfo := collectLocationInfo(p)
	panic(&ParserError{
		Err:          nil,
		Msg:          fmt.Sprintf("(%s, %d) %s", filepath.Base(file), line, fmt.Sprintf(msg, args...)),
		LocationInfo: locationInfo,
		StackTrace:   stack_trace,
	})
}

func parser_panic_wrapper(p *parser) {
	if err := recover(); err != nil {
		if err, ok := err.(*ParserError); ok {
			panic(err)
		}

		stack_trace := debug.Stack()
		locationInfo := collectLocationInfo(p)
		err, _ := err.(error)
		panic(&ParserError{
			Err:          err,
			Msg:          "unknown parser panic",
			LocationInfo: locationInfo,
			StackTrace:   stack_trace,
		})
	}
}

type ParserError struct {
	Err          error  // the error being wrapped (maybe nil)
	Msg          string // an additional message describing the error
	LocationInfo string // the module that was parsed when this error occured
	StackTrace   []byte // a stack trace
}

func (err *ParserError) Error() string {
	return fmt.Sprintf("ParserError(%s): %s\nWraps: %v\nStackTrace:\n%s", err.LocationInfo, err.Msg, err.Err, string(err.StackTrace))
}

func (err *ParserError) Unwrap() error {
	return err.Err
}
