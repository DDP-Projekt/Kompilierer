package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/DDP-Projekt/Kompilierer/cmd/internal/compression"
	"github.com/badgerodon/penv"
	"github.com/kardianos/osext"
	cp "github.com/otiai10/copy"
)

var (
	gccCmd    = "gcc"
	makeCmd   = "make"
	arCmd     = "ar"
	vscodeCmd = "code"
	kddpCmd   = "bin/kddp"
	cwd       = "./"
)

func exit(code int) {
	InfoF("Press ENTER to exit...")
	if !always_yes {
		fmt.Scanln()
	}
	os.Exit(code)
}

func main() {
	flag.BoolVar(&always_yes, "force", false, "always answer yes to prompts")
	flag.Parse()
	if !prompt("Welcome to the DDP Installer!\nThis setup will simply unpack some files and ask you for permission to change some environment variables and such.\nDo you want to continue") {
		return
	}

	cwd_, err := os.Getwd()
	if err != nil {
		WarnF("error getting current working directory: %s", err)
	} else {
		cwd = cwd_
	}
	installLocales()

	_, hasGcc := LookupCommand(gccCmd)

	if !hasGcc && runtime.GOOS == "windows" {
		InfoF("gcc not found, installing mingw64")
		InfoF("unzipping mingw64.zip")
		err := compression.DecompressFolder("mingw64.zip", "mingw64")
		if err != nil {
			ErrorF("Error while unzipping mingw64: %s", err)
			ErrorF("gcc not available, aborting")
			exit(1)
		}
		DoneF("unzipped mingw64")

		gccCmd, err = filepath.Abs(filepath.Join("mingw64", "bin", "gcc"))
		if err != nil {
			WarnF("error getting absolute Path: %s", err)
		}
		gccCmd = filepath.ToSlash(gccCmd)
		arCmd, err = filepath.Abs(filepath.Join("mingw64", "bin", "ar"))
		if err != nil {
			WarnF("error getting absolute Path: %s", err)
		}
		arCmd = filepath.ToSlash(arCmd)
		makeCmd, err = filepath.Abs(filepath.Join("mingw64", "bin", "mingw32-make"))
		if err != nil {
			WarnF("error getting absolute Path: %s", err)
		}
		makeCmd = filepath.ToSlash(makeCmd)

		DoneF("Installed mingw64")
		DoneF("using newly installed mingw64 for gcc, ar and make")
	} else if !hasGcc && runtime.GOOS != "windows" {
		ErrorF("gcc not found, aborting")
		exit(1)
	}

	if makeCmd == "make" { // if we don't use the zipped mingw32-make
		_, hasMake := LookupCommand(makeCmd)

		if !hasMake && runtime.GOOS == "windows" {
			InfoF("make not found, looking for mingw32-make")
			makeCmd, hasMake = LookupCommand("mingw32-make")
			if !hasMake {
				ErrorF("mingw32-make not found, aborting")
				exit(1)
			}
			makeCmd = filepath.ToSlash(makeCmd)
		} else if !hasMake && runtime.GOOS != "windows" {
			WarnF("make not found")
		}
	}

	if isSameGccVersion() {
		DoneF("gcc versions match")
	} else {
		InfoF("re-building runtime and stdlib")
		recompileLibs()
	}

	if vscodeCmd, hasVscode := LookupCommand(vscodeCmd); hasVscode && prompt("Do you want to install vscode-ddp (the DDP vscode extension)") {
		InfoF("installing vscode-ddp as vscode extension")
		if _, err := runCmd("", vscodeCmd, "--install-extension", "DDP-Projekt.vscode-ddp", "--force"); err == nil {
			DoneF("Installed vscode-ddp")
		}
	}

	if prompt("Do you want to set the DDPPATH environment variable") {
		if exedir, err := osext.ExecutableFolder(); err != nil {
			WarnF("Could not retreive executable path")
		} else {
			InfoF("Setting the environment variable DDPPATH to %s", exedir)
			if err := penv.SetEnv("DDPPATH", exedir); err != nil {
				ErrorF("Error setting DDPPATH: %s\nConsider adding it yourself", err)
			}
		}
	}

	if prompt("Do you want to add the DDP/bin directory to your PATH") {
		if exedir, err := osext.ExecutableFolder(); err != nil {
			WarnF("Could not retreive executable path")
		} else {
			binPath := filepath.Join(exedir, "bin")
			InfoF("Appending %s to the PATH", binPath)
			if err := penv.AppendEnv("PATH", binPath); err != nil {
				ErrorF("Error appending to PATH: %s\nConsider adding DDP/bin to PATH yourself", err)
			}
		}
	}

	if !errored {
		DoneF("DDP is now installed")
		if prompt("Do you want to delete files that are not needed anymore") {
			if runtime.GOOS == "windows" {
				InfoF("deleting mingw64.zip")
				if err := os.Remove("mingw64.zip"); err != nil {
					WarnF("error removing mingw64.zip: %s", err)
				} else {
					DoneF("removed mingw64.zip")
				}
			}
		}
		DoneF("The ddp-setup finished successfuly, you can now delete it")
	}
	exit(0)
}

