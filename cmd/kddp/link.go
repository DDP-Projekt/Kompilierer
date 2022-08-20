package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/DDP-Projekt/Kompilierer/pkg/compiler"
	"github.com/DDP-Projekt/Kompilierer/pkg/scanner"
)

// invokes gcc on the input file and links it with the ddpstdlib
func invokeGCC(inputFile, outputFile string, dependencies *compiler.CompileResult, out io.Writer, nodeletes bool, gcc_flags string, extern_gcc_flags string) error {
	link_dependencies := make([]string, 0)

	extern_flags := strings.Split(extern_gcc_flags, " ")
	if extern_gcc_flags == "" {
		extern_flags = []string{}
	}

	for path := range dependencies.Dependencies {
		filename := filepath.Base(path)
		switch filename {
		case "ddpstdlib.lib", "ddpstdlib.a", "ddpruntime.lib", "ddpruntime.a":
			continue
		}

		switch filepath.Ext(path) {
		case ".lib", ".a", ".o":
			link_dependencies = append(link_dependencies, path)
		case ".c":
			if outPath, err := compileCFile(path, extern_flags, out); err != nil {
				return err
			} else {
				link_dependencies = append(link_dependencies, outPath)
				if !nodeletes {
					defer os.Remove(outPath)
				}
			}
		default:
			return fmt.Errorf("unexpected dependency '%s'", path)
		}
	}

	stdlibdir := filepath.Join(scanner.DDPPATH, "ddpstdlib.lib")
	runtimedir := filepath.Join(scanner.DDPPATH, "ddpruntime.lib")
	if runtime.GOOS == "linux" {
		stdlibdir = changeExtension(stdlibdir, ".a")
		runtimedir = changeExtension(runtimedir, ".a")
	}
	args := append(make([]string, 0), "-O2", "-o", outputFile) // -lm needed for math.h (don't know why I need this on linux)
	args = append(args, link_dependencies...)
	args = append(args, inputFile, stdlibdir, runtimedir, "-lm")

	if gcc_flags != "" {
		flags := strings.Split(gcc_flags, " ")
		args = append(args, flags...)
	}

	cmd := exec.Command("gcc", args...)
	if out != nil {
		cmd.Stdout = out
		cmd.Stderr = out
		out.Write([]byte(cmd.String() + "\n"))
	}
	return cmd.Run()
}

// helper for invokeGCC
// returns the path to the output file
func compileCFile(inputFile string, gcc_flags []string, out io.Writer) (string, error) {
	outPath := changeExtension(changeFilename(inputFile, "ddpextern_"+filepath.Base(inputFile)), ".o")
	args := append(make([]string, 0, 5), "-O2", "-c", "-o", outPath, inputFile)
	if len(gcc_flags) > 0 {
		args = append(args, gcc_flags...)
	}
	cmd := exec.Command("gcc", args...)
	if out != nil {
		cmd.Stdout = out
		cmd.Stderr = out
		out.Write([]byte(cmd.String() + "\n"))
	}
	return outPath, cmd.Run()
}

// helper to change a filename in a filepath
func changeFilename(path, newName string) string {
	return path[:len(path)-len(filepath.Base(path))] + newName
}
