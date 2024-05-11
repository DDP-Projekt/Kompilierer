package main

import (
	"context"
	"errors"
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
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update [--jetzt] [--vergleiche-version] [--pre-release]",
	Short: "Aktualisiert kddp",
	Long:  `Überprüft ob eine neue Version von kddp verfügbar ist und aktualisiert diesen gegebenenfalls`,
	RunE: func(cmd *cobra.Command, args []string) error {
		is_sub_process := updateUseArchive != ""

		if !is_sub_process {
			fmt.Printf("Aktuelle Version: %s\n", DDPVERSION)
			latestRelease, err := getLatestRelease(updatePreRelease)
			if err != nil {
				return fmt.Errorf("Fehler beim Abrufen der neuesten Version:\n\t%w", err)
			}
			latest_version := latestRelease.GetTagName()
			fmt.Printf("Neueste Version:  %s\n\n", latest_version)

			if is_newer, err := is_newer_version(latest_version, DDPVERSION); err != nil {
				return fmt.Errorf("Fehler beim Vergleichen der Versionen:\n\t%w", err)
			} else if is_newer {
				fmt.Println("Es ist eine neuere Version verfügbar")
			} else {
				fmt.Println("DDP ist auf dem neusten Stand")
				return nil
			}
			if updateCompareVersion { // early return if only the version should be compared
				return nil
			}

			if updateNow {
				fmt.Printf("DDP jetzt updaten? (j/n): ")
				var answer string
				if _, err := fmt.Scanln(&answer); err != nil {
					return fmt.Errorf("Fehler beim Lesen der Eingabe:\n\t%w", err)
				}
				answer = strings.TrimSpace(strings.ToLower(answer))
				if answer != "j" && answer != "y" {
					return nil
				}
			}

			archive_type := ".zip"
			if runtime.GOOS == "linux" {
				archive_type = ".tar.gz"
			}
			updateUseArchive = fmt.Sprintf("DDP-%s-%s-%s", latest_version, runtime.GOOS, runtime.GOARCH)
			if runtime.GOOS == "windows" {
				updateUseArchive += "-no-mingw"
			}
			updateUseArchive += archive_type
			updateUseArchive = filepath.Join(ddppath.InstallDir, updateUseArchive)

			if err := downloadAssetTo(filepath.Base(updateUseArchive), updateUseArchive, latestRelease); err != nil {
				return fmt.Errorf("Fehler beim Herunterladen der neuesten Version:\n\t%w", err)
			}
		}

		archive, err := archive_reader.New(updateUseArchive)
		if err != nil {
			return err
		}

		if !is_sub_process {
			infof("\nUpdate %s", kddp_bin_name)

			// get the kddp.exe from the archive
			kddp_exe, _, size, err := archive.GetElementFunc(func(path string) bool {
				return strings.HasSuffix(path, kddp_bin_name)
			})
			if err != nil {
				return fmt.Errorf("Fehler beim entpacken von %s:\n\t%w", kddp_bin_name, err)
			}
			defer kddp_exe.Close()

			if err := do_selfupdate(&progressReader{r: kddp_exe, max: size, msg: "Entpacke " + kddp_bin_name, should_print: true}); err != nil {
				return fmt.Errorf("Fehler beim Updaten von %s:\n\t%w", kddp_bin_name, err)
			}

			fmt.Println("\nUpdate war erfolgreich")

			infof("Lösche %s", updateUseArchive)
			archive.Close()
			if err := os.Remove(updateUseArchive); err != nil {
				return fmt.Errorf("Fehler beim Löschen der Archivdatei:\n\t%w", err)
			}

			return nil
		}
		defer archive.Close()

		infof("Update Bibliotheken")
		if bib_err := update_lib(archive); bib_err != nil {
			err = fmt.Errorf("Fehler beim Updaten der Bibliotheken:\n\t%w", bib_err)
		}

		infof("Update DDPLS")
		if ls_err := update_ddpls(archive); ls_err != nil {
			err = errors.Join(err, fmt.Errorf("Fehler beim Updaten von DDPLS:\n\t%w", ls_err))
		}
		return err
	},
}

var (
	updateCompareVersion bool   // flag for update
	updatePreRelease     bool   // flag for update
	updateUseArchive     string // flag for update
	updateNow            bool   // flag for update

	gh            *github.Client // github client for the update command
	kddp_bin_name = exePath("kddp")
)

func init() {
	updateCmd.Flags().BoolVar(&updateCompareVersion, "vergleiche-version", false, "Vergleicht neue Version mit der installierten")
	updateCmd.Flags().BoolVar(&updatePreRelease, "pre-release", false, "Aktualisiert auf eine Vorabversion")
	updateCmd.Flags().StringVar(&updateUseArchive, "use-archive", "", "Nur für interne Zwecke")
	updateCmd.Flags().Lookup("use-archive").Hidden = true
	updateCmd.Flags().BoolVar(&updateNow, "jetzt", false, "Aktualisiert sofort ohne zu fragen")

	gh = github.NewClient(nil)
}

func infof(format string, a ...any) {
	if verbose {
		fmt.Printf(format+"\n", a...)
	}
}