func installLocales() {
	InfoF("installing german locales")
	if runtime.GOOS == "linux" {
		if _, err := runCmd("", "locale-gen", "de_DE.UTF-8"); err != nil {
			WarnF("error installing german locale: %s", err)
		}
	} else if runtime.GOOS == "windows" {
		WarnF("you are using windows, make sure you have the correct language packs installed")
	}
}

func isSameGccVersion() bool {
	gccVersion, err := runCmd("", gccCmd, "-dumpfullversion")
	if err != nil {
		return false
	}
	gccVersion = strings.Trim(gccVersion, "\r\n") // TODO: this
	kddpVersionOutput, err := runCmd("", filepath.Join("bin", "kddp"), "version", "--wortreich")
	if err != nil {
		return false
	}
	gccVersionLine := strings.Split(kddpVersionOutput, "\n")[2]
	kddpGccVersion := strings.Trim(strings.Split(gccVersionLine, " ")[2], "\r\n")
	match := gccVersion == kddpGccVersion
	if !match {
		InfoF("local gcc version, and kddp gcc version mismatch (%s vs %s)", gccVersion, kddpGccVersion)
	}
	return match
}

func recompileLibs() {
	make_args := make([]string, 0)
	rmArg := ""
	if runtime.GOOS == "windows" {
		make_args = append(make_args, fmt.Sprintf("CC=%s", gccCmd), fmt.Sprintf("AR=%s %s", arCmd, "rcs"))
		rmArg = fmt.Sprintf("%s %s", filepath.Join(cwd, "bin", "kddp.exe"), "rm")
	}

	if _, err := runCmd("lib/runtime/", makeCmd, make_args...); err != nil {
		return
	}
	DoneF("re-compiled the runtime")
	if _, err := runCmd("lib/stdlib/", makeCmd, make_args...); err != nil {
		return
	}
	DoneF("re-compiled the stdlib")

	InfoF("removing pre-compiled runtime")
	if err := os.Remove("lib/libddpruntime.a"); err != nil {
		WarnF("error removing pre-compiled runtime: %s", err)
	}
	InfoF("removing pre-compiled lib/main.o lib/ddp_list_types_defs.o lib/ddp_list_types_defs.ll")
	if err := os.Remove("lib/main.o"); err != nil {
		WarnF("error removing pre-compiled lib/main.o: %s", err)
	}
	if err := os.Remove("lib/ddp_list_types_defs.o"); err != nil {
		WarnF("error removing pre-compiled lib/ddp_list_types_defs.o: %s", err)
	}
	if err := os.Remove("lib/ddp_list_types_defs.ll"); err != nil {
		WarnF("error removing pre-compiled lib/ddp_list_types_defs.ll: %s", err)
	}
	InfoF("removing pre-compiled stdlib")
	if err := os.Remove("lib/libddpstdlib.a"); err != nil {
		WarnF("error removing pre-compiled stdlib: %s", err)
	}

	InfoF("copying re-compiled runtime")
	if err := cp.Copy("lib/runtime/libddpruntime.a", "lib/libddpruntime.a"); err != nil {
		ErrorF("error copying re-compiled runtime: %s", err)
	}
	InfoF("copying re-compiled lib/main.o")
	if err := cp.Copy("lib/runtime/source/main.o", "lib/main.o"); err != nil {
		ErrorF("error copying re-compiled runtime: %s", err)
	}
	InfoF("regenerating lib/ddp_list_types_defs.ll and lib/ddp_list_types_defs.o")
	if _, err := runCmd("", kddpCmd, "dump-list-defs", "-o", "lib/ddp_list_types_defs", "--llvm_ir", "--object"); err != nil {
		ErrorF("error regenerating lib/ddp_list_types_defs.ll and lib/ddp_list_types_defs.o: %s", err)
	}
	InfoF("copying re-compiled stdlib")
	if err := cp.Copy("lib/stdlib/libddpstdlib.a", "lib/libddpstdlib.a"); err != nil {
		ErrorF("error copying re-compiled stdlib: %s", err)
	}

	InfoF("cleaning runtime directory")
	clean_args := make([]string, 0, 2)
	clean_args = append(clean_args, "clean")
	if rmArg != "" {
		clean_args = append(clean_args, rmArg)
	}
	if _, err := runCmd("lib/runtime/", makeCmd, clean_args...); err != nil {
		WarnF("error while cleaning runtime directory: %s", err)
	}
	InfoF("cleaning stdlib directory")
	if _, err := runCmd("lib/stdlib/", makeCmd, clean_args...); err != nil {
		WarnF("error while cleaning stdlib directory: %s", err)
	}

	DoneF("recompiled libraries")
}

func runCmd(dir string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmdStr := cmd.String()
	InfoF(cmdStr)
	out, err := cmd.CombinedOutput()
	if err != nil {
		ErrorF("'%s' failed (%s) output: %s", cmdStr, err, out)
	}
	return string(out), err
}

func LookupCommand(cmd string) (string, bool) {
	InfoF("Looking for %s", cmd)
	path, err := exec.LookPath(cmd)
	if err == nil {
		DoneF("Found %s in %s", cmd, path)
	} else {
		WarnF("Unable to find %s", cmd)
	}
	return path, err == nil
}
