// file command.go defines the sub-commands
// of kddp like build, help, etc.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"

	"github.com/DDP-Projekt/Kompilierer/cmd/internal/linker"
	"github.com/DDP-Projekt/Kompilierer/src/compiler"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/ddppath"
	"github.com/DDP-Projekt/Kompilierer/src/parser"
)

// interface for a sub-command
type Command interface {
	Init([]string) error // initialize the command and parse it's flags
	Run() error          // run the command
	Name() string        // every command must specify a name
	Usage() string       // and a usage
}

var commands = []Command{
	NewHelpCommand(),
	NewBuildCommand(),
	NewParseCommand(),
	NewVersionCommand(),
	NewRunCommand(),
	NewDumpListDefsCommand(),
}

// $kddp help prints some help information
type HelpCommand struct {
	fs  *flag.FlagSet // FlagSet for the arguments
	cmd string        // command to print information about, may be empty
}

func NewHelpCommand() *HelpCommand {
	return &HelpCommand{
		fs:  flag.NewFlagSet("hilfe", flag.ExitOnError),
		cmd: "",
	}
}

func (cmd *HelpCommand) Init(args []string) error {
	// maybe a sub-command was specified
	if len(args) > 0 {
		cmd.cmd = args[0]
	}
	return cmd.fs.Parse(args)
}

func (cmd *HelpCommand) Run() error {
	// if a sub-command was specified print only its usage
	if cmd.cmd != "" {
		for _, command := range commands {
			if command.Name() == cmd.cmd {
				fmt.Println(command.Usage())
				return nil
			}
		}
	}

	// otherwise, print the usage of every command
	fmt.Println("Verfügbare Befehle:")
	for _, cmd := range commands {
		fmt.Println(cmd.Usage() + "\n")
	}

	return nil
}

func (cmd *HelpCommand) Name() string {
	return cmd.fs.Name()
}

func (cmd *HelpCommand) Usage() string {
	return `hilfe <Befehl>: Zeigt Nutzungsinformationen über den Befehl`
}

// $kddp build is the main command which compiles a .ddp file into a executable
type BuildCommand struct {
	fs *flag.FlagSet // FlagSet for the arguments
	// arguments
	filePath         string // input file (.ddp), neccessery, first argument
	outPath          string // path for the output file, specified by the -o flag, may be empty
	gcc_flags        string // custom flags that are passed to gcc
	extern_gcc_flags string // custom flags passed to extern .c files
	nodeletes        bool   // should temp files be deleted, specified by the --nodeletes flag
	verbose          bool   // print verbose output, specified by the --verbose flag
	link_modules     bool   // wether to link the llvm modules together
	link_list_defs   bool   // wether to link the list-defs into the main module
}

func NewBuildCommand() *BuildCommand {
	return &BuildCommand{
		fs:               flag.NewFlagSet("kompiliere", flag.ExitOnError),
		filePath:         "",
		outPath:          "",
		gcc_flags:        "",
		extern_gcc_flags: "",
		nodeletes:        false,
		verbose:          false,
		link_modules:     true,
		link_list_defs:   true,
	}
}

func (cmd *BuildCommand) Init(args []string) error {
	// a input .ddp file is necessary
	if len(args) < 1 {
		return fmt.Errorf("Der kompiliere Befehl braucht eine Eingabedatei")
	}

	// the first argument must be the input file (.ddp)
	cmd.filePath = args[0]
	if filepath.Ext(cmd.filePath) != ".ddp" {
		return fmt.Errorf("Die Eingabedatei ist keine .ddp Datei")
	}

	// set all the flags
	cmd.fs.StringVar(&cmd.outPath, "o", cmd.outPath, "Optionaler Pfad der Ausgabedatei")
	cmd.fs.StringVar(&cmd.gcc_flags, "gcc_optionen", cmd.gcc_flags, "Benutzerdefinierte Optionen, die gcc übergeben werden")
	cmd.fs.StringVar(&cmd.extern_gcc_flags, "externe_gcc_optionen", cmd.extern_gcc_flags, "Benutzerdefinierte Optionen, die gcc für jede externe .c Datei übergeben werden")
	cmd.fs.BoolVar(&cmd.nodeletes, "nichts_loeschen", cmd.nodeletes, "Keine temporären Dateien löschen")
	cmd.fs.BoolVar(&cmd.verbose, "wortreich", cmd.verbose, "Gibt wortreiche Informationen während des Befehls")
	cmd.fs.BoolVar(&cmd.link_modules, "module_linken", cmd.link_modules, "Ob alle Module in das Hauptmodul gelinkt werden sollen")
	cmd.fs.BoolVar(&cmd.link_list_defs, "list_defs_linken", cmd.link_list_defs, "Ob die eingebauten Listen Definitionen in das Hauptmodul gelinkt werden sollen")
	return cmd.fs.Parse(args[1:])
}

