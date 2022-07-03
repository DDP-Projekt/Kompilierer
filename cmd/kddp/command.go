// file command.go defines the sub-commands
// of kddp like build, help, etc.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

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
	NewLinkCommand(),
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
		fmt.Println(cmd.Usage())
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
	filePath  string // input file (.ddp), neccessery, first argument
	outPath   string // path for the output file, specified by the -o flag, may be empty
	nodeletes bool   // should temp files be deleted, specified by the --nodeletes flag
	verbose   bool   // print verbose output, specified by the --verbose flag
	targetIR  bool   // only compile to llvm ir, specified by the -c flag
	// some flags not specified in the commandline
	targetASM bool
	targetOBJ bool
	targetEXE bool
}

func NewBuildCommand() *BuildCommand {
	return &BuildCommand{
		fs:        flag.NewFlagSet("build", flag.ExitOnError),
		filePath:  "",
		outPath:   "",
		nodeletes: false,
		verbose:   false,
		targetIR:  false,
		targetASM: false,
		targetOBJ: false,
		targetEXE: true,
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
	extension := ".ll" // assume the -c flag
	if !cmd.targetIR { // -c was not set
		cmd.targetEXE = false
		switch ext := filepath.Ext(cmd.outPath); ext {
		case ".ll":
			cmd.targetIR = true
		case ".s", ".asm":
			extension = ext
			cmd.targetASM = true
		case ".o", ".obj":
			extension = ext
			cmd.targetOBJ = true
		case ".exe":
			extension = ext
			cmd.targetEXE = true
		case "":
			extension = ext
			cmd.targetEXE = true
		default: // by default we create a executable
			if runtime.GOOS == "windows" {
				extension = ".exe"
			} else if runtime.GOOS == "linux" {
				extension = ""
			}
			cmd.targetEXE = true
		}
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

	// temp paths (might not need them)
	llPath := changeExtension(cmd.outPath, ".ll")
	objPath := changeExtension(cmd.outPath, ".o")

	print("creating .ll output directory: %s", llPath)
	if file, err := os.OpenFile(llPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create .ll output directory: %s", err.Error())
	} else {
		if !cmd.targetIR && !cmd.nodeletes { // if the target is not llvm ir we remove the temp file
			defer func() { // defer, to remove the file after it has been used
				print("removing temporary .ll file: %s", llPath)
				if err := os.Remove(llPath); err != nil {
					print("failed to remove temporary .ll file: %s", err.Error())
				}
			}()
		}
		defer file.Close()
		print("parsing and compiling llvm ir from %s", cmd.filePath)
		// compile the input file to llvm ir
		if err := compiler.CompileTo(cmd.filePath, nil, errHndl, file); err != nil {
			return fmt.Errorf("failed to compile the source code: %s", err.Error())
		}
		if cmd.targetIR { // if the target is llvm ir we are finished
			return nil
		}
	}

	// the target was not llvm ir, so we compile it to an object-, assembly-, or executable file
	if cmd.targetEXE { // compile to object-file and continue after that
		print("compiling llir from %s to %s", llPath, objPath)
		if err := compileToObject(llPath, objPath); err != nil { // compile the object file
			return fmt.Errorf("failed to compile llir: %s", err.Error())
		}
		if !cmd.nodeletes {
			defer func() { // defer, to remove the temp file only after it has been used
				print("removing temporary .obj file: %s", objPath)
				if err := os.Remove(objPath); err != nil {
					print("failed to remove temporary .obj file: %s", err.Error())
				}
			}()
		}
	} else if cmd.targetASM || cmd.targetOBJ { // compile to assembly or object, and return
		print("compiling llir from %s to %s", llPath, cmd.outPath)
		if err := compileToObject(llPath, cmd.outPath); err != nil {
			return fmt.Errorf("failed to compile llir: %s", err.Error())
		}
		return nil
	}

	// if --verbose was specified, print gccs output
	var cmdOut io.Writer = nil
	if cmd.verbose {
		cmdOut = os.Stdout
	}

	// the target is an executable so we link the produced object file
	print("invoking gcc on %s", objPath)
	if err := invokeGCC(objPath, cmd.outPath, cmdOut); err != nil {
		return fmt.Errorf("failed to invoke gcc: %s", err.Error())
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
		-c: compile to llvm ir but don't assemble or link`
}

// $kddp link links the specified object files with the ddpstdlib
// mainly for testing
type LinkCommand struct {
	fs       *flag.FlagSet // FlagSet for the arguments
	filePath string        // input file, neccessery
	outPath  string        // path for the output file, specified by the -o flag, may be empty
	verbose  bool          // print verbose output, specified by the --verbose flag
}

func NewLinkCommand() *LinkCommand {
	return &LinkCommand{
		fs:       flag.NewFlagSet("link", flag.ExitOnError),
		filePath: "",
		outPath:  "",
		verbose:  false,
	}
}

func (cmd *LinkCommand) Init(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("link requires a file name")
	}

	cmd.filePath = args[0]
	if filepath.Ext(cmd.filePath) != ".obj" && filepath.Ext(cmd.filePath) != ".o" {
		return fmt.Errorf("the provided file is not a .obj or .o file")
	}

	cmd.fs.StringVar(&cmd.outPath, "o", "", "provide a optional filepath where the output is written to")
	cmd.fs.BoolVar(&cmd.verbose, "verbose", false, "print verbose link output")
	return cmd.fs.Parse(args[1:])
}

func (cmd *LinkCommand) Run() error {
	print := func(format string, args ...any) {
		if cmd.verbose {
			fmt.Printf(format+"\n", args...)
		}
	}

	var cmdOut io.Writer = nil
	if cmd.verbose {
		cmdOut = os.Stdout
	}

	// the target is an executable so we link the produced object file
	print("invoking gcc on %s", cmd.filePath)
	if err := invokeGCC(cmd.filePath, cmd.outPath, cmdOut); err != nil {
		print("failed to invoke gcc: %s", err.Error())
		return nil
	}

	return nil
}

func (cmd *LinkCommand) Name() string {
	return cmd.fs.Name()
}

func (cmd *LinkCommand) Usage() string {
	return `build <filename> <options>: link the given .obj or .o file into a executable expecting it to come from ddp code
options:
		-o <filepath>: specify the name of the output file
		-verbose: print verbose output`
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
