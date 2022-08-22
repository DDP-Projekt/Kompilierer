package compiler

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/DDP-Projekt/Kompilierer/pkg/ast"
	"github.com/DDP-Projekt/Kompilierer/pkg/parser"
	"github.com/DDP-Projekt/Kompilierer/pkg/scanner"
	"github.com/DDP-Projekt/Kompilierer/pkg/token"
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
	ErrorHandler scanner.ErrorHandler
	// wether or not to delete intermediate .ll files
	DeleteIntermediateFiles bool
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
		return errors.New("no source given in options")
	}
	if options.ErrorHandler == nil {
		options.ErrorHandler = func(token.Token, string) {}
	}
	return nil
}

// compile ddp-source-code from the given Options
// if an error occured, the result is nil
func Compile(options Options) (*Result, error) {
	// validate the options
	err := validateOptions(&options)
	if err != nil {
		return nil, err
	}

	// compile the ddp-source into an Ast
	var Ast *ast.Ast
	if options.Source != nil {
		Ast, err = parser.ParseSource(options.FileName, options.Source, options.ErrorHandler)
	} else if options.From != nil {
		src, err := io.ReadAll(options.From)
		if err != nil {
			return nil, err
		}
		Ast, err = parser.ParseSource(options.FileName, src, options.ErrorHandler)
	} else {
		Ast, err = parser.ParseFile(options.FileName, options.ErrorHandler)
	}

	if err != nil {
		return nil, err
	}

	// if set, only compile to llvm ir and return
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
	result, err := New(Ast, options.ErrorHandler).Compile(file)
	if err != nil {
		file.Close()
		return nil, err
	}
	file.Close()

	if options.DeleteIntermediateFiles {
		defer os.Remove(tempFileName)
	}

	// object files are written to the given io.Writer (most commonly a file)
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
