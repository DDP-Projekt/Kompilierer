package main

import (
	"fmt"
	"runtime"
	"runtime/debug"

	"github.com/spf13/cobra"
)

var versionCmd = cobra.Command{
	Use:   "version [Optionen]",
	Short: "Zeigt die Version des Kompilierers",
	Long:  `Zeigt die Version des Kompilierers, sowie weitere Informationen zu genutzten GCC, LLVM und Go Versionen`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("%s %s %s\n", DDPVERSION, runtime.GOOS, runtime.GOARCH)

		if bi, ok := debug.ReadBuildInfo(); ok {
			if verbose {
				fmt.Printf("Go Version: %s\n", bi.GoVersion)
			}

			if versionGoBuildInfo {
				fmt.Printf("Go build info:\n")
				for _, v := range bi.Settings {
					fmt.Printf("%s: %s\n", v.Key, v.Value)
				}
			}
		} else if verbose {
			fmt.Printf("Go Version: undefined\n")
		}

		if verbose {
			fmt.Printf("GCC Version: %s ; Full: %s\n", GCCVERSION, GCCVERSIONFULL)
			fmt.Printf("LLVM Version: %s\n", LLVMVERSION)
		}

		return nil
	},
}

var (
	versionGoBuildInfo bool

	DDPVERSION     string = "undefined"
	LLVMVERSION    string = "undefined"
	GCCVERSION     string = "undefined"
	GCCVERSIONFULL string = "undefined"
)

func init() {
	versionCmd.Flags().BoolVar(&versionGoBuildInfo, "go-build-info", false, "Zeige Go build Informationen")
}