// invokes the main behaviour of kddp
func (cmd *BuildCommand) Run() error {
	// helper function to print verbose output if the flag was set
	print := func(format string, args ...any) {
		if cmd.verbose {
			fmt.Printf(format+"\n", args...)
		}
	}

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
		cmd.outPath = changeExtension(cmd.filePath, extension)
	} else { // otherwise we use the provided name
		cmd.outPath = changeExtension(cmd.outPath, extension)
	}

	print("Erstelle Ausgabeordner: %s", filepath.Dir(cmd.outPath))
	// make the output file directory
	if err := os.MkdirAll(filepath.Dir(cmd.outPath), os.ModePerm); err != nil {
		return fmt.Errorf("Fehler beim Erstellen des Ausgabeordners: %s", err.Error())
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

	src, err := os.ReadFile(cmd.filePath)
	if err != nil {
		return err
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
		return err
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
		ExternGCCFlags:          cmd.extern_gcc_flags,
		LinkInListDefs:          !cmd.link_list_defs, // if they are allready linked in, don't link them again
	}); err != nil {
		return fmt.Errorf("Fehler beim Linken: %s (%s)", err, string(output))
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

// $kddp parse parses the given .ddp file and outputs the resulting ast
// mainly for testing
type ParseCommand struct {
	fs       *flag.FlagSet
	filePath string
	outPath  string
}

func NewParseCommand() *ParseCommand {
	return &ParseCommand{
		fs:       flag.NewFlagSet("parse", flag.ExitOnError),
		filePath: "",
		outPath:  "",
	}
}

func (cmd *ParseCommand) Init(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("Der parse Befehl braucht eine Eingabedatei")
	}

	cmd.filePath = args[0]
	if filepath.Ext(cmd.filePath) != ".ddp" {
		return fmt.Errorf("Die Eingabedatei ist keine .ddp Datei")
	}

	cmd.fs.StringVar(&cmd.outPath, "o", cmd.outPath, "Optionaler Pfad der Ausgabedatei")
	return cmd.fs.Parse(args[1:])
}

func (cmd *ParseCommand) Run() error {
	module, err := parser.Parse(parser.Options{FileName: cmd.filePath, ErrorHandler: ddperror.MakeBasicHandler(os.Stderr)})
	if err != nil {
		return err
	}

	if module.Ast.Faulty {
		fmt.Println("Der generierte Abstrakte Syntaxbaum ist fehlerhaft")
	}

	if cmd.outPath != "" {
		if file, err := os.OpenFile(cmd.outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm); err != nil {
			return fmt.Errorf("Ausgabedatei konnte nicht geöffnet werden")
		} else {
			defer file.Close()
			if _, err := file.WriteString(module.Ast.String()); err != nil {
				return fmt.Errorf("Ausgabedatei konnte nicht beschrieben werden: %s", err.Error())
			}
		}
	} else {
		fmt.Println(module.Ast.String())
	}

	return nil
}

func (cmd *ParseCommand) Name() string {
	return cmd.fs.Name()
}

func (cmd *ParseCommand) Usage() string {
	return `parse <Eingabedatei> <Optionen>: Parse die Eingabedatei zu einem Abstrakten Syntaxbaum
Optionen:
	-o <Pfad>: Optionaler Pfad der Ausgabedatei`
}

// $kddp version provides information about the version of the used kddp build
type VersionCommand struct {
	fs         *flag.FlagSet
	verbose    bool
	build_info bool
}

func NewVersionCommand() *VersionCommand {
	return &VersionCommand{
		fs:         flag.NewFlagSet("version", flag.ExitOnError),
		verbose:    false,
		build_info: false,
	}
}

