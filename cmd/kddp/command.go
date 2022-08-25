// file command.go defines the sub-commands
// of kddp like build, help, etc.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"

	"github.com/DDP-Projekt/Kompilierer/internal/linker"
	"github.com/DDP-Projekt/Kompilierer/pkg/compiler"
	"github.com/DDP-Projekt/Kompilierer/pkg/interpreter"
	"github.com/DDP-Projekt/Kompilierer/pkg/parser"
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
	NewInterpretCommand(),
	NewBuildCommand(),
	NewParseCommand(),
	NewVersionCommand(),
}

// $kddp interpret walks the parsed ast and executes it
// it is meant mainly for testing
type InterpretCommand struct {
	fs       *flag.FlagSet // FlagSet for the arguments
	filePath string        // path to the input file
	outPath  string        // path to the output file for stdout/stderr, may be empty
}

func NewInterpretCommand() *InterpretCommand {
	return &InterpretCommand{
		fs: flag.NewFlagSet("interpret", flag.ExitOnError),
	}
}

func (cmd *InterpretCommand) Init(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("interpret requires a file name")
	}

	// first argument must be the input file
	cmd.filePath = args[0]
	if filepath.Ext(cmd.filePath) != ".ddp" {
		return fmt.Errorf("the provided file is not a .ddp file")
	}

	// parse command flags
	cmd.fs.StringVar(&cmd.outPath, "o", "", "provide a optional filepath where the output is written to")
	return cmd.fs.Parse(args[1:])
}

