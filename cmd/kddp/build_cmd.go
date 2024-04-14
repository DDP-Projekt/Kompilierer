package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/DDP-Projekt/Kompilierer/cmd/internal/gcc"
	"github.com/DDP-Projekt/Kompilierer/cmd/internal/linker"
	"github.com/DDP-Projekt/Kompilierer/src/compiler"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "kompiliere [-o Ausgabe-Datei [--main main.o] [--gcc-flags GCC-Flags] [--extern-gcc-flags Externe-GCC-Flags] [--nodeletes] [--verbose] [--link-modules] [--link-list-defs] [--gcc-executable Pfad-zu-GCC>] <Datei>",
	Short: "Kompiliert eine .ddp Datei",
	Long:  `Kompiliert eine .ddp Datei in eine ausführbare, llvm oder objekt Datei.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := ""
		if len(args) > 0 {
			filePath = args[0]
			if filepath.Ext(filePath) != ".ddp" {
				return fmt.Errorf("Die Eingabedatei '%s' ist keine .ddp Datei", filePath)
			}
		}

		// helper function to print verbose output if the flag was set
		print := func(format string, args ...any) {
			if verbose {
				fmt.Printf(format+"\n", args...)
			}
		}

		// set the gcc executable
		gcc.SetGcc(buildGCCExecutable)

		// determine the final output type (by file extension)
		compOutType := compiler.OutputIR
		extension := ".ll" // assume the -c flag
		targetExe := false // -c was not set
		switch ext := filepath.Ext(buildOutputPath); ext {
		case ".ll":
			compOutType = compiler.OutputIR
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

		// disable comments if the .ll files are deleted anyways
		if compOutType != compiler.OutputIR && !buildNoDeletes {
			compiler.Comments_Enabled = false
		}

		// create the path to the output file
		if buildOutputPath == "" { // if no output file was specified, we use the name of the input .ddp file
			if filePath == "" {
				buildOutputPath = "ddp" + extension // if no input file was specified, we use the default name "ddp"
			} else {
				buildOutputPath = changeExtension(filePath, extension)
			}
		} else { // otherwise we use the provided name
			buildOutputPath = changeExtension(buildOutputPath, extension)
		}

		print("Erstelle Ausgabeordner: %s", filepath.Dir(buildOutputPath))
		// make the output file directory
		if err := os.MkdirAll(filepath.Dir(buildOutputPath), os.ModePerm); err != nil {
			return fmt.Errorf("Fehler beim Erstellen des Ausgabeordners: %w", err)
		}

		objPath := changeExtension(buildOutputPath, ".o")

		var (
			to  *os.File
			err error
		)
		if targetExe {
			to, err = os.OpenFile(objPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
			if !buildNoDeletes {
				defer func() {
					print("Lösche %s", objPath)
					os.Remove(objPath)
				}()
			}
		} else {
			to, err = os.OpenFile(buildOutputPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
		}
		if err != nil {
			return err
		}
		defer to.Close()

		var src []byte
		// if no input file was specified, we read from stdin
		if filePath == "" {
			filePath = "stdin"
			if src, err = io.ReadAll(os.Stdin); err != nil {
				return fmt.Errorf("Fehler beim Lesen von stdin: %w", err)
			}
		} else {
			if src, err = os.ReadFile(filePath); err != nil {
				return fmt.Errorf("Fehler beim Lesen von %s: %w", filePath, err)
			}
		}

		errorHandler := ddperror.MakeAdvancedHandler(filePath, src, os.Stderr)

		print("Kompiliere DDP-Quellcode nach %s", buildOutputPath)
		result, err := compiler.Compile(compiler.Options{
			FileName:                filePath,
			Source:                  src,
			From:                    nil,
			To:                      to,
			OutputType:              compOutType,
			ErrorHandler:            errorHandler,
			Log:                     print,
			DeleteIntermediateFiles: !buildNoDeletes,
			LinkInModules:           buildLinkModules,
			LinkInListDefs:          buildLinkListDefs,
			OptimizationLevel:       buildOptimizationLevel,
		})
		if err != nil {
			return fmt.Errorf("Fehler beim Kompilieren: %w", err)
		}

		if !targetExe {
			return nil
		}

		// the target is an executable so we link the produced object file
		print("Objekte werden gelinkt")
		if output, err := linker.LinkDDPFiles(linker.Options{
			InputFile:               objPath,
			OutputFile:              buildOutputPath,
			Dependencies:            result,
			Log:                     print,
			DeleteIntermediateFiles: !buildNoDeletes,
			GCCFlags:                buildGCCFlags,
			MainFile:                buildMainPath,
			ExternGCCFlags:          buildExternGCCFlags,
			LinkInListDefs:          !buildLinkListDefs, // if they are allready linked in, don't link them again
		}); err != nil {
			return fmt.Errorf("Fehler beim Linken: %w (%s)", err, string(output))
		}
		return nil
	},
}

var (
	buildOutputPath        string // flag for kompiliere
	buildMainPath          string // flag for kompiliere
	buildGCCFlags          string // flag for kompiliere
	buildExternGCCFlags    string // flag for kompiliere
	buildNoDeletes         bool   // flag for kompiliere
	buildLinkModules       bool   // flag for kompiliere
	buildLinkListDefs      bool   // flag for kompiliere
	buildGCCExecutable     string // flag for kompiliere
	buildOptimizationLevel uint   // flag for kompiliere
)

func init() {
	buildCmd.Flags().StringVarP(&buildOutputPath, "ausgabe", "o", "", "Optionaler Pfad der Ausgabedatei (.exe, .ll, .o, .obj, .s, .asm).")
	buildCmd.Flags().StringVar(&buildMainPath, "main", "", "Optionaler Pfad zur main.o Datei")
	buildCmd.Flags().StringVar(&buildGCCFlags, "gcc-optionen", "", "Benutzerdefinierte Optionen, die gcc übergeben werden")
	buildCmd.Flags().StringVar(&buildExternGCCFlags, "externe-gcc-optionen", "", "Benutzerdefinierte Optionen, die gcc für jede externe .c Datei übergeben werden")
	buildCmd.Flags().BoolVar(&buildNoDeletes, "nichts-loeschen", false, "Keine temporären Dateien löschen")
	buildCmd.Flags().BoolVar(&buildLinkModules, "module-linken", true, "Ob alle Module in das Hauptmodul gelinkt werden sollen")
	buildCmd.Flags().BoolVar(&buildLinkListDefs, "list-defs-linken", true, "Ob die eingebauten Listen Definitionen in das Hauptmodul gelinkt werden sollen")
	buildCmd.Flags().StringVar(&buildGCCExecutable, "gcc-executable", gcc.Cmd(), "Pfad zur gcc executable, die genutzt werden soll")
	buildCmd.Flags().UintVarP(&buildOptimizationLevel, "optimierungs-stufe", "O", 1, "Menge und Art der Optimierungen, die angewandt werden")
}

// helper function
// returns path with the specified extension
func changeExtension(path, ext string) string {
	return path[:len(path)-len(filepath.Ext(path))] + ext
}
