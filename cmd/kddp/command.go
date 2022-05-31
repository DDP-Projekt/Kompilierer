package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/Die-Deutsche-Programmiersprache/KDDP/pkg/compiler"
	"github.com/Die-Deutsche-Programmiersprache/KDDP/pkg/interpreter"
	"github.com/Die-Deutsche-Programmiersprache/KDDP/pkg/parser"
)

type Command interface {
	Init([]string) error
	Run() error
	Name() string
	Usage() string
}

var commands = []Command{
	NewHelpCommand(),
	NewInterpretCommand(),
	NewBuildCommand(),
	NewParseCommand(),
	NewLinkCommand(),
}

type InterpretCommand struct {
	fs       *flag.FlagSet
	filePath string
	outPath  string // may be empty
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

	cmd.filePath = args[0]
	if filepath.Ext(cmd.filePath) != ".ddp" {
		return fmt.Errorf("the provided file is not a .ddp file")
	}

	cmd.fs.StringVar(&cmd.outPath, "o", "", "provide a optional filepath where the output is written to")
	return cmd.fs.Parse(args[1:])
}

func (cmd *InterpretCommand) Run() error {
	errHndl := func(msg string) { fmt.Println(msg) }

	ast, err := parser.ParseFile(cmd.filePath, errHndl)
	if err != nil {
		return err
	}

	interpreter := interpreter.New(ast, errHndl)

	if cmd.outPath != "" {
		file, err := os.OpenFile(cmd.outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
		if err != nil {
			return fmt.Errorf("unable to open the output file")
		}
		interpreter.Stdout = file
	}

	return interpreter.Interpret()
}

func (cmd *InterpretCommand) Name() string {
	return cmd.fs.Name()
}

func (cmd *InterpretCommand) Usage() string {
	return `interpret <filename> <options>: interprets the given .ddp file
options:
	-o <filepath>: writes the programs stdout to the given file`
}

type HelpCommand struct {
	fs  *flag.FlagSet
	cmd string
}

func NewHelpCommand() *HelpCommand {
	return &HelpCommand{
		fs: flag.NewFlagSet("help", flag.ExitOnError),
	}
}

func (cmd *HelpCommand) Init(args []string) error {
	if len(args) > 0 {
		cmd.cmd = args[0]
	}
	return cmd.fs.Parse(args)
}

func (cmd *HelpCommand) Run() error {
	if cmd.cmd != "" {
		for _, command := range commands {
			if command.Name() == cmd.cmd {
				fmt.Println(command.Usage())
				return nil
			}
		}
	}

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

type BuildCommand struct {
	fs        *flag.FlagSet
	filePath  string // input file, neccessery
	outPath   string // path for the output file, may be empty
	nodeletes bool   // should temp files be deleted
	verbose   bool   // print verbose output
	targetIR  bool   // only compile to llvm ir
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
	if len(args) < 1 {
		return fmt.Errorf("build requires a file name")
	}

	cmd.filePath = args[0]
	if filepath.Ext(cmd.filePath) != ".ddp" {
		return fmt.Errorf("the provided file is not a .ddp file")
	}

	cmd.fs.StringVar(&cmd.outPath, "o", "", "provide a optional filepath where the output is written to")
	cmd.fs.BoolVar(&cmd.nodeletes, "nodeletes", false, "don't delete temporary files such as .ll or .obj files")
	cmd.fs.BoolVar(&cmd.verbose, "verbose", false, "print verbose build output")
	cmd.fs.BoolVar(&cmd.targetIR, "c", false, "only produce llvm ir")
	return cmd.fs.Parse(args[1:])
}

func (cmd *BuildCommand) Run() error {
	print := func(format string, args ...any) {
		if cmd.verbose {
			fmt.Printf(format+"\n", args...)
		}
	}

	// determine the final output
	extension := ".ll"
	if !cmd.targetIR {
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
		default:
			if runtime.GOOS == "windows" {
				extension = ".exe"
			} else if runtime.GOOS == "linux" {
				extension = ""
			}
			cmd.targetEXE = true
		}
	}

	// create the path to the output file
	if cmd.outPath == "" {
		cmd.outPath = changeExtension(cmd.filePath, extension)
	} else {
		cmd.outPath = changeExtension(cmd.outPath, extension)
	}

	print("creating output directory: %s", filepath.Dir(cmd.outPath))
	// make the output file directory
	if err := os.MkdirAll(filepath.Dir(cmd.outPath), os.ModePerm); err != nil {
		print("failed to create output directory: %s", err.Error())
		return nil
	}

	// temp paths (might not need them)
	llPath := changeExtension(cmd.outPath, ".ll")
	objPath := changeExtension(cmd.outPath, ".obj")

	print("creating .ll output directory: %s", llPath)
	if file, err := os.OpenFile(llPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm); err != nil {
		print("failed to create .ll output directory: %s", err.Error())
		return nil
	} else {
		if !cmd.targetIR && !cmd.nodeletes { // if the target is not llvm ir we remove the temp file
			defer func() {
				print("removing temporary .ll file: %s", llPath)
				if err := os.Remove(llPath); err != nil {
					print("failed to remove temporary .ll file: %s", err.Error())
				}
			}()
		}
		defer file.Close()
		print("parsing and compiling llvm ir from %s", cmd.filePath)
		if ir, err := compiler.CompileFile(cmd.filePath, func(msg string) { fmt.Println(msg) }); err != nil {
			print("failed to compile the source code: %s", err.Error())
			return nil
		} else {
			print("writing llvm ir to %s", llPath)
			_, err = file.WriteString(ir)
			if err != nil {
				print("failed to write llvm ir: %s", err.Error())
				return nil
			}
		}
		if cmd.targetIR { // if the target is llvm ir we are finished
			return nil
		}
	}

	// the target was not llvm ir, so we invoke llc
	if cmd.targetEXE {
		print("compiling llir from %s to %s", llPath, objPath)
		if err := compileToObject(llPath, objPath); err != nil {
			print("failed to compile llir: %s", err.Error())
			return nil
		}
		if !cmd.nodeletes {
			defer func() {
				print("removing temporary .obj file: %s", objPath)
				if err := os.Remove(objPath); err != nil {
					print("failed to remove temporary .obj file: %s", err.Error())
				}
			}()
		}
	} else if cmd.targetASM || cmd.targetOBJ {
		print("compiling llir from %s to %s", llPath, cmd.outPath)
		if err := compileToObject(llPath, cmd.outPath); err != nil {
			print("failed to compile llir: %s", err.Error())
			return nil
		}
		return nil
	}

	var cmdOut io.Writer = nil
	if cmd.verbose {
		cmdOut = os.Stdout
	}

	// the target is an executable so we link the produced object file
	print("invoking gcc on %s", objPath)
	if err := invokeGCC(objPath, cmd.outPath, cmdOut); err != nil {
		print("failed to invoke gcc: %s", err.Error())
		return nil
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

type LinkCommand struct {
	fs       *flag.FlagSet
	filePath string // input file, neccessery
	outPath  string // path for the output file, may be empty
	verbose  bool   // print verbose output
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

func changeExtension(path, ext string) string {
	return path[:len(path)-len(filepath.Ext(path))] + ext
}

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
	errHndl := func(msg string) { fmt.Println(msg) }

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
