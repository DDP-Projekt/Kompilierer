package compiler

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/parser"
	"github.com/bafto/Go-LLVM-Bindings/llvm"
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
	// writer where the result is written to
	// must be non-nil
	To io.Writer
	// type of the output
	OutputType OutputType
	// ErrorHandler used for the scanner, parser, ...
	// May be nil
	ErrorHandler ddperror.Handler
	// optional Log function to print intermediate messages
	Log func(string, ...any)
	// wether or not to delete intermediate .ll files
	DeleteIntermediateFiles bool
	// wether all compiled DDP Modules
	// should be linked into a single llvm-ir-Module
	LinkInModules bool
	/*// wether the generated code for lists should be
	// linked into the main module
	LinkLists bool*/
}

func (options *Options) ToParserOptions() parser.Options {
	return parser.Options{
		FileName:     options.FileName,
		Source:       options.Source,
		Tokens:       nil,
		Modules:      nil,
		ErrorHandler: options.ErrorHandler,
	}
}

// the result of a compilation
type Result struct {
	// a set which contains all files needed
	// to link the final executable
	// contains .c, .lib, .a and .o files
	Dependencies map[string]struct{}
}

func validateOptions(options *Options) error {
	if options.Source == nil && options.From == nil && options.FileName == "" {
		return errors.New("Kein Quellcode gegeben")
	}
	if options.To == nil {
		return errors.New("Kein Ziel io.Writer gegeben")
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
//
// !!! Warning !!!
// any temporary files created by this function will
// share their name with the corresponding .ddp file
// they were parsed from, but with a different
// file extension (.ll, .o, .asm, etc.)
// so be careful with file descriptors pointing to
// such files
func Compile(options Options) (*Result, error) {
	// validate the options
	err := validateOptions(&options)
	if err != nil {
		return nil, err
	}

	// compile the ddp-source into an Ast
	options.Log("Parse DDP Quellcode")
	if options.Source == nil && options.From != nil {
		options.Source, err = io.ReadAll(options.From)
		if err != nil {
			return nil, err
		}
	}

	ddp_main_module, err := parser.Parse(options.ToParserOptions())
	if err != nil {
		return nil, err
	}

	options.Log("Kompiliere den Abstrakten Syntaxbaum zu LLVM ir")

	if !options.LinkInModules {
		switch options.OutputType {
		case OutputIR:
			return newCompiler(ddp_main_module, options.ErrorHandler).compile(options.To, true)
		case OutputAsm, OutputObj:
			ll_path := changeExtension(ddp_main_module.FileName, ".ll")
			options.Log("Erstelle temporäre Datei '%s'", ll_path)
			temp_file, err := os.OpenFile(ll_path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.ModePerm)
			if err != nil {
				return nil, err
			}
			if options.DeleteIntermediateFiles {
				defer options.Log("Lösche temporäre Datei '%s'", ll_path)
				defer os.Remove(ll_path)
			}

			comp_result, err := newCompiler(ddp_main_module, options.ErrorHandler).compile(temp_file, true)
			if err != nil {
				return nil, err
			}

			options.Log("Erstelle llvm Context")
			llctx, err := newllvmContext()
			if err != nil {
				return nil, err
			}
			defer options.Log("Entsorge llvm Context")
			defer llctx.Dispose()

			options.Log("Parse llvm-ir zu llvm-Module")
			mod, ctx, err := llctx.parseLLFile(temp_file.Name())
			if err != nil {
				return nil, err
			}
			defer mod.Dispose()
			defer ctx.Dispose()

			file_type := llvm.AssemblyFile
			if options.OutputType == OutputObj {
				file_type = llvm.ObjectFile
				options.Log("Kompiliere llvm-ir zu Object Code")
			} else {
				options.Log("Kompiliere llvm-ir zu Assembler")
			}

			if _, err := llctx.compileModule(mod, file_type, options.To); err != nil {
				return nil, err
			}

			return comp_result, nil
		default:
			return nil, errors.New("invalid compiler.OutputType")
		}
	}
	// options.LinkInModules == true

	options.Log("Erstelle llvm Context")
	llctx, err := newllvmContext()
	if err != nil {
		return nil, err
	}
	defer options.Log("Entsorge llvm Context")
	defer llctx.Dispose()

	ll_path := changeExtension(ddp_main_module.FileName, ".ll")
	options.Log("Erstelle temporäre Datei '%s'", ll_path)
	temp_file, err := os.OpenFile(ll_path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return nil, err
	}

	output_paths, dependencies, err := compileWithImports(ddp_main_module, temp_file, options.ErrorHandler)
	if err != nil {
		return nil, err
	}

	ll_modules := map[string]llvm.Module{}
	for path, _ := range output_paths {
		options.Log("Parse '%s' zu llvm-Module", path)
		llmod, ctx, err := llctx.parseLLFile(path)
		if err != nil {
			return nil, err
		}
		defer llmod.Dispose()
		defer ctx.Dispose()
		ll_modules[path] = llmod

		if options.DeleteIntermediateFiles {
			defer options.Log("Lösche temporäre Datei '%s'", path)
			defer os.Remove(path)
		}
	}
	ll_main_module := ll_modules[ll_path]
	delete(ll_modules, ll_path)

	options.Log("Linke llvm Module")
	if err := llvmLinkAllModules(ll_main_module, mapToSlice(ll_modules)); err != nil {
		return nil, err
	}

	if _, err := io.WriteString(options.To, ll_main_module.String()); err != nil {
		return nil, err
	}

	// if we output llvm ir we are finished here
	if options.OutputType == OutputIR {
		return &Result{Dependencies: dependencies}, nil
	}

	file_type := llvm.AssemblyFile
	if options.OutputType == OutputObj {
		file_type = llvm.ObjectFile
		options.Log("Kompiliere llvm-ir zu Object Code")
	} else {
		options.Log("Kompiliere llvm-ir zu Assembler")
	}

	if _, err := llctx.compileModule(ll_main_module, file_type, options.To); err != nil {
		return nil, err
	}

	return &Result{Dependencies: dependencies}, nil
}

func changeExtension(path, ext string) string {
	return path[:len(path)-len(filepath.Ext(path))] + ext
}

func mapToSlice[T comparable, U any](m map[T]U) []U {
	result := make([]U, 0, len(m))
	for _, v := range m {
		result = append(result, v)
	}
	return result
}
