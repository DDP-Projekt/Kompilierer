package compiler

import (
	"errors"
	"fmt"
	"io"
	"runtime/debug"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ast/annotators"
	"github.com/DDP-Projekt/Kompilierer/src/compiler/llvm"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/ddppath"
	"github.com/DDP-Projekt/Kompilierer/src/parser"
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
		FileName:     options.FileName,
		Source:       options.Source,
		Tokens:       nil,
		Modules:      nil,
		ErrorHandler: options.ErrorHandler,
		Annotators:   annos,
	}
}

// the result of a compilation
type Result struct {
	// the resulting module
	llMod llvm.Module
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

	llContext, err := newllvmModuleContext(ddp_main_module.FileName)
	if err != nil {
		return nil, fmt.Errorf("Fehler beim erstellen des LLVM Context: %w", err)
	}

	if !options.LinkInModules {
		compiler, err := newCompiler(ddp_main_module.FileName, ddp_main_module, llContext, options.ErrorHandler, options.OptimizationLevel)
		if err != nil {
			return nil, err
		}
		comp_result := compiler.compile(true)
		defer comp_result.llMod.Dispose()

		// early return
		if !options.LinkInListDefs && options.OutputType == OutputIR {
			options.To.Write([]byte(comp_result.llMod.String()))
			return &comp_result, nil
		}

		if options.LinkInListDefs {
			options.Log("Parse %s zu llvm-Module", ddppath.DDP_List_Types_Defs_LL)
			list_defs, err := parseListDefsIntoContext(&llContext)
			if err != nil {
				return nil, fmt.Errorf("Fehler beim Parsen von %s: %w", ddppath.DDP_List_Types_Defs_LL, err)
			}
			defer list_defs.Dispose()
			// no defer list_defs.Dispose() because it will be destroyed when linking it into the main module

			options.Log("Linke ddp_list_types_defs mit dem Hauptmodul")
			if err := llvmLinkAllModules(comp_result.llMod, []llvm.Module{list_defs}); err != nil {
				return nil, fmt.Errorf("Fehler beim Linken von %s: %w", ddppath.DDP_List_Types_Defs_LL, err)
			}
		}

		switch options.OutputType {
		case OutputIR:
			_, err := io.WriteString(options.To, comp_result.llMod.String())
			return &comp_result, err
		case OutputAsm, OutputObj:
			file_type := llvm.AssemblyFile
			if options.OutputType == OutputObj {
				file_type = llvm.ObjectFile
				options.Log("Kompiliere llvm-ir zu Object Code")
			} else {
				options.Log("Kompiliere llvm-ir zu Assembler")
			}

			if options.OptimizationLevel >= 1 {
				if err := llContext.optimizeModule(comp_result.llMod); err != nil {
					return nil, fmt.Errorf("Fehler beim Optimieren des Modules: %w", err)
				}
			}

			if _, err := llContext.compileModule(comp_result.llMod, file_type, options.To); err != nil {
				return nil, fmt.Errorf("Fehler beim Kompilieren von llvm-ir: %w", err)
			}

			return &comp_result, nil
		default:
			return nil, errors.New("invalid compiler.OutputType")
		}
	}
	// options.LinkInModules == true

	results, dependencies, err := compileWithImports(ddp_main_module, func(m *ast.Module) (llvmTargetContext, error) {
		return llContext, nil
	}, options.ErrorHandler, options.OptimizationLevel)
	if err != nil {
		return nil, err
	}
	ll_modules := make(map[*ast.Module]llvm.Module, len(results))
	for mod, r := range results {
		ll_modules[mod] = r.llMod
	}

	// optionally link in list-defs
	if options.LinkInListDefs {
		options.Log("Parse %s zu llvm-Module", ddppath.DDP_List_Types_Defs_LL)
		list_defs, err := parseListDefsIntoContext(&llContext)
		if err != nil {
			return nil, fmt.Errorf("Fehler beim Parsen von %s: %w", ddppath.DDP_List_Types_Defs_LL, err)
		}
		// no defer list_defs.Dispose() because it will be destroyed when linking it into the main module
		ll_modules[&ast.Module{FileName: ddppath.DDP_List_Types_Defs_LL}] = list_defs
	}

	ll_main_module := ll_modules[ddp_main_module]
	delete(ll_modules, ddp_main_module)
	defer func() {
		ll_main_module.Dispose()
		for _, mod := range ll_modules {
			mod.Dispose()
		}
	}()

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

	if err := llContext.optimizeModule(ll_main_module); err != nil {
		return nil, fmt.Errorf("Fehler beim Optimieren des Modules: %w", err)
	}

	file_type := llvm.AssemblyFile
	if options.OutputType == OutputObj {
		file_type = llvm.ObjectFile
		options.Log("Kompiliere llvm-ir zu Object Code")
	} else {
		options.Log("Kompiliere llvm-ir zu Assembler")
	}

	if _, err := llContext.compileModule(ll_main_module, file_type, options.To); err != nil {
		return nil, err
	}

	return &Result{Dependencies: dependencies}, nil
}

// writes the definitions of the inbuilt ddp list types to w
// optimizationLevel is the same as in the compiler Options
func DumpListDefinitions(w io.Writer, outputType OutputType, errorHandler ddperror.Handler, optimizationLevel uint) (err error) {
	defer panic_wrapper(&err)

	context, err := newllvmModuleContext(ddppath.LIST_DEFS_NAME)
	if err != nil {
		return fmt.Errorf("Fehler beim erstellen des LLVM Context: %w", err)
	}

	compiler, err := newCompiler(ddppath.LIST_DEFS_NAME, nil, context, errorHandler, optimizationLevel)
	if err != nil {
		return err
	}
	list_defs := compiler.dumpListDefinitions()
	defer list_defs.Dispose()

	if optimizationLevel >= 1 {
		if err := context.optimizeModule(list_defs); err != nil {
			return fmt.Errorf("Fehler beim Optimieren des Modules: %w", err)
		}
	}

	switch outputType {
	case OutputIR:
		_, err := io.WriteString(w, list_defs.String())
		return err
	case OutputBC:
		buff := llvm.WriteBitcodeToMemoryBuffer(list_defs)
		defer buff.Dispose()
		_, err := w.Write(buff.Bytes())
		return err
	case OutputAsm:
		_, err := context.compileModule(list_defs, llvm.AssemblyFile, w)
		return err
	case OutputObj:
		_, err := context.compileModule(list_defs, llvm.ObjectFile, w)
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
