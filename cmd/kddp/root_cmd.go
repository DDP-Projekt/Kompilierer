package main

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const rootUsage = `kddp <Befehl> [Optionen] [Argumente]`

// rootCmd is the root command of the kddp compiler
var rootCmd = &cobra.Command{
	Use:     rootUsage,
	Short:   "DDP Kompilierer",
	Long:    `Der Kompilierer der deutschen Programmiersprache (DDP)`,
	Version: fmt.Sprintf("%s %s %s\n", DDPVERSION, runtime.GOOS, runtime.GOARCH),
}

// global verbose flag for all commands
var verbose bool

func init() {
	// Flags inherited by all sub commands
	rootCmd.PersistentFlags().BoolVarP(&verbose, "wortreich", "w", false, "Gibt wortreiche Informationen aus")

	// sub commands
	rootCmd.AddCommand(
		versionCmd,
		dumpListDefsCommand,
		parseCmd,
		buildCmd,
		runCmd,
	)

	setDefaultCommandOptions(rootCmd)

	// make the default help command german
	rootCmd.InitDefaultHelpCmd()
	for _, cmd := range rootCmd.Commands() {
		// make the default help command german
		if cmd.Name() == "help" {
			cmd.Use = "hilfe [Befehl]"
			cmd.Short = "Zeigt Informationen zu einem Befehl"
			cmd.Long = `Hilfe zeigt Informationen für jeden Befehl des Kompilierers.
Gib einfach ` + rootCmd.Name() + ` hilfe [Pfad zum Befehl] für vollständige Informationen ein.`
			rootCmd.RemoveCommand(cmd)
			rootCmd.AddCommand(cmd)
			rootCmd.SetHelpCommand(cmd)
		}

		setDefaultCommandOptions(cmd)
	}

	// make the default help topic command german
	rootCmd.SetUsageTemplate(`Nutzung:
  {{.UseLine}}{{if gt (len .Aliases) 0}}

Aliase:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Beispiele:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}

Verfügbare Befehle:{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}

{{.Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

Weitere Befehle:{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Optionen:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Globale Optionen:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Weitere Hilfe Themen:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Probiere "{{.CommandPath}} hilfe <Befehl>" oder "{{.CommandPath}} <Befehl> -h|--hilfe" für mehr Informationen zu einem Befehl.{{end}}
`)
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

func normalizeHelpFlagGerman(f *pflag.FlagSet, name string) pflag.NormalizedName {
	// for backwards compatibility, replace underscores with dashes
	name = strings.ReplaceAll(name, "_", "-")

	switch name {
	case "hilfe":
		return "help"
	}
	return pflag.NormalizedName(name)
}

func germanizeHelpFlag(cmd *cobra.Command) {
	// make the default help flag german
	cmd.InitDefaultHelpFlag()
	cmd.Flags().Lookup("help").Name = "hilfe"
	cmd.Flags().Lookup("help").Usage = "Zeigt Informationen zum Befehl"
	cmd.Flags().SetNormalizeFunc(normalizeHelpFlagGerman)
}

func germanizeVersionFlag(cmd *cobra.Command) {
	// make the default version flag german
	cmd.InitDefaultVersionFlag()
	if versionFlag := cmd.Flags().Lookup("version"); versionFlag != nil {
		versionFlag.Usage = "Zeigt die Version des Kompilierers"
	}
}

func setDefaultCommandOptions(cmd *cobra.Command) {
	cmd.DisableFlagsInUseLine = true
	cmd.SetErrPrefix("Fehler:")
	// make the default help flag german
	germanizeHelpFlag(cmd)
	germanizeVersionFlag(cmd)
}
