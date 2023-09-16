package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/DDP-Projekt/Kompilierer/cmd/internal/archive_reader"
	"github.com/DDP-Projekt/Kompilierer/cmd/internal/gcc"
	"github.com/DDP-Projekt/Kompilierer/src/ddppath"
	"github.com/google/go-github/v55/github"
	"github.com/minio/selfupdate"
	cp "github.com/otiai10/copy"
)

// $kddp version provides information about the version of the used kddp build
type UpdateCommand struct {
	fs              *flag.FlagSet
	verbose         bool
	compare_version bool
	pre_release     bool
	use_archive     string // only meant for internal use
	now             bool
	gh              *github.Client
	infof           func(format string, a ...any)
}

func NewUpdateCommand() *UpdateCommand {
	cmd := &UpdateCommand{
		fs:              flag.NewFlagSet("update", flag.ExitOnError),
		verbose:         false,
		compare_version: false,
		pre_release:     false,
		use_archive:     "",
		now:             false,
		gh:              github.NewClient(nil),
	}
	cmd.infof = func(format string, a ...any) {
		if cmd.verbose {
			fmt.Printf(format+"\n", a...)
		}
	}
	return cmd
}

func (cmd *UpdateCommand) Init(args []string) error {
	cmd.fs.BoolVar(&cmd.verbose, "wortreich", cmd.verbose, "Zeige wortreiche Informationen")
	cmd.fs.BoolVar(&cmd.compare_version, "vergleiche_version", cmd.compare_version, "Vergleicht nur die installierte Version mit der neuesten Version ohne zu updaten")
	cmd.fs.BoolVar(&cmd.pre_release, "pre_release", cmd.pre_release, "pre-release Versionen mit einbeziehen")
	cmd.fs.StringVar(&cmd.use_archive, "use_archive", cmd.use_archive, "Nur für interne Zwecke")
	cmd.fs.BoolVar(&cmd.now, "jetzt", cmd.now, "Falls eine neue Version verfügbar ist, wird diese ohne zu fragen sofort heruntergeladen und installiert")
	return parseFlagSet(cmd.fs, args)
}

var kddp_bin_name = exePath("kddp")

