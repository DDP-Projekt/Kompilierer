// entry point of kddp
package main

import (
	"fmt"
	"os"

	"github.com/DDP-Projekt/Kompilierer/pkg/token"
)

func errHndl(t token.Token, msg string) {
	fmt.Printf("Fehler in %s in Zeile %d, Spalte %d: %s\n", t.File, t.Line(), t.Column(), msg)
}

func main() {
	// run sub-commands like build or help
	if err := runCommands(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func runCommands() error {
	// if no sub-command is provided, error
	if len(os.Args) < 2 {
		return fmt.Errorf(
			`usage: kddp <command> <options>
for more information try: kddp help
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

	return fmt.Errorf("Unknown command '%s'\nFor a list of all commands try: kddp help", subcmd)
}