func (cmd *VersionCommand) Init(args []string) error {
	cmd.fs.BoolVar(&cmd.verbose, "wortreich", cmd.verbose, "Zeige wortreiche Informationen")
	cmd.fs.BoolVar(&cmd.build_info, "go_build_info", cmd.build_info, "Zeige Go build Informationen")
	return cmd.fs.Parse(args)
}

var (
	DDPVERSION     string = "undefined"
	LLVMVERSION    string = "undefined"
	GCCVERSION     string = "undefined"
	GCCVERSIONFULL string = "undefined"
)

func (cmd *VersionCommand) Run() error {
	fmt.Printf("%s %s %s\n", DDPVERSION, runtime.GOOS, runtime.GOARCH)

	if bi, ok := debug.ReadBuildInfo(); ok {
		if cmd.verbose {
			fmt.Printf("Go Version: %s\n", bi.GoVersion)
		}

		if cmd.build_info {
			fmt.Printf("Go build info:\n")
			for _, v := range bi.Settings {
				fmt.Printf("%s: %s\n", v.Key, v.Value)
			}
		}
	} else if cmd.verbose {
		fmt.Printf("Go Version: undefined\n")
	}

	if cmd.verbose {
		fmt.Printf("GCC Version: %s ; Full: %s\n", GCCVERSION, GCCVERSIONFULL)
		fmt.Printf("LLVM Version: %s\n", LLVMVERSION)
	}

	return nil
}

func (cmd *VersionCommand) Name() string {
	return cmd.fs.Name()
}

func (cmd *VersionCommand) Usage() string {
	return `version <Optionen>: Zeige informationen zu dieser DDP Version
Optionen:
	--wortreich: Zeige wortreiche Informationen
	--go_build_info: Zeige Go build Informationen`
}

// $kddp run compiles and runs the specified
type RunCommand struct {
	fs *flag.FlagSet // FlagSet for the arguments
	// arguments
	filePath         string // input file (.ddp), neccessery, first argument
	gcc_flags        string // custom flags that are passed to gcc
	extern_gcc_flags string // custom flags passed to extern .c files
	verbose          bool   // print verbose output, specified by the --verbose flag
}

func NewRunCommand() *RunCommand {
	return &RunCommand{
		fs:               flag.NewFlagSet("starte", flag.ExitOnError),
		filePath:         "",
		gcc_flags:        "",
		extern_gcc_flags: "",
		verbose:          false,
	}
}

func (cmd *RunCommand) Init(args []string) error {
	// a input .ddp file is necessary
	if len(args) < 1 {
		return fmt.Errorf("Der starte Befehl braucht eine Eingabedatei")
	}

	// the first argument must be the input file (.ddp)
	cmd.filePath = args[0]
	if filepath.Ext(cmd.filePath) != ".ddp" {
		return fmt.Errorf("Die Eingabedatei ist keine .ddp Datei")
	}

	// set all the flags
	cmd.fs.StringVar(&cmd.gcc_flags, "gcc_optionen", cmd.gcc_flags, "Benutzerdefinierte Optionen, die gcc übergeben werden")
	cmd.fs.StringVar(&cmd.extern_gcc_flags, "externe_gcc_optionen", cmd.extern_gcc_flags, "Benutzerdefinierte Optionen, die gcc für jede externe .c Datei übergeben werden")
	cmd.fs.BoolVar(&cmd.verbose, "wortreich", cmd.verbose, "Gibt wortreiche Informationen während des Befehls")
	return cmd.fs.Parse(args[1:])
}

func (cmd *RunCommand) Run() error {
	// helper function to print verbose output if the flag was set
	print := func(format string, args ...any) {
		if cmd.verbose {
			fmt.Printf(format+"\n", args...)
		}
	}

	outDir, err := os.MkdirTemp("", "KDDP_RUN")
	if err != nil {
		return err
	}
	defer os.RemoveAll(outDir)

	exePath := filepath.Join(outDir, filepath.Base(changeExtension(cmd.filePath, ".exe")))

	buildCmd := NewBuildCommand()
	buildCmd.filePath = cmd.filePath
	buildCmd.outPath = exePath
	buildCmd.verbose = cmd.verbose
	buildCmd.gcc_flags = cmd.gcc_flags
	buildCmd.extern_gcc_flags = cmd.extern_gcc_flags

	print("Kompiliere den Quellcode")
	if err = buildCmd.Run(); err != nil {
		return err
	}

	print("Starte das Programm\n")
	ddpExe := exec.Command(exePath, cmd.fs.Args()...)
	ddpExe.Stdin = os.Stdin
	ddpExe.Stdout = os.Stdout
	ddpExe.Stderr = os.Stderr

	return ddpExe.Run()
}

