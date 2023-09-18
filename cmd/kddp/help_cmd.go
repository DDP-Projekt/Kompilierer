package main

import (
	"flag"
	"fmt"
)

// $kddp help prints some help information
type HelpCommand struct {
	fs  *flag.FlagSet // FlagSet for the arguments
	cmd string        // command to print information about, may be empty
}

func NewHelpCommand() *HelpCommand {
	return &HelpCommand{
		fs:  flag.NewFlagSet("hilfe", flag.ExitOnError),
		cmd: "",
	}
}

func (cmd *HelpCommand) Init(args []string) error {
	if err := parseFlagSet(cmd.fs, args); err != nil {
		return err
	}
	cmd.cmd = cmd.fs.Arg(0)
	return nil
}

func (cmd *HelpCommand) Run() error {
	// if a sub-command was specified print only its usage
	if cmd.cmd != "" {
		for _, command := range commands {
			if command.Name() == cmd.cmd {
				fmt.Println(command.Usage())
				return nil
			}
		}
	}

	// otherwise, print the usage of every command
	fmt.Println("Verfügbare Befehle:")
	for _, cmd := range commands {
		fmt.Println(cmd.Usage() + "\n")
	}

	return nil
}

func (cmd *HelpCommand) Name() string {
	return cmd.fs.Name()
}

func (cmd *HelpCommand) Usage() string {
	return `hilfe <Befehl>: Zeigt Nutzungsinformationen über den Befehl`
}
