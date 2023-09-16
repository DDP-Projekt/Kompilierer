package linker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DDP-Projekt/Kompilierer/cmd/internal/gcc"
	"github.com/DDP-Projekt/Kompilierer/src/compiler"
	"github.com/DDP-Projekt/Kompilierer/src/ddppath"
)

type Options struct {
	// input and output file paths
	// InputFile should be a .o file compiled from ddp-source-code
	InputFile, OutputFile string
	// the external dependencies of the InputFile
	// in the form of a compiler result
	// may be nil for no dependencies
	Dependencies *compiler.Result
	// optional Log function to print intermediate messages
	Log func(string, ...any)
	// wether or not to delete temporary files
	DeleteIntermediateFiles bool
	// flags for the final gcc call
	GCCFlags string
	// the .o file containing main(argc, argv)
	// if this is not set, the linker will use the default main
	MainFile string
	// flags passed to external .c files
	// for example to specify include directories
	ExternGCCFlags string
	// wether the inbuilt list-defs need to be linked in
	LinkInListDefs bool
}

func validateOptions(options *Options) error {
	if options.InputFile == "" {
		return fmt.Errorf("Keine Eingabedatei")
	}
	if options.Dependencies == nil {
		options.Dependencies = &compiler.Result{Dependencies: map[string]struct{}{}}
	}
	if options.Log == nil {
		options.Log = func(string, ...any) {}
	}
	if options.MainFile == "" {
		options.Log("Keine main.o angegeben, verwende Standard")
		options.MainFile = ddppath.Main_O
	}
	return nil
}

// link the given input file (a .o file compiled from ddp-source-code) with the given dependencies and flags
// to the ddpruntime, stdlib and ddp_list_types_defs into a executable
func LinkDDPFiles(options Options) ([]byte, error) {
	err := validateOptions(&options)
	if err != nil {
		return nil, fmt.Errorf("Ungültige Linker Optionen: %w", err)
	}

	// split the flags passed to gcc when compiling extern .c files
	extern_gcc_flags := strings.Split(options.ExternGCCFlags, " ")
	if options.ExternGCCFlags == "" {
		extern_gcc_flags = []string{}
	}

	var (
		link_objects = map[string][]string{}       // library-search-paths to library-filename map
		input_files  = []string{options.InputFile} // all input files (.o)
	)

	// add external files to be linked
	if options.DeleteIntermediateFiles {
		defer options.Log("Lösche temporäre Dateien")
	}
	for path := range options.Dependencies.Dependencies {
		filename := filepath.Base(path)
		// stdlib and runtime are linked by default
		// ignore them because of the Duden
		switch filename {
		case "libddpstdlib.a", "libddpruntime.a", ddppath.LIST_DEFS_NAME + ".o":
			continue
		}

		switch filepath.Ext(path) {
		case ".lib", ".a": // libraries are linked using the -l: flag
			if objs, ok := link_objects[filepath.Dir(path)]; ok {
				link_objects[filepath.Dir(path)] = append(objs, filename)
			} else {
				link_objects[filepath.Dir(path)] = []string{filename}
			}
		case ".o": // object files are simple input files
			input_files = append(input_files, path)
		case ".c": // .c files must be compiled first
			if outPath, output, err := compileCFile(path, extern_gcc_flags, options.Log); err != nil {
				return output, fmt.Errorf("Fehler beim Kompilieren von '%s': %w", path, err)
			} else {
				input_files = append(input_files, outPath)
				if options.DeleteIntermediateFiles {
					defer os.Remove(outPath)
				}
			}
		default:
			return nil, fmt.Errorf("Unerwartete Abhängigkeit '%s'", path)
		}
	}

	args := append(make([]string, 0), "-o", options.OutputFile, "-O2", "-L"+ddppath.Lib)

	// add all librarie-search-paths
	for k := range link_objects {
		args = append(args, "-L"+k)
	}

	// add the input files
	args = append(args, input_files...)

	// add external dependencies
	for _, libs := range link_objects {
		for _, lib := range libs {
			args = append(args, "-l:"+lib)
		}
	}

	// add default dependencies at the end, because dependencies might depend on the ddp runtime and list_types_defs
	args = append(args, "-lddpstdlib")
	if options.LinkInListDefs {
		args = append(args, ddppath.DDP_List_Types_Defs_O)
	}
	args = append(args, "-lddpruntime", "-lm")
	args = append(args, options.MainFile)

	// add additional gcc-flags such as other needed libraries
	if options.GCCFlags != "" {
		flags := strings.Split(options.GCCFlags, " ")
		args = append(args, flags...)
	}

	cmd := gcc.New(args...)
	options.Log("%s", cmd.String())
	return cmd.CombinedOutput()
}

// helper for invokeGCC
// returns the path to the output file, and the result of cmd.CombinedOutput()
func compileCFile(inputFile string, gcc_flags []string, Log func(string, ...any)) (string, []byte, error) {
	outPath := changeExtension(changeFilename(inputFile, "ddpextern_"+filepath.Base(inputFile)), ".o")

	args := append(make([]string, 0, 5), "-O2", "-c", "-Wall", "-o", outPath, inputFile)
	if len(gcc_flags) > 0 {
		args = append(args, gcc_flags...)
	}

	Log("Rufe gcc auf '%s' auf", inputFile)
	cmd := gcc.New(args...)
	Log(cmd.String())
	out, err := cmd.CombinedOutput()
	return outPath, out, err
}

// helper to change a filename in a filepath
func changeFilename(path, newName string) string {
	return path[:len(path)-len(filepath.Base(path))] + newName
}

func changeExtension(path, ext string) string {
	return path[:len(path)-len(filepath.Ext(path))] + ext
}