func (cmd *RunCommand) Name() string {
	return cmd.fs.Name()
}

func (cmd *RunCommand) Usage() string {
	return `starte <Eingabedatei> <Optionen>: Kompiliert und führt die gegebene .ddp Datei aus
Optionen:
	--wortreich: Gibt wortreiche Informationen während des Befehls
	--gcc_optionen: Benutzerdefinierte Optionen, die gcc übergeben werden
	--externe_gcc_optionen: Benutzerdefinierte Optionen, die gcc für jede externe .c Datei übergeben werden`
}

// $kddp dumplistdefs dumps the list definitions to the specified file
type DumpListDefsCommand struct {
	fs *flag.FlagSet // FlagSet for the arguments
	// arguments
	filePath string // output file path
	asm      bool   // wether to output assembly
	llvm_ir  bool   // wether to output llvm ir
	llvm_bc  bool   // wether to output llvm bitcode
	object   bool   // wether to output an object file
}

func NewDumpListDefsCommand() *DumpListDefsCommand {
	return &DumpListDefsCommand{
		fs:       flag.NewFlagSet("dump-list-defs", flag.ExitOnError),
		filePath: ddppath.LIST_DEFS_NAME,
		asm:      false,
		llvm_ir:  false,
		llvm_bc:  false,
		object:   false,
	}
}

func (cmd *DumpListDefsCommand) Init(args []string) error {
	// a input .ddp file is necessary
	if len(args) < 1 {
		return fmt.Errorf("Der starte Befehl braucht eine Eingabedatei")
	}

	// set all the flags
	cmd.fs.StringVar(&cmd.filePath, "o", cmd.filePath, "Ausgabe Datei")
	cmd.fs.BoolVar(&cmd.asm, "asm", cmd.asm, "Wether to output assembly")
	cmd.fs.BoolVar(&cmd.llvm_ir, "llvm_ir", cmd.llvm_ir, "Wether to output llvm ir")
	cmd.fs.BoolVar(&cmd.llvm_bc, "llvm_bc", cmd.llvm_bc, "Wether to output llvm bitcode")
	cmd.fs.BoolVar(&cmd.object, "object", cmd.object, "Wether to output object file")
	return cmd.fs.Parse(args)
}

func (cmd *DumpListDefsCommand) Run() error {
	outputTypes := []compiler.OutputType{}
	if cmd.asm {
		outputTypes = append(outputTypes, compiler.OutputAsm)
	}
	if cmd.llvm_ir {
		outputTypes = append(outputTypes, compiler.OutputIR)
	}
	if cmd.llvm_bc {
		outputTypes = append(outputTypes, compiler.OutputBC)
	}
	if cmd.object {
		outputTypes = append(outputTypes, compiler.OutputObj)
	}

	for _, outType := range outputTypes {
		ext := ".ll"
		switch outType {
		case compiler.OutputAsm:
			ext = ".asm"
		case compiler.OutputObj:
			ext = ".o"
		case compiler.OutputBC:
			ext = ".bc"
		}
		file, err := os.OpenFile(changeExtension(cmd.filePath, ext), os.O_CREATE|os.O_TRUNC|os.O_RDWR, os.ModePerm)
		if err != nil {
			return err
		}
		defer file.Close()
		if err := compiler.DumpListDefinitions(file, outType, ddperror.MakeBasicHandler(os.Stderr)); err != nil {
			return err
		}
	}

	return nil
}

func (cmd *DumpListDefsCommand) Name() string {
	return cmd.fs.Name()
}

func (cmd *DumpListDefsCommand) Usage() string {
	return `dump-list-defs <Optionen>: Schreibt die Definitionen der eingebauten Listen Typen in die gegebene Ausgbe Datei
Optionen:
	-o: Ausgabe Datei
	--asm: ob assembly ausgegeben werden soll
	--llvm_ir: ob llvm ir ausgegeben werden soll
	--object: ob eine Objekt Datei ausgageben werden soll`
}
