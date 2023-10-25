package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

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
	// set all the flags
	cmd.fs.StringVar(&cmd.gcc_flags, "gcc_optionen", cmd.gcc_flags, "Benutzerdefinierte Optionen, die gcc übergeben werden")
	cmd.fs.StringVar(&cmd.extern_gcc_flags, "externe_gcc_optionen", cmd.extern_gcc_flags, "Benutzerdefinierte Optionen, die gcc für jede externe .c Datei übergeben werden")
	cmd.fs.BoolVar(&cmd.verbose, "wortreich", cmd.verbose, "Gibt wortreiche Informationen während des Befehls")
	if err := parseFlagSet(cmd.fs, args); err != nil {
		return err
	}
	if cmd.fs.NArg() >= 1 {
		cmd.filePath = cmd.fs.Arg(0)
		if filepath.Ext(cmd.filePath) != ".ddp" {
			return fmt.Errorf("Die Eingabedatei ist keine .ddp Datei")
		}
	}
	return nil
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
		return fmt.Errorf("Fehler beim Erstellen des temporären Ordners: %w", err)
	}
	defer os.RemoveAll(outDir)

	exePath := filepath.Join(outDir, filepath.Base(changeExtension(cmd.filePath, ".exe")))
	if cmd.filePath == "" {
		exePath = filepath.Join(outDir, "ddp.exe")
	}

	buildCmd := NewBuildCommand()
	buildCmd.filePath = cmd.filePath
	buildCmd.outPath = exePath
	buildCmd.verbose = cmd.verbose
	buildCmd.gcc_flags = cmd.gcc_flags
	buildCmd.extern_gcc_flags = cmd.extern_gcc_flags

	print("Kompiliere den Quellcode")
	if err = buildCmd.Run(); err != nil {
		return fmt.Errorf("Fehler beim Kompilieren: %w", err)
	}

	print("Starte das Programm\n")
	args := cmd.fs.Args()
	if cmd.filePath != "" {
		args = args[1:]
	}
	ddpExe := exec.Command(exePath, args...)
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
