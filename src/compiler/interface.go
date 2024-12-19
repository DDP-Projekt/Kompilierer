package compiler

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"runtime/debug"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ast/annotators"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/ddppath"
	"github.com/DDP-Projekt/Kompilierer/src/parser"
	"github.com/bafto/Go-LLVM-Bindings/llvm"
)

type OutputType int

const (
	OutputIR  OutputType = iota // textual llvm ir
	OutputBC                    // llvm bitcode, currently unused
	OutputAsm                   // assembly depending on the target platform
	OutputObj                   // object file depending on the target platform
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
	// wether to use the old or new alias parsing
	StrictAliases bool
	// optional Log function to print intermediate messages
	Log func(string, ...any)
	// wether or not to delete intermediate .ll files
	DeleteIntermediateFiles bool
	// wether all compiled DDP Modules
	// should be linked into a single llvm-ir-Module
	LinkInModules bool
	// wether the generated code for lists should be
	// linked into the main module
	LinkInListDefs bool
	// level of optimizations
	//	-  0: no optimizations
	//	-  1: only LLVM optimizations
	//	- >2: all optimizations
	OptimizationLevel uint
}

func (options *Options) ToParserOptions() parser.Options {
	var annos []ast.Annotator
	if options.OptimizationLevel >= 2 {
		annos = append(annos, &annotators.ConstFuncParamAnnotator{})
	}
	return parser.Options{
		FileName:      options.FileName,
		Source:        options.Source,
		Tokens:        nil,
		Modules:       nil,
		ErrorHandler:  options.ErrorHandler,
		StrictAliases: options.StrictAliases,
		Annotators:    annos,
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
func Compile(options Options) (result *Result, err error) {
	defer panic_wrapper(&err)

	// validate the options
	err = validateOptions(&options)
	if err != nil {
		return nil, fmt.Errorf("Ungültige Compiler Optionen: %w", err)
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
		return nil, fmt.Errorf("Fehler beim Parsen: %w", err)
	}

	options.Log("Kompiliere den Abstrakten Syntaxbaum zu LLVM ir")

	if !options.LinkInModules {
		irBuff := &bytes.Buffer{}
		comp_result, err := newCompiler(ddp_main_module, options.ErrorHandler, options.OptimizationLevel).compile(irBuff, true)
		if err != nil {
			return nil, err
		}

		// early return
		if !options.LinkInListDefs && options.OutputType == OutputIR {
			options.To.Write(irBuff.Bytes())
			return comp_result, nil
		}

		// if we did not return, we need it as a llvm.Module
		options.Log("Erstelle llvm Context")
		llctx, err := newllvmContext()
		if err != nil {
			return nil, fmt.Errorf("Fehler beim Erstellen des llvm Context: %w", err)
		}
		defer options.Log("Entsorge llvm Context")
		defer llctx.Dispose()

		options.Log("Parse llvm-ir zu llvm-Module")
		mod, err := llctx.parseIR(irBuff.Bytes())
		if err != nil {
			return nil, fmt.Errorf("Fehler beim Parsen von llvm-ir: %w", err)
		}
		defer mod.Dispose()

		if options.LinkInListDefs {
			options.Log("Parse %s zu llvm-Module", ddppath.DDP_List_Types_Defs_LL)
			list_defs, err := llctx.parseListDefs()
			if err != nil {
				return nil, fmt.Errorf("Fehler beim Parsen von %s: %w", ddppath.DDP_List_Types_Defs_LL, err)
			}
			// no defer list_defs.Dispose() because it will be destroyed when linking it into the main module

			options.Log("Linke ddp_list_types_defs mit dem Hauptmodul")
			if err := llvmLinkAllModules(mod, []llvm.Module{list_defs}); err != nil {
				return nil, fmt.Errorf("Fehler beim Linken von %s: %w", ddppath.DDP_List_Types_Defs_LL, err)
			}
		}

		switch options.OutputType {
		case OutputIR:
			_, err := io.WriteString(options.To, mod.String())
			return comp_result, err
		case OutputAsm, OutputObj:
			file_type := llvm.AssemblyFile
			if options.OutputType == OutputObj {
				file_type = llvm.ObjectFile
				options.Log("Kompiliere llvm-ir zu Object Code")
			} else {
				options.Log("Kompiliere llvm-ir zu Assembler")
			}

			if options.OptimizationLevel >= 1 {
				llctx.optimizeModule(mod)
			}

			if _, err := llctx.compileModule(mod, file_type, options.To); err != nil {
				return nil, fmt.Errorf("Fehler beim Kompilieren von llvm-ir: %w", err)
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
		return nil, fmt.Errorf("Fehler beim Erstellen des llvm Context: %w", err)
	}
	defer options.Log("Entsorge llvm Context")
	defer llctx.Dispose()

	ll_modules_ir := map[string]*bytes.Buffer{}

	dependencies, err := compileWithImports(ddp_main_module, func(m *ast.Module) io.Writer {
		ll_modules_ir[m.FileName] = &bytes.Buffer{}
		return ll_modules_ir[m.FileName]
	}, options.ErrorHandler, options.OptimizationLevel)
	if err != nil {
		return nil, err
	}

	ll_modules := map[string]llvm.Module{}
	// optionally link in list-defs
	if options.LinkInListDefs {
		options.Log("Parse %s zu llvm-Module", ddppath.DDP_List_Types_Defs_LL)
		list_defs, err := llctx.parseListDefs()
		if err != nil {
			return nil, fmt.Errorf("Fehler beim Parsen von %s: %w", ddppath.DDP_List_Types_Defs_LL, err)
		}
		// no defer list_defs.Dispose() because it will be destroyed when linking it into the main module
		ll_modules[ddppath.DDP_List_Types_Defs_LL] = list_defs
	}

	for name := range ll_modules_ir {
		options.Log("Parse '%s' zu llvm-Module", name)
		// we do not need to defer llmod.Dispose here
		// because the modules will be destroyed when linking them into the main module
		llmod, err := llctx.parseIR(ll_modules_ir[name].Bytes())
		if err != nil {
			ll_modules_ir[name].WriteTo(options.To) // TODO DEBUG
			return nil, err
		}
		ll_modules[name] = llmod
	}
	ll_main_module := ll_modules[ddp_main_module.FileName]
	delete(ll_modules, ddp_main_module.FileName)

	defer ll_main_module.Dispose()
	options.Log("Linke llvm Module")
	if err := llvmLinkAllModules(ll_main_module, mapToSlice(ll_modules)); err != nil {
		return nil, fmt.Errorf("Fehler beim Linken von llvm-Modulen: %w", err)
	}

	// if we output llvm ir we are finished here
	if options.OutputType == OutputIR {
		if _, err := io.WriteString(options.To, ll_main_module.String()); err != nil {
			return nil, err
		}

		return &Result{Dependencies: dependencies}, nil
	}

	llctx.optimizeModule(ll_main_module)

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

// writes the definitions of the inbuilt ddp list types to w
// optimizationLevel is the same as in the compiler Options
func DumpListDefinitions(w io.Writer, outputType OutputType, errorHandler ddperror.Handler, optimizationLevel uint) (err error) {
	defer panic_wrapper(&err)

	irBuff := bytes.Buffer{}
	if err := newCompiler(nil, errorHandler, optimizationLevel).dumpListDefinitions(&irBuff); err != nil {
		return err
	}

	llctx, err := newllvmContext()
	if err != nil {
		return fmt.Errorf("Fehler beim Erstellen des llvm Context: %w", err)
	}
	defer llctx.Dispose()

	list_mod, err := llctx.parseIR(irBuff.Bytes())
	if err != nil {
		return fmt.Errorf("Fehler beim Parsen von llvm-ir: %w", err)
	}
	defer list_mod.Dispose()

	if optimizationLevel >= 1 {
		llctx.optimizeModule(list_mod)
	}

	switch outputType {
	case OutputIR:
		_, err := io.WriteString(w, list_mod.String())
		return err
	case OutputBC:
		buff := llvm.WriteBitcodeToMemoryBuffer(list_mod)
		defer buff.Dispose()
		_, err := w.Write(buff.Bytes())
		return err
	case OutputAsm:
		_, err := llctx.compileModule(list_mod, llvm.AssemblyFile, w)
		return err
	case OutputObj:
		_, err := llctx.compileModule(list_mod, llvm.ObjectFile, w)
		return err
	}
	return errors.New("invalid compiler.OutputType")
}

func mapToSlice[T comparable, U any](m map[T]U) []U {
	result := make([]U, 0, len(m))
	for _, v := range m {
		result = append(result, v)
	}
	return result
}

// wraps a panic with more information and re-panics
func panic_wrapper(out_err *error) {
	if err := recover(); err != nil {
		if err, ok := err.(*CompilerError); ok {
			panic(err)
		}

		stack_trace := debug.Stack()
		*out_err = &CompilerError{
			Err:        err,
			Msg:        getCompilerErrorMsg(err),
			ModulePath: "not found",
			StackTrace: stack_trace,
		}
	}
}
