package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/google/go-github/v55/github"
)

// $kddp version provides information about the version of the used kddp build
type UpdateCommand struct {
	fs              *flag.FlagSet
	verbose         bool
	compare_version bool
	pre_release     bool
	gh              *github.Client
}

func NewUpdateCommand() *UpdateCommand {
	return &UpdateCommand{
		fs:              flag.NewFlagSet("update", flag.ExitOnError),
		verbose:         false,
		compare_version: false,
		pre_release:     false,
	}
}

func (cmd *UpdateCommand) Init(args []string) error {
	cmd.gh = github.NewClient(nil)

	cmd.fs.BoolVar(&cmd.verbose, "wortreich", cmd.verbose, "Zeige wortreiche Informationen")
	cmd.fs.BoolVar(&cmd.compare_version, "vergleiche_version", cmd.compare_version, "Vergleicht nur die installierte Version mit der neuesten Version ohne zu updaten")
	cmd.fs.BoolVar(&cmd.pre_release, "pre_release", cmd.pre_release, "pre-release Versionen mit einbeziehen")
	return parseFlagSet(cmd.fs, args)
}

func (cmd *UpdateCommand) Run() error {
	fmt.Printf("\nAktuelle Version: %s\n", DDPVERSION)
	latestRelease, err := cmd.getLatestRelease(cmd.pre_release)
	if err != nil {
		return err
	}
	fmt.Printf("Neueste Version:  %s\n\n", latestRelease.GetTagName())

	if is_newer, err := is_newer_version(DDPVERSION, latestRelease.GetTagName()); err != nil {
		return err
	} else if is_newer {
		fmt.Println("Es ist eine neuere Version verfügbar")
	} else {
		fmt.Println("DDP ist auf dem neusten Stand")
	}
	if cmd.compare_version { // early return if only the version should be compared
		return nil
	}

	return nil
}

func (cmd *UpdateCommand) Name() string {
	return cmd.fs.Name()
}

func (cmd *UpdateCommand) Usage() string {
	return `update <Optionen>: Updatet KDDP auf die neueste Version oder gibt aus ob eine neue Version verfügbar ist
Optionen:
	--wortreich: Zeige wortreiche Informationen
	--vergleiche_version: Vergleiche die installierte Version mit der neuesten Version
	--pre_release: pre-release Versionen mit einbeziehen`
}

const version_fmt = "v%d.%d.%d-%s"

// returns wether a is a newer version than b
func is_newer_version(a, b string) (bool, error) {
	parse_version := func(version string) (major, minor, patch int, additional string, err error) {
		var n int
		if n, err = fmt.Sscanf(version, version_fmt, &major, &minor, &patch, &additional); n < 3 {
			err = fmt.Errorf("ungültiges Versions Format: %s", version)
		}
		if additional != "" && additional != "pre" && additional != "alpha" && additional != "beta" {
			err = fmt.Errorf("ungültiges Versions Format: %s", version)
		}
		return
	}

	a_major, a_minor, a_patch, a_additional, err := parse_version(a)
	if err != nil {
		return false, err
	}
	b_major, b_minor, b_patch, b_additional, err := parse_version(b)
	if err != nil {
		return false, err
	}
	switch a_additional {
	case "pre":
		if b_additional != "pre" {
			return false, nil
		}
	case "alpha":
		if b_additional == "pre" {
			return true, nil
		}
	case "beta":
		if b_additional == "pre" || b_additional == "alpha" {
			return true, nil
		}
	default:
		if b_additional != "" {
			return true, nil
		}
	}
	return a_major > b_major || a_minor > b_minor || a_patch > b_patch, nil
}

// returns the latest version of kddp by checking the github releases
// pass true to also check pre-releases
func (cmd *UpdateCommand) getLatestRelease(pre_release bool) (*github.RepositoryRelease, error) {
	if !pre_release {
		release, _, err := cmd.gh.Repositories.GetLatestRelease(context.Background(), "DDP-Projekt", "Kompilierer")
		return release, err
	}
	releases, _, err := cmd.gh.Repositories.ListReleases(context.Background(), "DDP-Projekt", "Kompilierer", &github.ListOptions{})
	if err != nil {
		return nil, err
	}
	if len(releases) == 0 {
		return nil, fmt.Errorf("no releases found")
	}
	latestRelease := releases[0]
	for _, release := range releases[1:] {
		if !release.GetDraft() && release.GetPublishedAt().After(latestRelease.GetPublishedAt().Time) {
			latestRelease = release
		}
	}
	return latestRelease, nil
}
