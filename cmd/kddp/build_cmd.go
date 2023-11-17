package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/DDP-Projekt/Kompilierer/cmd/internal/gcc"
	"github.com/DDP-Projekt/Kompilierer/cmd/internal/linker"
	"github.com/DDP-Projekt/Kompilierer/src/compiler"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
)

// $kddp build is the main command which compiles a .ddp file into a executable
type BuildCommand struct {
	fs *flag.FlagSet // FlagSet for the arguments
	// arguments
	filePath         string // input file (.ddp), neccessery, first argument
	outPath          string // path for the output file, specified by the -o flag, may be empty
	main_path        string // path to the main.o file, specified by the --main flag, may be empty
	gcc_flags        string // custom flags that are passed to gcc
	extern_gcc_flags string // custom flags passed to extern .c files
	nodeletes        bool   // should temp files be deleted, specified by the --nodeletes flag
	verbose          bool   // print verbose output, specified by the --verbose flag
	link_modules     bool   // wether to link the llvm modules together
	link_list_defs   bool   // wether to link the list-defs into the main module
	gcc_executable   string // path to the gcc executable
}

func NewBuildCommand() *BuildCommand {
	return &BuildCommand{
		fs:               flag.NewFlagSet("kompiliere", flag.ExitOnError),
		filePath:         "",
		outPath:          "",
		main_path:        "",
		gcc_flags:        "",
		extern_gcc_flags: "",
		nodeletes:        false,
		verbose:          false,
		link_modules:     true,
		link_list_defs:   true,
		gcc_executable:   gcc.Cmd(),
	}
}

func (cmd *BuildCommand) Init(args []string) error {
	// set all the flags
	cmd.fs.StringVar(&cmd.outPath, "o", cmd.outPath, "Optionaler Pfad der Ausgabedatei")
	cmd.fs.StringVar(&cmd.main_path, "main", cmd.main_path, "Optionaler Pfad zur main.o Datei")
	cmd.fs.StringVar(&cmd.gcc_flags, "gcc_optionen", cmd.gcc_flags, "Benutzerdefinierte Optionen, die gcc übergeben werden")
	cmd.fs.StringVar(&cmd.extern_gcc_flags, "externe_gcc_optionen", cmd.extern_gcc_flags, "Benutzerdefinierte Optionen, die gcc für jede externe .c Datei übergeben werden")
	cmd.fs.BoolVar(&cmd.nodeletes, "nichts_loeschen", cmd.nodeletes, "Keine temporären Dateien löschen")
	cmd.fs.BoolVar(&cmd.verbose, "wortreich", cmd.verbose, "Gibt wortreiche Informationen während des Befehls")
	cmd.fs.BoolVar(&cmd.link_modules, "module_linken", cmd.link_modules, "Ob alle Module in das Hauptmodul gelinkt werden sollen")
	cmd.fs.BoolVar(&cmd.link_list_defs, "list_defs_linken", cmd.link_list_defs, "Ob die eingebauten Listen Definitionen in das Hauptmodul gelinkt werden sollen")
	cmd.fs.StringVar(&cmd.gcc_executable, "gcc_executable", cmd.gcc_executable, "Pfad zum gcc, der genutzt werden soll")

	if err := parseFlagSet(cmd.fs, args); err != nil {
		return fmt.Errorf("Fehler beim Parsen der Argumente: %w", err)
	}
	if cmd.fs.NArg() >= 1 {
		cmd.filePath = cmd.fs.Arg(0)
		if filepath.Ext(cmd.filePath) != ".ddp" {
			return fmt.Errorf("Die Eingabedatei ist keine .ddp Datei")
		}
	}
	return nil
}

