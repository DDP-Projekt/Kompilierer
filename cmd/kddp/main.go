// entry point of kddp
package main

import (
	"fmt"
	"os"
)

func main() {
	// catch panics and instead set the returned error
	defer handle_panics()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func handle_panics() {
	if err := recover(); err != nil {
		fmt.Fprintf(os.Stderr,
			`Unerwarteter Fehler: %s
Dieser Fehler ist vermutlich ein Bug im DDP-Kompilierer.
Bitte erstelle einen Issue unter https://github.com/DDP-Projekt/Kompilierer oder melde ihn anderweitig den Entwicklern.
`,
			err)
		os.Exit(1)
	}
}
