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
	use_archive     string // mainly meant for after kddp has updated itself
	old_kddp        string // only used when kddp updates itself
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
		old_kddp:        "",
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
	cmd.fs.StringVar(&cmd.use_archive, "use_archive", cmd.use_archive, "Nutze das angegebene Archiv anstatt die neueste Version herunterzuladen")
	cmd.fs.StringVar(&cmd.old_kddp, "old_kddp", cmd.old_kddp, "Nur für interne Zwecke")
	return parseFlagSet(cmd.fs, args)
}

func (cmd *UpdateCommand) Run() error {
	fmt.Printf("\nAktuelle Version: %s\n", DDPVERSION)
	latestRelease, err := cmd.getLatestRelease(cmd.pre_release)
	if err != nil {
		return err
	}
	latest_version := latestRelease.GetTagName()
	fmt.Printf("Neueste Version:  %s\n\n", latest_version)

	if is_newer, err := is_newer_version(DDPVERSION, latest_version); err != nil {
		return err
	} else if is_newer {
		fmt.Println("Es ist eine neuere Version verfügbar")
	} else {
		fmt.Println("DDP ist auf dem neusten Stand")
	}
	if cmd.compare_version { // early return if only the version should be compared
		return nil
	}

	if cmd.use_archive == "" {
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

		if err := cmd.downloadAssetTo(cmd.use_archive, cmd.use_archive, latestRelease); err != nil {
			return err
		}
	}

	archive, err := archive_reader.New(cmd.use_archive)
	if err != nil {
		return err
	}
	defer archive.Close()

	should_update_kddp := true
	if _, err := os.Stat(filepath.Join(ddppath.InstallDir, update_cache_name)); !errors.Is(err, os.ErrNotExist) {
		should_update_kddp = false
	}

	if should_update_kddp {
		cmd.infof("Update kddp.exe")

		// get the kddp.exe from the archive
		kddp_exe, size, err := archive.GetElementFunc(func(path string) bool {
			return strings.HasSuffix(path, "kddp.exe")
		})
		if err != nil {
			return err
		}
		defer kddp_exe.Close()

		if err := cmd.do_selfupdate(&progressReader{r: kddp_exe, max: size, msg: "Entpacke kddp.exe"}); err != nil {
			return err
		}

		if err := cmd.serialize_update_cache(); err != nil {
			return err
		}
		fmt.Println("\nUpdate von kddp.exe erfolgreich\nUm das update abzuschließen bitte 'kddp update' erneut ausführen")
		return nil
	}

	if err := cmd.parse_update_cache(); err != nil {
		return err
	}

	cmd.infof("Update Bibliotheken")
	if err := cmd.update_lib(archive); err != nil {
		return err
	}

	cmd.infof("Update DDPLS")
	if err := cmd.update_ddpls(archive); err != nil {
		return err
	}

	cmd.infof("Lösche %s", cmd.use_archive)
	if err := os.Remove(cmd.use_archive); err != nil {
		return err
	}
	cmd.infof("Lösche %s", cmd.old_kddp)
	if err := os.Remove(cmd.old_kddp); err != nil {
		return err
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

func (cmd *UpdateCommand) do_selfupdate(kddp_exe io.Reader) error {
	cmd.old_kddp = filepath.Join(ddppath.Bin, "kddp.exe.old")
	if err := selfupdate.Apply(kddp_exe, selfupdate.Options{OldSavePath: cmd.old_kddp}); err != nil {
		if rerr := selfupdate.RollbackError(err); rerr != nil {
			fmt.Println("Rollback fehlgeschlagen:", rerr)
			fmt.Println("Bitte manuell die neueste Version von kddp.exe herunterladen und ersetzen")
			return err
		} else {
			fmt.Println("Update fehlgeschlagen:", err)
			return err
		}
	}
	fmt.Printf("\n")
	return nil
}

const update_cache_name = "update_cache.txt"

func (cmd *UpdateCommand) serialize_update_cache() error {
	var args []string
	args = append(args, fmt.Sprintf("%s=%v", "wortreich", cmd.verbose))
	args = append(args, fmt.Sprintf("%s=%s", "use_archive", cmd.use_archive))
	args = append(args, fmt.Sprintf("%s=%s", "old_kddp", cmd.old_kddp))
	file, err := os.Create(filepath.Join(ddppath.InstallDir, update_cache_name))
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(strings.Join(args, ";"))
	return err
}

func (cmd *UpdateCommand) parse_update_cache() error {
	file, err := os.ReadFile(filepath.Join(ddppath.InstallDir, update_cache_name))
	if err != nil {
		return err
	}
	update_cache_args := strings.Split(string(file), ";")
	for _, arg := range update_cache_args {
		split := strings.Split(arg, "=")
		arg_name, arg_value := split[0], split[1]
		switch arg_name {
		case "wortreich":
			cmd.verbose = arg_value == "true"
		case "use_archive":
			cmd.use_archive = arg_value
		case "old_kddp":
			cmd.old_kddp = arg_value
		default:
			return fmt.Errorf("ungültiges Argument in %s: %s", update_cache_name, arg_name)
		}
	}
	return nil
}

// updates the lib directory
func (cmd *UpdateCommand) update_lib(archive *archive_reader.ArchiveReader) error {
	empty_dir := func(path string) error {
		if err := os.RemoveAll(path); err != nil {
			return err
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

	if err := archive.IterateElementsFunc(func(path string, r io.Reader, size uint64) error {
		if lib_regex.MatchString(path) {
			path = lib_replace_regex.ReplaceAllString(path, "")
			path = filepath.Join(ddppath.Lib, path)
			if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
				return err
			}
			f, err := os.Create(path)
			if err != nil {
				return err
			}
			defer f.Close()
			pr := &progressReader{r: r, max: size, msg: fmt.Sprintf("Entpacke %s", filepath.Base(path))}
			if _, err := io.Copy(f, pr); err != nil {
				return err
			}
			fmt.Printf("\n")
		}
		return nil
	}); err != nil {
		return err
	}

	gcc_version_cmd := gcc.New("-dumpfullversion")
	gcc_version_out, err := gcc_version_cmd.CombinedOutput()
	if err != nil {
		return err
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
		return err
	}
	if err := runCmd(stdlib_src, make_cmd, make_args...); err != nil {
		return err
	}

	cmd.infof("Bibliotheken werden kopiert")
	if err := cp.Copy(filepath.Join(runtime_src, "libddpruntime.a"), filepath.Join(ddppath.Lib, "libddpruntime.a")); err != nil {
		return err
	}
	if err := cp.Copy(filepath.Join(runtime_src, "main.o"), filepath.Join(ddppath.Lib, "main.o")); err != nil {
		return err
	}
	if err := cp.Copy(filepath.Join(stdlib_src, "libddpstdlib.a"), filepath.Join(ddppath.Lib, "libddpstdlib.a")); err != nil {
		return err
	}

	list_defs_cmd := NewDumpListDefsCommand()
	list_defs_cmd.Init([]string{"-o", filepath.Join(ddppath.Lib, "ddp_list_types_defs"), "--llvm_ir", "--object"})
	if err := list_defs_cmd.Run(); err != nil {
		return err
	}

	cmd.infof("Bibliotheken werden aufgeräumt")
	if err := runCmd(runtime_src, make_cmd, "clean", rmArg); err != nil {
		return err
	}
	if err := runCmd(stdlib_src, make_cmd, "clean", rmArg); err != nil {
		return err
	}

	return nil
}

func (cmd *UpdateCommand) update_ddpls(archive *archive_reader.ArchiveReader) error {
	// get the DDPLS.exe from the archive
	ddpls_exe, size, err := archive.GetElementFunc(func(path string) bool {
		return strings.HasSuffix(path, "DDPLS.exe")
	})
	if err != nil {
		return err
	}
	defer ddpls_exe.Close()

	ddpls_file, err := os.OpenFile(filepath.Join(ddppath.Bin, "DDPLS.exe"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer ddpls_file.Close()

	pr := &progressReader{r: ddpls_exe, max: size, msg: "Ersetze DDPLS.exe"}
	if _, err := io.Copy(ddpls_file, pr); err != nil {
		return err
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
			err = fmt.Errorf("ungültiges Versions Format: %s", version)
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

func (cmd *UpdateCommand) downloadAssetTo(assetName, targetPath string, release *github.RepositoryRelease) error {
	for _, asset := range release.Assets {
		if asset.GetName() == assetName {
			r, _, err := cmd.gh.Repositories.DownloadReleaseAsset(context.Background(), "DDP-Projekt", "Kompilierer", asset.GetID(), http.DefaultClient)
			if err != nil {
				return err
			}
			defer r.Close()
			f, err := os.Create(targetPath)
			if err != nil {
				return err
			}
			defer f.Close()
			pr := &progressReader{r: r, max: uint64(asset.GetSize()), msg: fmt.Sprintf("Lade %s herunter", assetName)}
			if _, err := io.Copy(f, pr); err != nil {
				return err
			}
			return nil
		}
	}
	return errors.New("asset not found")
}

type progressReader struct {
	r     io.Reader
	total uint64
	max   uint64
	msg   string
}

func (pr *progressReader) Read(p []byte) (n int, err error) {
	n, err = pr.r.Read(p)
	pr.total += uint64(n)
	fmt.Printf("\r%s: %v of %v MB (%v%%)", pr.msg, pr.total/1000000, pr.max/1000000, pr.total*100/pr.max)
	return
}
