package main

import (
	"fmt"
	"os"

	"github.com/DDP-Projekt/Kompilierer/src/compiler"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/ddppath"
	"github.com/spf13/cobra"
)

var dumpListDefsCommand = &cobra.Command{
	Use:   "dump-list-defs [Optionen]",
	Short: "wird nur intern verwendet",
	Long:  `Schreibt die Definitionen der eingebauten Listen Typen in die gegebene Ausgbe Datei. Wird nur intern verwendet`,
	RunE: func(cmd *cobra.Command, args []string) error {
		outputTypes := []compiler.OutputType{}
		if asm {
			outputTypes = append(outputTypes, compiler.OutputAsm)
		}
		if llvm_ir {
			outputTypes = append(outputTypes, compiler.OutputIR)
		}
		if llvm_bc {
			outputTypes = append(outputTypes, compiler.OutputBC)
		}
		if object {
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
			file, err := os.OpenFile(changeExtension(filePath, ext), os.O_CREATE|os.O_TRUNC|os.O_RDWR, os.ModePerm)
			if err != nil {
				return fmt.Errorf("Fehler beim Öffnen der Ausgabedatei: %w", err)
			}
			defer file.Close()
			if err := compiler.DumpListDefinitions(file, outType, ddperror.MakeBasicHandler(os.Stderr)); err != nil {
				return err
			}
		}

		return nil
	},
}

var (
	filePath = ddppath.LIST_DEFS_NAME
	asm      bool
	llvm_ir  bool
	llvm_bc  bool
	object   bool
)

func init() {
	dumpListDefsCommand.Flags().StringVarP(&filePath, "output", "o", filePath, "Ausgabe Datei")
	dumpListDefsCommand.Flags().BoolVar(&asm, "asm", asm, "ob die assembly Variante ausgegeben werden soll")
	dumpListDefsCommand.Flags().BoolVar(&llvm_ir, "llvm-ir", llvm_ir, "ob die llvm ir Variante ausgegeben werden soll")
	dumpListDefsCommand.Flags().BoolVar(&llvm_bc, "llvm-bc", llvm_bc, "ob die llvm bitcode Variante ausgegeben werden soll")
	dumpListDefsCommand.Flags().BoolVar(&object, "object", object, "ob eine Objekt Datei ausgegeben werden soll")
}
