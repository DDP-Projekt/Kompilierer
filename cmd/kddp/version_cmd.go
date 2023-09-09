package main

import (
	"flag"
	"fmt"
	"runtime"
	"runtime/debug"
)

// $kddp version provides information about the version of the used kddp build
type VersionCommand struct {
	fs         *flag.FlagSet
	verbose    bool
	build_info bool
}

func NewVersionCommand() *VersionCommand {
	return &VersionCommand{
		fs:         flag.NewFlagSet("version", flag.ExitOnError),
		verbose:    false,
		build_info: false,
	}
}

func (cmd *VersionCommand) Init(args []string) error {
	cmd.fs.BoolVar(&cmd.verbose, "wortreich", cmd.verbose, "Zeige wortreiche Informationen")
	cmd.fs.BoolVar(&cmd.build_info, "go_build_info", cmd.build_info, "Zeige Go build Informationen")
	return parseFlagSet(cmd.fs, args)
}

var (
	DDPVERSION     string = "undefined"
	LLVMVERSION    string = "undefined"
	GCCVERSION     string = "undefined"
	GCCVERSIONFULL string = "undefined"
)

func (cmd *VersionCommand) Run() error {
	fmt.Printf("%s %s %s\n", DDPVERSION, runtime.GOOS, runtime.GOARCH)

	if bi, ok := debug.ReadBuildInfo(); ok {
		if cmd.verbose {
			fmt.Printf("Go Version: %s\n", bi.GoVersion)
		}

		if cmd.build_info {
			fmt.Printf("Go build info:\n")
			for _, v := range bi.Settings {
				fmt.Printf("%s: %s\n", v.Key, v.Value)
			}
		}
	} else if cmd.verbose {
		fmt.Printf("Go Version: undefined\n")
	}

	if cmd.verbose {
		fmt.Printf("GCC Version: %s ; Full: %s\n", GCCVERSION, GCCVERSIONFULL)
		fmt.Printf("LLVM Version: %s\n", LLVMVERSION)
	}

	return nil
}

func (cmd *VersionCommand) Name() string {
	return cmd.fs.Name()
}

func (cmd *VersionCommand) Usage() string {
	return `version <Optionen>: Zeige informationen zu dieser DDP Version
Optionen:
	--wortreich: Zeige wortreiche Informationen
	--go_build_info: Zeige Go build Informationen`
}
