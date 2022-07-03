// entry point of kddp
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/DDP-Projekt/Kompilierer/pkg/token"
)

var startTime = time.Now() // used to measure the time that everything took (will be removed for release builds)

// sets the startTime to time.Now() and returns the time since the last call
func resetTimer() time.Duration {
	elapsed := time.Since(startTime)
	startTime = time.Now()
	return elapsed
}

func errHndl(t token.Token, msg string) {
	fmt.Printf("Fehler in %s in Zeile %d, Spalte %d: %s\n", t.File, t.Line, t.Column, msg)
}

func main() {
	//resetTimer()
	// run sub-commands like build or help
	if err := runCommands(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	//fmt.Printf("Took %dms\n", resetTimer().Milliseconds())
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

	return fmt.Errorf("Unknown command '%s'", subcmd)
}
