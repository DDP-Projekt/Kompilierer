package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/parser"
	"github.com/spf13/cobra"
)

var parseCmd = &cobra.Command{
	Use:   "parse [-o Ausgabe-Datei] <Datei>",
	Short: "Parst eine .ddp Datei in einen Abstrakten Syntaxbaum",
	Long:  `Parst eine .ddp Datei in einen Abstrakten Syntaxbaum und gibt diesen aus`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]
		if filepath.Ext(filePath) != ".ddp" {
			return fmt.Errorf("Die Eingabedatei '%s' ist keine .ddp Datei", filePath)
		}

		var src []byte
		if filePath == "" {
			var err error
			if src, err = io.ReadAll(os.Stdin); err != nil {
				return fmt.Errorf("Fehler beim Lesen von stdin: %w", err)
			}
		}
		module, err := parser.Parse(parser.Options{FileName: filePath, Source: src, ErrorHandler: ddperror.MakeBasicHandler(os.Stderr)})
		if err != nil {
			return fmt.Errorf("Fehler beim Parsen: %w", err)
		}

		if module.Ast.Faulty {
			fmt.Println("Der generierte Abstrakte Syntaxbaum ist fehlerhaft")
		}

		if parseOutputPath != "" {
			if file, err := os.OpenFile(parseOutputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm); err != nil {
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
	},
}

var parseOutputPath string // flag for parse

func init() {
	parseCmd.Flags().StringVarP(&parseOutputPath, "ausgabe", "o", "", "Optionaler Pfad zur Ausgabedatei")
}