func (cmd *InterpretCommand) Run() error {
	// parse the input file into a ast
	ast, err := parser.ParseFile(cmd.filePath, errHndl)
	if err != nil {
		return err
	}

	interpreter := interpreter.New(ast, errHndl) // create the interpreter with the parsed ast

	// if a output file was specified, we set the interpreters stdout and stderr
	if cmd.outPath != "" {
		file, err := os.OpenFile(cmd.outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
		if err != nil {
			return fmt.Errorf("unable to open the output file")
		}
		defer file.Close()
		interpreter.Stdout = file
		interpreter.Stderr = file
	}

	return interpreter.Interpret() // interpret the ast
}

func (cmd *InterpretCommand) Name() string {
	return cmd.fs.Name()
}

func (cmd *InterpretCommand) Usage() string {
	return `interpret <filename> <options>: interprets the given .ddp file
options:
	-o <filepath>: writes the programs stdout to the given file`
}

// $kddp help prints some help information
type HelpCommand struct {
	fs  *flag.FlagSet // FlagSet for the arguments
	cmd string        // command to print information about, may be empty
}

func NewHelpCommand() *HelpCommand {
	return &HelpCommand{
		fs: flag.NewFlagSet("help", flag.ExitOnError),
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
	fmt.Println("available commands:")
	for _, cmd := range commands {
		fmt.Println(cmd.Usage() + "\n")
	}

	return nil
}

func (cmd *HelpCommand) Name() string {
	return cmd.fs.Name()
}

func (cmd *HelpCommand) Usage() string {
	return `help <command>: displays usage information`
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
	targetIR         bool   // only compile to llvm ir, specified by the -c flag
}

func NewBuildCommand() *BuildCommand {
	return &BuildCommand{
		fs:               flag.NewFlagSet("build", flag.ExitOnError),
		filePath:         "",
		outPath:          "",
		gcc_flags:        "",
		extern_gcc_flags: "",
		nodeletes:        false,
		verbose:          false,
		targetIR:         false,
	}
}

func (cmd *BuildCommand) Init(args []string) error {
	// a input .ddp file is necessary
	if len(args) < 1 {
		return fmt.Errorf("build requires a file name")
	}

	// the first argument must be the input file (.ddp)
	cmd.filePath = args[0]
	if filepath.Ext(cmd.filePath) != ".ddp" {
		return fmt.Errorf("the provided file is not a .ddp file")
	}

	// set all the flags
	cmd.fs.StringVar(&cmd.outPath, "o", "", "provide a optional filepath where the output is written to")
	cmd.fs.StringVar(&cmd.gcc_flags, "gcc_flags", "", "custom flags that are passed to gcc")
	cmd.fs.StringVar(&cmd.extern_gcc_flags, "extern_gcc_flags", "", "custom flags passed to extern .c files")
	cmd.fs.BoolVar(&cmd.nodeletes, "nodeletes", false, "don't delete temporary files such as .ll or .obj files")
	cmd.fs.BoolVar(&cmd.verbose, "verbose", false, "print verbose build output")
	cmd.fs.BoolVar(&cmd.targetIR, "c", false, "only produce llvm ir")
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
	targetExe := false
	if !cmd.targetIR { // -c was not set
		switch ext := filepath.Ext(cmd.outPath); ext {
		case ".ll":
			cmd.targetIR = true
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

	print("creating output directory: %s", filepath.Dir(cmd.outPath))
	// make the output file directory
	if err := os.MkdirAll(filepath.Dir(cmd.outPath), os.ModePerm); err != nil {
		return fmt.Errorf("failed to create output directory: %s", err.Error())
	}

	objPath := changeExtension(cmd.outPath, ".o")

	var to *os.File
	var err error
	if targetExe {
		to, err = os.OpenFile(objPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	} else {
		to, err = os.OpenFile(cmd.outPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	}
	if err != nil {
		return err
	}

	print("compiling ddp-source to %s", cmd.outPath)
	result, err := compiler.Compile(compiler.Options{
		FileName:                cmd.filePath,
		Source:                  nil,
		From:                    nil,
		To:                      to,
		OutputType:              compOutType,
		ErrorHandler:            errHndl,
		Log:                     print,
		DeleteIntermediateFiles: !cmd.nodeletes,
	})
	to.Close()
	if err != nil {
		return err
	}

	if !targetExe {
		return nil
	}

	if !cmd.nodeletes {
		defer func() {
			print("removing %s", objPath)
			os.Remove(objPath)
		}()
	}

	// the target is an executable so we link the produced object file
	print("linking objects")
	if err := linker.LinkDDPFiles(linker.Options{
		InputFile:               objPath,
		OutputFile:              cmd.outPath,
		Dependencies:            result,
		Log:                     print,
		DeleteIntermediateFiles: !cmd.nodeletes,
		GCCFlags:                cmd.gcc_flags,
		ExternGCCFlags:          cmd.extern_gcc_flags,
	}); err != nil {
		return fmt.Errorf("linking failed: %s", err)
	}

	return nil
}

func (cmd *BuildCommand) Name() string {
	return cmd.fs.Name()
}

func (cmd *BuildCommand) Usage() string {
	return `build <filename> <options>: build the given .ddp file into a executable
options:
		-o <filepath>: specify the name of the output file
		-verbose: print verbose output
		-c: compile to llvm ir but don't assemble or link
		-gcc_flags: custom flags that are passed to gcc`
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
		fs: flag.NewFlagSet("parse", flag.ExitOnError),
	}
}

func (cmd *ParseCommand) Init(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("parse requires a filepath")
	}

	cmd.filePath = args[0]
	if filepath.Ext(cmd.filePath) != ".ddp" {
		return fmt.Errorf("the provided file is not a .ddp file")
	}

	cmd.fs.StringVar(&cmd.outPath, "o", "", "provide a optional filepath where the output is written to")
	return cmd.fs.Parse(args[1:])
}

func (cmd *ParseCommand) Run() error {
	ast, err := parser.ParseFile(cmd.filePath, errHndl)
	if err != nil {
		return err
	}

	if ast.Faulty {
		fmt.Println("the generated ast is faulty")
	}

	if cmd.outPath != "" {
		if file, err := os.OpenFile(cmd.outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm); err != nil {
			return fmt.Errorf("unable to open the output file")
		} else {
			defer file.Close()
			if _, err := file.WriteString(ast.String()); err != nil {
				return fmt.Errorf("unable to write to the output file: %s", err.Error())
			}
		}
	} else {
		fmt.Println(ast.String())
	}

	return nil
}

func (cmd *ParseCommand) Name() string {
	return cmd.fs.Name()
}

func (cmd *ParseCommand) Usage() string {
	return `parse <filepath> <options>: parse the specified ddp file into a ddp ast
options:
	-o <filepath>: specify the name of the output file; if none is set output is written to the terminal`
}

// $kddp version provides information about the version of the used kddp build
type VersionCommand struct {
	fs         *flag.FlagSet
	verbose    bool
	build_info bool
}

func NewVersionCommand() *VersionCommand {
	return &VersionCommand{
		fs: flag.NewFlagSet("version", flag.ExitOnError),
	}
}

func (cmd *VersionCommand) Init(args []string) error {
	cmd.fs.BoolVar(&cmd.verbose, "verbose", false, "show verbose output for all versions")
	cmd.fs.BoolVar(&cmd.build_info, "build_info", false, "show go build info")
	return cmd.fs.Parse(args)
}

var (
	DDPVERSION     string = "undefined"
	LLVMVERSION    string = "undefined"
	GCCVERSION     string = "undefined"
	GCCVERSIONFULL string = "undefined"
)

func (cmd *VersionCommand) Run() error {
	fmt.Printf("ddp version: %s %s %s\n", DDPVERSION, runtime.GOOS, runtime.GOARCH)

	if bi, ok := debug.ReadBuildInfo(); ok {
		fmt.Printf("go version: %s\n", bi.GoVersion)
		if cmd.build_info {
			fmt.Printf("go build info:\n")
			for _, v := range bi.Settings {
				fmt.Printf("%s: %s\n", v.Key, v.Value)
			}
		}
	} else {
		fmt.Println("No go version available")
	}

	if cmd.verbose && GCCVERSIONFULL != "undefined" {
		fmt.Printf("gcc version: %s\n", GCCVERSIONFULL)
	} else {
		fmt.Printf("gcc version: %s\n", GCCVERSION)
	}
	fmt.Printf("llvm version: %s\n", LLVMVERSION)

	return nil
}

func (cmd *VersionCommand) Name() string {
	return cmd.fs.Name()
}

func (cmd *VersionCommand) Usage() string {
	return `version <options>: display version information for kddp
options:
	--verbose: show verbose output for all versions
	--build_info: show go build info`
}
