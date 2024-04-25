package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "starte [--gcc-optionen GCC-Optionen] [--externe-gcc-optionen Externe-GCC-Optionen] <Datei>",
	Short: "Kompiliert und führt die angegebene .ddp Datei aus",
	Long:  `Kompiliert und führt die angegebene .ddp Datei aus.`,
	Args:  cobra.MinimumNArgs(1),
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

		outDir, err := os.MkdirTemp("", "KDDP_RUN")
		if err != nil {
			return fmt.Errorf("Fehler beim Erstellen des temporären Ordners: %w", err)
		}
		defer os.RemoveAll(outDir)

		exePath := filepath.Join(outDir, filepath.Base(changeExtension(filePath, ".exe")))
		if filePath == "" {
			exePath = filepath.Join(outDir, "ddp.exe")
		}

		buildOutputPath = exePath
		buildGCCFlags = runGCCFlags
		buildExternGCCFlags = runExternGCCFlags

		print("Kompiliere den Quellcode")
		if err = buildCmd.RunE(buildCmd, []string{filePath}); err != nil {
			return fmt.Errorf("Fehler beim Kompilieren: %w", err)
		}

		print("Starte das Programm\n")
		if filePath != "" && len(args) > 1 {
			args = args[1:]
		}
		ddpExe := exec.Command(exePath, args...)
		ddpExe.Stdin = os.Stdin
		ddpExe.Stdout = os.Stdout
		ddpExe.Stderr = os.Stderr

		return ddpExe.Run()
	},
}

var (
	runGCCFlags       string // flag for starte
	runExternGCCFlags string // flag for starte
)

func init() {
	runCmd.Flags().StringVar(&runGCCFlags, "gcc-optionen", "", "Benutzerdefinierte Optionen, die gcc übergeben werden")
	runCmd.Flags().StringVar(&runExternGCCFlags, "externe-gcc-optionen", "", "Benutzerdefinierte Optionen, die gcc für jede externe .c Datei übergeben werden")
}
