package main

import (
	"fmt"
	"os"
	"time"
)

var startTime = time.Now()

func resetTimer() time.Duration {
	elapsed := time.Since(startTime)
	startTime = time.Now()
	return elapsed
}

func main() {
	resetTimer()
	if err := runCommands(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Printf("Took %dms\n", resetTimer().Milliseconds())
}

func runCommands() error {
	if len(os.Args) < 2 {
		return fmt.Errorf(
			`usage: ddp <command> <options>
for more information try: ddp help
`)
	}

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
