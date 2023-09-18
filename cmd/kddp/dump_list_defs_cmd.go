package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/DDP-Projekt/Kompilierer/src/compiler"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/ddppath"
)

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
	return parseFlagSet(cmd.fs, args)
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
			return fmt.Errorf("Fehler beim Öffnen der Ausgabedatei: %w", err)
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
