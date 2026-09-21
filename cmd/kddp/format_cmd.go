package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	formatierer "github.com/DDP-Projekt/Formatierer"
	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ast/annotators"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/parser"
	"github.com/spf13/cobra"
)

var formatCmd = &cobra.Command{
	Use:   "formatiere [--leerzeichen] <Datei>",
	Short: "Formatiert eine .ddp Datei",
	Long:  `Formatiert eine .ddp Datei`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]
		if filepath.Ext(filePath) != ".ddp" {
			return fmt.Errorf("Die Eingabedatei '%s' ist keine .ddp Datei", filePath)
		}

		file, err := os.OpenFile(filePath, os.O_RDONLY, os.ModePerm)
		defer file.Close()

		if err != nil {
			return fmt.Errorf("Ausgabedatei konnte nicht geöffnet werden: %w", err)
		}

		src, err := io.ReadAll(file)
		if err != nil {
			return fmt.Errorf("Ausgabedatei konnte nicht gelesen werden: %w", err)
		}

		module, err := parser.Parse(parser.Options{
			FileName:     filePath,
			Source:       src,
			ErrorHandler: ddperror.MakeBasicHandler(os.Stderr),
			Annotators: []ast.Annotator{
				&annotators.ConstFuncParamAnnotator{},
			},
		})
		if err != nil {
			return fmt.Errorf("Fehler beim Parsen: %w", err)
		}

		file.Close()
		if file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_TRUNC, os.ModePerm); err != nil {
			return fmt.Errorf("Fehler beim Öffnen: %w", err)
		} else {
			err = formatierer.WriteFormattedDocument(file, string(src), module, formatierer.FormattingOptions{
				InsertSpaces: spaces,
			})
			if err != nil {
				return fmt.Errorf("Fehler beim Formatieren: %w", err)
			}
		}

		return nil
	},
}

var spaces bool

func init() {
	formatCmd.Flags().BoolVar(&spaces, "leerzeichen", false, "Ob Leerzeichen oder Tabs für die Einrückung benutzt werden soll")
}
