package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/parser"
)

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
	cmd.fs.StringVar(&cmd.outPath, "o", cmd.outPath, "Optionaler Pfad der Ausgabedatei")
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

func (cmd *ParseCommand) Run() error {
	var src []byte
	if cmd.filePath == "" {
		var err error
		if src, err = io.ReadAll(os.Stdin); err != nil {
			return fmt.Errorf("Fehler beim Lesen von stdin: %w", err)
		}
	}
	module, err := parser.Parse(parser.Options{FileName: cmd.filePath, Source: src, ErrorHandler: ddperror.MakeBasicHandler(os.Stderr)})
	if err != nil {
		return fmt.Errorf("Fehler beim Parsen: %w", err)
	}

	if module.Ast.Faulty {
		fmt.Println("Der generierte Abstrakte Syntaxbaum ist fehlerhaft")
	}

	if cmd.outPath != "" {
		if file, err := os.OpenFile(cmd.outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm); err != nil {
			return fmt.Errorf("Ausgabedatei konnte nicht geöffnet werden: %w", err)
		} else {
			defer file.Close()
			if _, err := file.WriteString(module.Ast.String()); err != nil {
				return fmt.Errorf("Ausgabedatei konnte nicht beschrieben werden: %w", err)
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
