package compiler

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/DDP-Projekt/Kompilierer/pkg/ast"
	"github.com/DDP-Projekt/Kompilierer/pkg/ddperror"
	"github.com/DDP-Projekt/Kompilierer/pkg/parser"
)

type OutputType int

const (
	OutputIR OutputType = iota
	OutputAsm
	OutputObj
)

// Options on how to compile the given source code
// and where to get it from
type Options struct {
	// Optional Filename to name the source
	// this file is read if Source and From are nil
	FileName string
	// Optional ddp-source-code
	// if nil, From is used next
	Source []byte
	// Optional reader to read the source code from
	// if Source is nil
	From io.Reader
	// Optional Writer to write the compiler output to
	// must be non-nil if the output is a object file
	To io.Writer
	// type of the output
	// OutputObj should be written to To
	// OutputAsm/IR may be returned in a *Result
	OutputType OutputType
	// ErrorHandler used for the scanner, parser, ...
	// May be nil
	ErrorHandler ddperror.Handler
	// optional Log function to print intermediate messages
	Log func(string, ...any)
	// wether or not to delete intermediate .ll files
	DeleteIntermediateFiles bool
}

func (options *Options) ToParserOptions() parser.Options {
	return parser.Options{
		FileName:     options.FileName,
		Source:       options.Source,
		Tokens:       nil,
		ErrorHandler: options.ErrorHandler,
	}
}

// the result of a compilation
type Result struct {
	// a set which contains all files needed
	// to link the final executable
	// contains .c, .lib, .a and .o files
	Dependencies map[string]struct{}
	// the llvm ir or assembly
	// empty if the output was written to a io.Writer
	Output string
}

func validateOptions(options *Options) error {
	if options.Source == nil && options.From == nil && options.FileName == "" {
		return errors.New("Kein Quellcode gegeben")
	}
	if options.ErrorHandler == nil {
		options.ErrorHandler = ddperror.EmptyHandler
	}
	if options.Log == nil {
		options.Log = func(string, ...any) {}
	}
	return nil
}

// compile ddp-source-code from the given Options
// if an error occured, the result is nil
// if the given source code is not valid ddp-code
// an error with an empty message is returned
func Compile(options Options) (*Result, error) {
	// validate the options
	err := validateOptions(&options)
	if err != nil {
		return nil, err
	}

	// compile the ddp-source into an Ast
	var Ast *ast.Ast
	options.Log("Parse DDP Quellcode")
	if options.Source == nil && options.From != nil {
		options.Source, err = io.ReadAll(options.From)
		if err != nil {
			return nil, err
		}
	}
	Ast, err = parser.Parse(options.ToParserOptions())

	if err != nil {
		return nil, err
	}

	// if set, only compile to llvm ir and return
	options.Log("Kompiliere den Abstrakten Syntaxbaum zu LLVM ir")
	if options.OutputType == OutputIR {
		return New(Ast, options.ErrorHandler).Compile(options.To)
	}

	// wrtite the llvm ir to an intermediate file
	tempFileName := "ddp_temp.ll"
	if options.FileName != "" {
		tempFileName = changeExtension(options.FileName, ".ll")
	}
	file, err := os.OpenFile(tempFileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return nil, err
	}
	if options.DeleteIntermediateFiles {
		defer options.Log("Lösche die temporäre Datei '%s'", tempFileName)
		defer os.Remove(tempFileName)
	}

	options.Log("Schreibe LLVM ir nach %s", tempFileName)
	result, err := New(Ast, options.ErrorHandler).Compile(file)
	file.Close()
	if err != nil {
		return nil, err
	}

	// object files are written to the given io.Writer (most commonly a file)
	options.Log("Kompiliere LLVM ir zu einer Objektdatei")
	if options.OutputType == OutputObj {
		_, err = compileToObject(tempFileName, options.OutputType, options.To)
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	// if options.To is nil, the assembly code is returned as string in result
	if options.To == nil {
		out := &strings.Builder{}
		_, err = compileToObject(tempFileName, options.OutputType, out)
		if err != nil {
			return nil, err
		}
		result.Output = out.String()
		return result, nil
	}

	// otherwise the assembly code is written to options.To
	_, err = compileToObject(tempFileName, options.OutputType, options.To)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func changeExtension(path, ext string) string {
	return path[:len(path)-len(filepath.Ext(path))] + ext
}
