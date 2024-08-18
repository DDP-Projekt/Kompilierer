package parser

import (
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/token"
)

type errorReporter struct {
	// a function to which errors are passed
	errorHandler ddperror.Handler
	// latest reported error
	lastError ddperror.Error
	// flag to not report following errors
	panicMode bool
	// wether an error was reported
	errored bool
	// filename of the reported errors
	fileName string
}

// wrap the errorHandler to set the reporters Errored variable
// if it is called
func (e *errorReporter) initializeReporter() {
	e.errorHandler = func(err ddperror.Error) {
		if err.Level == ddperror.LEVEL_ERROR {
			e.errored = true
		}
		e.errorHandler(err)
	}
}

func (e *errorReporter) errVal(err ddperror.Error) {
	if !e.panicMode {
		e.panicMode = true
		e.lastError = err
		e.errorHandler(e.lastError)
	}
}

// helper to report errors and enter panic mode
func (e *errorReporter) err(code ddperror.Code, Range token.Range, msg string) {
	e.errVal(ddperror.New(code, ddperror.LEVEL_ERROR, Range, msg, e.fileName))
}

// helper to report errors and enter panic mode
func (e *errorReporter) warn(code ddperror.Code, Range token.Range, msg string) {
	e.errorHandler(ddperror.New(code, ddperror.LEVEL_WARN, Range, msg, e.fileName))
}
