// entry point of kddp
package main

import (
	"fmt"
	"os"
)

func main() {
	// run sub-commands like build or help
	if err := runCommands(); err != nil {
		if err.Error() != "" {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		}
		os.Exit(1)
	}
}

func runCommands() error {
	// if no sub-command is provided, error
	if len(os.Args) < 2 {
		return fmt.Errorf(
			`Nutzung: kddp <Befehl> <Optionen>
Für mehr Informationen probiere $kddp hilfe
`)
	}

	// run the specified sub-command
	subcmd := os.Args[1]
	for _, cmd := range commands {
		if cmd.Name() == subcmd {
			if err := cmd.Init(os.Args[2:]); err != nil {
				return err
			}
			return cmd.Run()
		}
	}

	return fmt.Errorf("Unbekannter Befehl '%s'\nFür eine Liste aller Befehle probiere $kddp hilfe", subcmd)
}