func (cmd *UpdateCommand) Run() error {
	is_sub_process := cmd.use_archive != ""

	if !is_sub_process {
		fmt.Printf("Aktuelle Version: %s\n", DDPVERSION)
		latestRelease, err := cmd.getLatestRelease(cmd.pre_release)
		if err != nil {
			return fmt.Errorf("Fehler beim Abrufen der neuesten Version: %w", err)
		}
		latest_version := latestRelease.GetTagName()
		fmt.Printf("Neueste Version:  %s\n\n", latest_version)

		if is_newer, err := is_newer_version(latest_version, DDPVERSION); err != nil {
			return err
		} else if is_newer {
			fmt.Println("Es ist eine neuere Version verfügbar")
		} else {
			fmt.Println("DDP ist auf dem neusten Stand")
			return nil
		}
		if cmd.compare_version { // early return if only the version should be compared
			return nil
		}

		fmt.Printf("DDP jetzt updaten? (j/n): ")
		var answer string
		if _, err := fmt.Scanln(&answer); err != nil {
			return fmt.Errorf("Fehler beim Lesen der Eingabe: %w", err)
		}
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "j" && answer != "y" {
			return nil
		}

		archive_type := ".zip"
		if runtime.GOOS == "linux" {
			archive_type = ".tar.gz"
		}
		cmd.use_archive = fmt.Sprintf("DDP-%s-%s-%s", latest_version, runtime.GOOS, runtime.GOARCH)
		if runtime.GOOS == "windows" {
			cmd.use_archive += "-no-mingw"
		}
		cmd.use_archive += archive_type
		cmd.use_archive = filepath.Join(ddppath.InstallDir, cmd.use_archive)

		if err := cmd.downloadAssetTo(filepath.Base(cmd.use_archive), cmd.use_archive, latestRelease); err != nil {
			return fmt.Errorf("Fehler beim Herunterladen der neuesten Version: %w", err)
		}
	}

	archive, err := archive_reader.New(cmd.use_archive)
	if err != nil {
		return err
	}

	if !is_sub_process {
		cmd.infof("Update %s", kddp_bin_name)

		// get the kddp.exe from the archive
		kddp_exe, _, size, err := archive.GetElementFunc(func(path string) bool {
			return strings.HasSuffix(path, kddp_bin_name)
		})
		if err != nil {
			return err
		}
		defer kddp_exe.Close()

		if err := cmd.do_selfupdate(&progressReader{r: kddp_exe, max: size, msg: "Entpacke " + kddp_bin_name, should_print: true}); err != nil {
			return fmt.Errorf("Fehler beim Updaten von %s: %w", kddp_bin_name, err)
		}

		fmt.Println("\nUpdate war erfolgreich")

		cmd.infof("Lösche %s", cmd.use_archive)
		archive.Close()
		if err := os.Remove(cmd.use_archive); err != nil {
			return fmt.Errorf("Fehler beim Löschen der Archivdatei: %w", err)
		}

		return nil
	}
	defer archive.Close()

	cmd.infof("Update Bibliotheken")
	if err := cmd.update_lib(archive); err != nil {
		return fmt.Errorf("Fehler beim Updaten der Bibliotheken: %w", err)
	}

	cmd.infof("Update DDPLS")
	if err := cmd.update_ddpls(archive); err != nil {
		return fmt.Errorf("Fehler beim Updaten von DDPLS: %w", err)
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
	--pre_release: pre-release Versionen mit einbeziehen
	--jetzt: Falls eine neue Version verfügbar ist, wird diese ohne zu fragen sofort heruntergeladen und installiert`
}

func (cmd *UpdateCommand) do_selfupdate(kddp_exe io.Reader) error {
	old_kddp := filepath.Join(ddppath.Bin, kddp_bin_name+".old")
	if err := selfupdate.Apply(kddp_exe, selfupdate.Options{OldSavePath: old_kddp}); err != nil {
		if rerr := selfupdate.RollbackError(err); rerr != nil {
			fmt.Println("Rollback fehlgeschlagen:", rerr)
			fmt.Printf("Bitte manuell die neueste Version von %s herunterladen und ersetzen\n", kddp_bin_name)
			return fmt.Errorf("Fehler beim Rollback: %w", rerr)
		} else {
			fmt.Println("Update fehlgeschlagen:", err)
			return err
		}
	}
	cmd.infof("Starte neue Version von " + kddp_bin_name)

	// Get current process name and directory.
	execName, err := os.Executable()
	if err != nil {
		return fmt.Errorf("Fehler beim Abrufen des aktuellen Prozesses: %w", err)
	}
	execDir := filepath.Dir(execName)

	new_kddp := exec.Command(execName, "update",
		fmt.Sprintf("--use_archive=%s", cmd.use_archive),
		fmt.Sprintf("--wortreich=%v", cmd.verbose),
		fmt.Sprintf("--pre_release=%v", cmd.pre_release),
	)
	new_kddp.Dir = execDir
	new_kddp.Stdout = os.Stdout
	new_kddp.Stderr = os.Stderr
	new_kddp.Stdin = os.Stdin

	if err := new_kddp.Run(); err != nil {
		return fmt.Errorf("Fehler beim Neustarten von kddp: %w", err)
	}
	return nil
}

// updates the lib directory
func (cmd *UpdateCommand) update_lib(archive *archive_reader.ArchiveReader) error {
	empty_dir := func(path string) error {
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("Fehler beim Löschen von %s: %w", path, err)
		}
		return os.MkdirAll(path, os.ModePerm)
	}

	if err := empty_dir(ddppath.Lib); err != nil {
		return err
	}

	// write a regex that matches all files in the lib directory where lib could be preceeded by anything
	const (
		lib_regex_replace_pattern = `.*[\/\\]lib[\/\\]`
		lib_regex_pattern         = lib_regex_replace_pattern + `.*`
	)
	lib_regex := regexp.MustCompile(lib_regex_pattern)
	lib_replace_regex := regexp.MustCompile(lib_regex_replace_pattern)

	if err := archive.IterateElementsFunc(func(path string, isDir bool, r io.Reader, size uint64) error {
		if lib_regex.MatchString(path) {
			path = lib_replace_regex.ReplaceAllString(path, "")
			path = filepath.Join(ddppath.Lib, path)
			if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
				return fmt.Errorf("Fehler beim Erstellen des Ordners: %w", err)
			}
			if isDir {
				return nil
			}
			f, err := os.Create(path)
			if err != nil {
				return fmt.Errorf("Fehler beim Erstellen der Datei: %w", err)
			}
			defer f.Close()
			pr := &progressReader{r: r, max: size, msg: fmt.Sprintf("Entpacke %s", filepath.Base(path)), should_print: cmd.verbose}
			if _, err := io.Copy(f, pr); err != nil {
				return fmt.Errorf("Fehler beim Entpacken von %s: %w", path, err)
			}
			if cmd.verbose {
				fmt.Printf("\n")
			}
		}
		return nil
	}); err != nil {
		return err
	}

	gcc_version_cmd := gcc.New("-dumpfullversion")
	gcc_version_out, err := gcc_version_cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Fehler beim Abrufen der lokalen gcc Version: %w", err)
	}
	gcc_version := strings.Trim(string(gcc_version_out), "\r\n")
	if gcc_version == strings.Trim(GCCVERSIONFULL, "\r\n") {
		return nil // we do not have to recompile anything
	}

	cmd.infof("Bibliotheken werden neu kompiliert")
	make_cmd, ar_cmd := "make", "ar"
	if _, err := os.Stat(ddppath.Mingw64); err == nil || !os.IsNotExist(err) {
		make_cmd, ar_cmd = filepath.Join(ddppath.Mingw64, "bin", "mingw32-make.exe"), filepath.Join(ddppath.Mingw64, "bin", "ar.exe")
	}
	make_args, rmArg := make([]string, 0, 3), ""
	if runtime.GOOS == "windows" {
		make_args = append(make_args, fmt.Sprintf("CC=%s", gcc.Cmd()), fmt.Sprintf("AR=%s %s", ar_cmd, "rcs"))
		rmArg = "RM=" + filepath.Join(ddppath.Bin, "ddp-rm.exe")
	}

	runCmd := func(dir, name string, args ...string) error {
		cmd := exec.Command(name, args...)
		cmd.Dir = dir
		return cmd.Run()
	}

	runtime_src, stdlib_src := filepath.Join(ddppath.Lib, "runtime"), filepath.Join(ddppath.Lib, "stdlib")

	if err := runCmd(runtime_src, make_cmd, make_args...); err != nil {
		return fmt.Errorf("Fehler beim Kompilieren der Laufzeitbibliothek: %w", err)
	}
	if err := runCmd(stdlib_src, make_cmd, make_args...); err != nil {
		return fmt.Errorf("Fehler beim Kompilieren der Standardbibliothek: %w", err)
	}

	cmd.infof("Bibliotheken werden kopiert")
	if err := cp.Copy(filepath.Join(runtime_src, "libddpruntime.a"), filepath.Join(ddppath.Lib, "libddpruntime.a")); err != nil {
		return err
	}
	if err := cp.Copy(filepath.Join(runtime_src, "source", "main.o"), filepath.Join(ddppath.Lib, "main.o")); err != nil {
		return err
	}
	if err := cp.Copy(filepath.Join(stdlib_src, "libddpstdlib.a"), filepath.Join(ddppath.Lib, "libddpstdlib.a")); err != nil {
		return err
	}

	list_defs_cmd := NewDumpListDefsCommand()
	list_defs_cmd.Init([]string{"-o", filepath.Join(ddppath.Lib, "ddp_list_types_defs"), "--llvm_ir", "--object"})
	if err := list_defs_cmd.Run(); err != nil {
		return fmt.Errorf("Fehler beim Generieren der Liste der Typdefinitionen: %w", err)
	}

	cmd.infof("Bibliotheken werden aufgeräumt")
	if err := runCmd(runtime_src, make_cmd, "clean", rmArg); err != nil {
		return fmt.Errorf("Fehler beim Aufräumen der Laufzeitbibliothek: %w", err)
	}
	if err := runCmd(stdlib_src, make_cmd, "clean", rmArg); err != nil {
		return fmt.Errorf("Fehler beim Aufräumen der Standardbibliothek: %w", err)
	}

	return nil
}

func (cmd *UpdateCommand) update_ddpls(archive *archive_reader.ArchiveReader) error {
	ddpls_bin_name := exePath("DDPLS")
	// get the DDPLS.exe from the archive
	ddpls_exe, _, size, err := archive.GetElementFunc(func(path string) bool {
		return strings.HasSuffix(path, ddpls_bin_name)
	})
	if err != nil {
		return fmt.Errorf("Fehler beim Abrufen von %s: %w", ddpls_bin_name, err)
	}
	defer ddpls_exe.Close()

	ddpls_file, err := os.OpenFile(filepath.Join(ddppath.Bin, ddpls_bin_name), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return fmt.Errorf("Fehler beim Erstellen von %s: %w", ddpls_bin_name, err)
	}
	defer ddpls_file.Close()

	pr := &progressReader{r: ddpls_exe, max: size, msg: "Ersetze " + ddpls_bin_name, should_print: true}
	if _, err := io.Copy(ddpls_file, pr); err != nil {
		return fmt.Errorf("Fehler beim Ersetzen von %s: %w", ddpls_bin_name, err)
	}
	fmt.Printf("\n")
	return nil
}

// returns wether a is a newer version than b
func is_newer_version(a, b string) (bool, error) {
	const (
		version_fmt     = "v%d.%d.%d"
		version_pattern = `v[0-9]+\.[0-9]+\.[0-9]+(-pre|-alpha|-beta)?`
	)
	version_regex := regexp.MustCompile(version_pattern)
	if !version_regex.MatchString(a) {
		return false, fmt.Errorf("ungültiges Versions Format: %s", a)
	}
	if !version_regex.MatchString(b) {
		return false, fmt.Errorf("ungültiges Versions Format: %s", b)
	}

	parse_version := func(version string) (major, minor, patch int, additional string, err error) {
		var n int
		if n, err = fmt.Sscanf(version, version_fmt, &major, &minor, &patch); n < 3 {
			err = fmt.Errorf("Ungültiges Versions Format: %s", version)
		}
		if split := strings.Split(version, "-"); len(split) > 1 {
			additional = split[1]
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
		return nil, fmt.Errorf("Fehler beim Abrufen der Releases von Github: %w", err)
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

func (cmd *UpdateCommand) downloadAssetTo(assetName, targetPath string, release *github.RepositoryRelease) error {
	cmd.infof("searching for asset %s", assetName)
	for _, asset := range release.Assets {
		name := asset.GetName()
		cmd.infof("considering asset %s", name)
		if name == assetName {
			cmd.infof("found asset")
			r, _, err := cmd.gh.Repositories.DownloadReleaseAsset(context.Background(), "DDP-Projekt", "Kompilierer", asset.GetID(), http.DefaultClient)
			if err != nil {
				return fmt.Errorf("Fehler beim Herunterladen der Asset: %w", err)
			}
			defer r.Close()
			f, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("Fehler beim Erstellen der Datei: %w", err)
			}
			defer f.Close()
			pr := &progressReader{r: r, max: uint64(asset.GetSize()), msg: fmt.Sprintf("Lade %s herunter", assetName), should_print: true}
			if _, err := io.Copy(f, pr); err != nil {
				return fmt.Errorf("Fehler beim Herunterladen der Asset: %w", err)
			}
			return nil
		}
	}
	return errors.New("asset not found")
}

type progressReader struct {
	r            io.Reader
	total        uint64
	max          uint64
	msg          string
	should_print bool
}

func (pr *progressReader) Read(p []byte) (n int, err error) {
	n, err = pr.r.Read(p)
	pr.total += uint64(n)

	if !pr.should_print {
		return
	}
	// if total is smaller than 3 MB show progress in KB
	if pr.max < 3000000 {
		fmt.Printf("\r%s: %v of %v KB (%v%%)", pr.msg, pr.total/1000, pr.max/1000, pr.total*100/pr.max)
		return
	}
	fmt.Printf("\r%s: %v of %v MB (%v%%)", pr.msg, pr.total/1000000, pr.max/1000000, pr.total*100/pr.max)
	return
}

func exePath(path string) string {
	if runtime.GOOS == "windows" {
		return path + ".exe"
	}
	return path
}