// invokes the main behaviour of kddp
func (cmd *BuildCommand) Run() error {
	// helper function to print verbose output if the flag was set
	print := func(format string, args ...any) {
		if cmd.verbose {
			fmt.Printf(format+"\n", args...)
		}
	}

	// set the gcc executable
	gcc.SetGcc(cmd.gcc_executable)

	// determine the final output type (by file extension)
	compOutType := compiler.OutputIR
	extension := ".ll" // assume the -c flag
	targetExe := false // -c was not set
	switch ext := filepath.Ext(cmd.outPath); ext {
	case ".ll":
		compOutType = compiler.OutputIR
	case ".s", ".asm":
		extension = ext
		compOutType = compiler.OutputAsm
	case ".o", ".obj":
		extension = ext
		compOutType = compiler.OutputObj
	case ".exe":
		extension = ext
		targetExe = true
		compOutType = compiler.OutputObj
	case "":
		extension = ext
		targetExe = true
		compOutType = compiler.OutputObj
	default: // by default we create a executable
		if runtime.GOOS == "windows" {
			extension = ".exe"
		} else if runtime.GOOS == "linux" {
			extension = ""
		}
		targetExe = true
	}

	// disable comments if the .ll files are deleted anyways
	if compOutType != compiler.OutputIR && !cmd.nodeletes {
		compiler.Comments_Enabled = false
	}

	// create the path to the output file
	if cmd.outPath == "" { // if no output file was specified, we use the name of the input .ddp file
		if cmd.filePath == "" {
			cmd.outPath = "ddp" + extension // if no input file was specified, we use the default name "ddp"
		} else {
			cmd.outPath = changeExtension(cmd.filePath, extension)
		}
	} else { // otherwise we use the provided name
		cmd.outPath = changeExtension(cmd.outPath, extension)
	}

	print("Erstelle Ausgabeordner: %s", filepath.Dir(cmd.outPath))
	// make the output file directory
	if err := os.MkdirAll(filepath.Dir(cmd.outPath), os.ModePerm); err != nil {
		return fmt.Errorf("Fehler beim Erstellen des Ausgabeordners: %w", err)
	}

	objPath := changeExtension(cmd.outPath, ".o")

	var (
		to  *os.File
		err error
	)
	if targetExe {
		to, err = os.OpenFile(objPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
		if !cmd.nodeletes {
			defer func() {
				print("Lösche %s", objPath)
				os.Remove(objPath)
			}()
		}
	} else {
		to, err = os.OpenFile(cmd.outPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	}
	if err != nil {
		return err
	}
	defer to.Close()

	var src []byte
	// if no input file was specified, we read from stdin
	if cmd.filePath == "" {
		cmd.filePath = "stdin"
		if src, err = io.ReadAll(os.Stdin); err != nil {
			return fmt.Errorf("Fehler beim Lesen von stdin: %w", err)
		}
	} else {
		if src, err = os.ReadFile(cmd.filePath); err != nil {
			return fmt.Errorf("Fehler beim Lesen von %s: %w", cmd.filePath, err)
		}
	}

	errorHandler := ddperror.MakeAdvancedHandler(cmd.filePath, src, os.Stderr)

	print("Kompiliere DDP-Quellcode nach %s", cmd.outPath)
	result, err := compiler.Compile(compiler.Options{
		FileName:                cmd.filePath,
		Source:                  src,
		From:                    nil,
		To:                      to,
		OutputType:              compOutType,
		ErrorHandler:            errorHandler,
		Log:                     print,
		DeleteIntermediateFiles: !cmd.nodeletes,
		LinkInModules:           cmd.link_modules,
		LinkInListDefs:          cmd.link_list_defs,
	})
	if err != nil {
		return fmt.Errorf("Fehler beim Kompilieren: %w", err)
	}

	if !targetExe {
		return nil
	}

	// the target is an executable so we link the produced object file
	print("Objekte werden gelinkt")
	if output, err := linker.LinkDDPFiles(linker.Options{
		InputFile:               objPath,
		OutputFile:              cmd.outPath,
		Dependencies:            result,
		Log:                     print,
		DeleteIntermediateFiles: !cmd.nodeletes,
		GCCFlags:                cmd.gcc_flags,
		MainFile:                cmd.main_path,
		ExternGCCFlags:          cmd.extern_gcc_flags,
		LinkInListDefs:          !cmd.link_list_defs, // if they are allready linked in, don't link them again
	}); err != nil {
		return fmt.Errorf("Fehler beim Linken: %w (%s)", err, string(output))
	}

	return nil
}

func (cmd *BuildCommand) Name() string {
	return cmd.fs.Name()
}

func (cmd *BuildCommand) Usage() string {
	return `kompiliere <Eingabedatei> <Optionen>: Kompiliert die gegebene .ddp Datei zu einer ausführbaren Datei
Optionen:
	-o <Ausgabepfad>: Optionaler Pfad der Ausgabedatei
	--wortreich: Gibt wortreiche Informationen während des Befehls
	--nichts_loeschen: Temporäre Dateien werden nicht gelöscht
	--gcc_optionen: Benutzerdefinierte Optionen, die gcc übergeben werden
	--externe_gcc_optionen: Benutzerdefinierte Optionen, die gcc für jede externe .c Datei übergeben werden
	--module_linken: Ob alle Module in das Hauptmodul gelinkt werden sollen, standardmäßig wahr
	--list_defs_linken: Ob die eingebauten Listen Definitionen in das Hauptmodul gelinkt werden sollen, standardmäßig wahr`
}

// helper function
// returns path with the specified extension
func changeExtension(path, ext string) string {
	return path[:len(path)-len(filepath.Ext(path))] + ext
}