func do_selfupdate(kddp_exe io.Reader) error {
	old_kddp := filepath.Join(ddppath.Bin, kddp_bin_name+".old")
	if err := selfupdate.Apply(kddp_exe, selfupdate.Options{OldSavePath: old_kddp}); err != nil {
		if rerr := selfupdate.RollbackError(err); rerr != nil {
			fmt.Println("Rollback fehlgeschlagen:", rerr)
			fmt.Printf("Bitte manuell die neueste Version von %s herunterladen und ersetzen\n", kddp_bin_name)
			return fmt.Errorf("Fehler beim Rollback:\n\t%w", rerr)
		} else {
			fmt.Println("Update fehlgeschlagen:", err)
			return err
		}
	}

	infof("\nStarte neue Version von %s\n", kddp_bin_name)

	execName := filepath.Join(ddppath.Bin, kddp_bin_name)
	new_kddp := exec.Command(execName, "update",
		fmt.Sprintf("--use_archive=%s", updateUseArchive),
		fmt.Sprintf("--wortreich=%v", verbose),
		fmt.Sprintf("--pre_release=%v", updatePreRelease),
	)
	new_kddp.Dir = ddppath.InstallDir
	new_kddp.Stdout = os.Stdout
	new_kddp.Stderr = os.Stderr
	new_kddp.Stdin = os.Stdin

	infof("Command: %s\n", new_kddp.String())
	if err := new_kddp.Run(); err != nil {
		return fmt.Errorf("Fehler beim Neustarten von kddp:\n\t%w", err)
	}
	return nil
}

// updates the lib directory and the Duden directory
func update_lib(archive *archive_reader.ArchiveReader) (err error) {
	empty_dir := func(path string) error {
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("Fehler beim Löschen von %s:\n\t%w", path, err)
		}
		return os.MkdirAll(path, os.ModePerm)
	}

	err = empty_dir(ddppath.Lib)
	if delete_err := empty_dir(ddppath.Duden); delete_err != nil {
		err = errors.Join(err, delete_err)
	}

	// write a regex that matches all files in the lib directory where lib could be preceeded by anything
	const (
		lib_regex_replace_pattern   = `.*[\/\\]lib[\/\\]`
		lib_regex_pattern           = lib_regex_replace_pattern + `.*`
		duden_regex_replace_pattern = `.*[\/\\]Duden[\/\\]`
		duden_regex_pattern         = duden_regex_replace_pattern + `.*`
	)
	lib_regex := regexp.MustCompile(lib_regex_pattern)
	lib_replace_regex := regexp.MustCompile(lib_regex_replace_pattern)
	duden_regex := regexp.MustCompile(duden_regex_pattern)
	duden_replace_regex := regexp.MustCompile(duden_regex_replace_pattern)

	if it_err := archive.IterateElementsFunc(func(path string, isDir bool, r io.Reader, size uint64) error {
		create_file := func(path string) error {
			if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
				return fmt.Errorf("Fehler beim Erstellen des Ordners:\n\t%w", err)
			}
			if isDir {
				return nil
			}
			f, err := os.Create(path)
			if err != nil {
				return fmt.Errorf("Fehler beim Erstellen der Datei:\n\t%w", err)
			}
			defer f.Close()
			pr := &progressReader{r: r, max: size, msg: fmt.Sprintf("Entpacke %s", filepath.Base(path)), should_print: verbose}
			if _, err := io.Copy(f, pr); err != nil {
				return fmt.Errorf("Fehler beim Entpacken von %s:\n\t%w", path, err)
			}
			if verbose {
				fmt.Printf("\n")
			}
			return nil
		}

		if lib_regex.MatchString(path) {
			path = lib_replace_regex.ReplaceAllString(path, "")
			path = filepath.Join(ddppath.Lib, path)
			return create_file(path)
		} else if duden_regex.MatchString(path) {
			path = duden_replace_regex.ReplaceAllString(path, "")
			path = filepath.Join(ddppath.Duden, path)
			return create_file(path)
		}
		return nil
	}); it_err != nil {
		return errors.Join(err, it_err)
	}

	gcc_version_cmd := gcc.New("-dumpfullversion")
	gcc_version_out, gcc_err := gcc_version_cmd.CombinedOutput()
	if gcc_err != nil {
		err = errors.Join(err, fmt.Errorf("Fehler beim Abrufen der lokalen gcc Version:\n\t%w", gcc_err))
	} else {
		gcc_version := strings.Trim(string(gcc_version_out), "\r\n")
		if gcc_version == strings.Trim(GCCVERSIONFULL, "\r\n") {
			return err // we do not have to recompile anything
		}
	}
	// if there was an error we just recompile to be sure

	infof("Bibliotheken werden neu kompiliert")
	make_cmd, ar_cmd := "make", "ar"
	if _, err := os.Stat(ddppath.Mingw64); err == nil || !os.IsNotExist(err) {
		make_cmd, ar_cmd = filepath.Join(ddppath.Mingw64, "bin", "mingw32-make.exe"), filepath.Join(ddppath.Mingw64, "bin", "ar.exe")
	}
	make_args := make([]string, 0, 3)
	if runtime.GOOS == "windows" {
		make_args = append(make_args, fmt.Sprintf("CC=%s", gcc.Cmd()), fmt.Sprintf("AR=%s %s", ar_cmd, "rcs"))
	}

	runCmd := func(dir, name string, args ...string) error {
		cmd := exec.Command(name, args...)
		cmd.Dir = dir
		return cmd.Run()
	}

	runtime_src, stdlib_src := filepath.Join(ddppath.Lib, "runtime"), filepath.Join(ddppath.Lib, "stdlib")

	if run_err := runCmd(runtime_src, make_cmd, make_args...); run_err != nil {
		return errors.Join(err, fmt.Errorf("Fehler beim Kompilieren der Laufzeitbibliothek:\n\t%w", run_err))
	}
	if run_err := runCmd(stdlib_src, make_cmd, make_args...); run_err != nil {
		return errors.Join(fmt.Errorf("Fehler beim Kompilieren der Standardbibliothek:\n\t%w", run_err))
	}

	infof("Bibliotheken werden kopiert")
	if cp_err := cp.Copy(filepath.Join(runtime_src, "libddpruntime.a"), filepath.Join(ddppath.Lib, "libddpruntime.a")); cp_err != nil {
		return errors.Join(err, fmt.Errorf("Fehler beim Kopieren der Laufzeitbibliothek:\n\t%w", cp_err))
	}
	if cp_err := cp.Copy(filepath.Join(runtime_src, "source", "main.o"), filepath.Join(ddppath.Lib, "main.o")); cp_err != nil {
		return errors.Join(err, fmt.Errorf("Fehler beim Kopieren der main.o:\n\t%w", cp_err))
	}
	if cp_err := cp.Copy(filepath.Join(stdlib_src, "libddpstdlib.a"), filepath.Join(ddppath.Lib, "libddpstdlib.a")); cp_err != nil {
		return errors.Join(err, fmt.Errorf("Fehler beim Kopieren der Standardbibliothek:\n\t%w", cp_err))
	}

	infof("Generiere ddp_list_types_defs.*")
	if run_err := dumpListDefsCommand.RunE(dumpListDefsCommand, []string{"-o", filepath.Join(ddppath.Lib, "ddp_list_types_defs"), "--llvm-ir", "--object"}); run_err != nil {
		return errors.Join(err, fmt.Errorf("Fehler beim Generieren der Liste der Typdefinitionen:\n\t%w", run_err))
	}

	return err
}

