// file command.go defines the sub-commands
// of kddp like build, help, etc.
package main

import (
	"flag"
)

// https://stackoverflow.com/a/74146375
// ParseFlagSet works like flagset.Parse(), except positional arguments are not
// required to come after flag arguments.
func parseFlagSet(flagset *flag.FlagSet, args []string) error {
	var positionalArgs []string
	for {
		if err := flagset.Parse(args); err != nil {
			return err
		}
		// Consume all the flags that were parsed as flags.
		args = args[len(args)-flagset.NArg():]
		if len(args) == 0 {
			break
		}
		// There's at least one flag remaining and it must be a positional arg since
		// we consumed all args that were parsed as flags. Consume just the first
		// one, and retry parsing, since subsequent args may be flags.
		positionalArgs = append(positionalArgs, args[0])
		args = args[1:]
	}
	// Parse just the positional args so that flagset.Args()/flagset.NArgs()
	// return the expected value.
	// Note: This should never return an error.
	return flagset.Parse(positionalArgs)
}

// interface for a sub-command
type Command interface {
	Init([]string) error // initialize the command and parse it's flags
	Run() error          // run the command
	Name() string        // every command must specify a name
	Usage() string       // and a usage
}

var commands = []Command{
	NewHelpCommand(),
	NewBuildCommand(),
	NewParseCommand(),
	NewVersionCommand(),
	NewRunCommand(),
	NewDumpListDefsCommand(),
}
