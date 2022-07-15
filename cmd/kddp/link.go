package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/DDP-Projekt/Kompilierer/pkg/compiler"
)

// invokes gcc on the input file and links it with the ddpstdlib
func invokeGCC(inputFile, outputFile string, dependencies *compiler.CompileResult, out io.Writer, nodeletes bool) error {
	link_dependencies := make([]string, 0)

	for path := range dependencies.Dependencies {
		filename := filepath.Base(path)
		if filename == "ddpstdlib.lib" || filename == "ddpstdlib.a" {
			continue
		}

		switch filepath.Ext(path) {
		case ".lib", ".a", ".o":
			link_dependencies = append(link_dependencies, path)
		case ".c":
			if outPath, err := compileCFile(path, out); err != nil {
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

	stdlibdir := filepath.Join(filepath.Dir(os.Args[0]), "ddpstdlib.lib")
	if runtime.GOOS == "linux" {
		stdlibdir = changeExtension(stdlibdir, ".a")
	}
	link_dependencies = append(link_dependencies, stdlibdir)
	args := append(make([]string, 0), "-o", outputFile, inputFile) // -lm needed for math.h (don't know why I need this on linux)
	args = append(args, link_dependencies...)
	args = append(args, "-lm")
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
func compileCFile(inputFile string, out io.Writer) (string, error) {
	outPath := changeExtension(changeFilename(inputFile, "ddpextern_"+filepath.Base(inputFile)), ".o")
	cmd := exec.Command("gcc", "-c", "-o", outPath, inputFile)
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
