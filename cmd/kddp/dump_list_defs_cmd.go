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
	Use:    "dump-list-defs [-o <Ausgabe Datei>] [--asm] [--llvm-ir] [--llvm-bc] [--object]",
	Short:  "wird nur intern verwendet",
	Long:   `Schreibt die Definitionen der eingebauten Listen Typen in die gegebene Ausgbe Datei. Wird nur intern verwendet`,
	Hidden: true, // only used internally
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
			file, err := os.OpenFile(changeExtension(listDefsOutputPath, ext), os.O_CREATE|os.O_TRUNC|os.O_RDWR, os.ModePerm)
			if err != nil {
				return fmt.Errorf("Fehler beim Öffnen der Ausgabedatei: %w", err)
			}
			defer file.Close()
			if err := compiler.DumpListDefinitions(file, outType, ddperror.MakeBasicHandler(os.Stderr), listDefsOptimizationLevel); err != nil {
				return err
			}
		}

		return nil
	},
}

var (
	listDefsOutputPath        = ddppath.LIST_DEFS_NAME // flag for dump-list-defs
	asm                       bool                     // flag for dump-list-defs
	llvm_ir                   bool                     // flag for dump-list-defs
	llvm_bc                   bool                     // flag for dump-list-defs
	object                    bool                     // flag for dump-list-defs
	listDefsOptimizationLevel uint                     // flag for dump-list-defs
)

func init() {
	dumpListDefsCommand.Flags().StringVarP(&listDefsOutputPath, "ausgabe", "o", listDefsOutputPath, "Ausgaben Datei Muster (ohne Dateiendung)")
	dumpListDefsCommand.Flags().BoolVar(&asm, "asm", asm, "ob die assembly Variante ausgegeben werden soll")
	dumpListDefsCommand.Flags().BoolVar(&llvm_ir, "llvm-ir", llvm_ir, "ob die llvm ir Variante ausgegeben werden soll")
	dumpListDefsCommand.Flags().BoolVar(&llvm_bc, "llvm-bc", llvm_bc, "ob die llvm bitcode Variante ausgegeben werden soll")
	dumpListDefsCommand.Flags().BoolVar(&object, "object", object, "ob eine Objekt Datei ausgegeben werden soll")
	dumpListDefsCommand.Flags().UintVarP(&listDefsOptimizationLevel, "optimierungs-stufe", "O", 1, "Menge und Art der Optimierungen, die angewandt werden")
}