func update_ddpls(archive *archive_reader.ArchiveReader) error {
	ddpls_bin_name := exePath("DDPLS")
	// get the DDPLS.exe from the archive
	ddpls_exe, _, size, err := archive.GetElementFunc(func(path string) bool {
		return strings.HasSuffix(path, ddpls_bin_name)
	})
	if err != nil {
		return fmt.Errorf("Fehler beim Entpacken von %s:\n\t%w", ddpls_bin_name, err)
	}
	defer ddpls_exe.Close()

	ddpls_file, err := os.OpenFile(filepath.Join(ddppath.Bin, ddpls_bin_name), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return fmt.Errorf("Fehler beim Erstellen von %s:\n\t%w", ddpls_bin_name, err)
	}
	defer ddpls_file.Close()

	pr := &progressReader{r: ddpls_exe, max: size, msg: "Ersetze " + ddpls_bin_name, should_print: true}
	if _, err := io.Copy(ddpls_file, pr); err != nil {
		return fmt.Errorf("Fehler beim Ersetzen von %s:\n\t%w", ddpls_bin_name, err)
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
func getLatestRelease(pre_release bool) (*github.RepositoryRelease, error) {
	if !pre_release {
		release, _, err := gh.Repositories.GetLatestRelease(context.Background(), "DDP-Projekt", "Kompilierer")
		return release, err
	}
	releases, _, err := gh.Repositories.ListReleases(context.Background(), "DDP-Projekt", "Kompilierer", &github.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("Fehler beim Abrufen der Releases von Github:\n\t%w", err)
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

func downloadAssetTo(assetName, targetPath string, release *github.RepositoryRelease) error {
	infof("searching for asset %s", assetName)
	for _, asset := range release.Assets {
		name := asset.GetName()
		infof("considering asset %s", name)
		if name == assetName {
			infof("found asset")
			r, _, err := gh.Repositories.DownloadReleaseAsset(context.Background(), "DDP-Projekt", "Kompilierer", asset.GetID(), http.DefaultClient)
			if err != nil {
				return fmt.Errorf("Fehler beim Herunterladen der Asset:\n\t%w", err)
			}
			defer r.Close()
			f, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("Fehler beim Erstellen der Datei:\n\t%w", err)
			}
			defer f.Close()
			pr := &progressReader{r: r, max: uint64(asset.GetSize()), msg: fmt.Sprintf("Lade %s herunter", assetName), should_print: true}
			if _, err := io.Copy(f, pr); err != nil {
				return fmt.Errorf("Fehler beim Herunterladen der Asset:\n\t%w", err)
			}
			return nil
		}
	}
	return fmt.Errorf("asset %s not found", assetName)
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
