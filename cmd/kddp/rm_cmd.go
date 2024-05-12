package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:   "rm <Pfad>...",
	Short: "Löscht Dateien oder Verzeichnisse",
	Long:  `Löscht Dateien oder Verzeichnisse`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		for _, path := range args {
			if err := os.Remove(path); err != nil && !rmForce {
				err = errors.Join(err, fmt.Errorf("Fehler beim Löschen von %s: %w", path, err))
				fmt.Fprintf(os.Stderr, "%s\n", err)
			}
		}
		return err
	},
}

var rmForce bool // flag for rm

func init() {
	rmCmd.Flags().BoolVarP(&rmForce, "force", "f", false, "Ignoriere Fehler beim Löschen und fahre fort")
}
