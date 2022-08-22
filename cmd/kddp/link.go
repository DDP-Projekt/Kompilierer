package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/DDP-Projekt/Kompilierer/pkg/compiler"
	"github.com/DDP-Projekt/Kompilierer/pkg/scanner"
)

// invokes gcc on the input file and links it with the ddpstdlib
func invokeGCC(inputFile, outputFile string, dependencies *compiler.Result, out io.Writer, nodeletes bool, gcc_flags string, extern_gcc_flags string) error {
	// split the flags passed to gcc when compiling extern .c files
	extern_flags := strings.Split(extern_gcc_flags, " ")
	if extern_gcc_flags == "" {
		extern_flags = []string{}
	}

	var (
		link_objects = map[string][]string{} // library-search-paths to librarie filename map
		input_files  = []string{inputFile}   // all input files (.o)
	)

	// add external files to be linked
	for path := range dependencies.Dependencies {
		filename := filepath.Base(path)
		// stdlib and runtime are linked by default
		// ignore them because of the Duden
		switch filename {
		case "libddpstdlib.a", "libddpruntime.a":
			continue
		}

		switch filepath.Ext(path) {
		case ".lib", ".a": // libraries are linked using the -l: flag
			if objs, ok := link_objects[filepath.Dir(path)]; ok {
				link_objects[filepath.Dir(path)] = append(objs, filename)
			} else {
				link_objects[filepath.Dir(path)] = []string{filename}
			}
		case ".o": // object files are simple input files
			input_files = append(input_files, path)
		case ".c": // .c files must be compiled first
			if outPath, err := compileCFile(path, extern_flags, out); err != nil {
				return err
			} else {
				input_files = append(input_files, outPath)
				if !nodeletes {
					defer os.Remove(outPath)
				}
			}
		default:
			return fmt.Errorf("unexpected dependency '%s'", path)
		}
	}

	libdir := scanner.DDPPATH

	args := append(make([]string, 0), "-o", outputFile, "-O2", "-L"+libdir)

	// add all librarie-search-paths
	for k := range link_objects {
		args = append(args, "-L"+k)
	}
	// add the input files
	args = append(args, input_files...)
	// add default dependencies
	args = append(args, "-lddpstdlib", "-lddpruntime", "-lm")
	// add external dependencies
	for _, libs := range link_objects {
		for _, lib := range libs {
			args = append(args, "-l:"+lib)
		}
	}

	// add additional gcc-flags such as other needed libraries
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
	args := append(make([]string, 0, 5), "-O2", "-c", "-Wall", "-o", outPath, inputFile)
	if len(gcc_flags) > 0 {
		args = append(args, gcc_flags...)
	}
	if out != nil {
		fmt.Fprintf(out, "invoking gcc on %s\n", inputFile)
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
