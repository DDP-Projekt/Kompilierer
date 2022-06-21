package main

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// invokes gcc on the input file and links it with the ddpstdlib
func invokeGCC(inputFile, outputFile string, out io.Writer) error {
	libdir := filepath.Join(filepath.Dir(os.Args[0]), "ddpstdlib.lib")
	if runtime.GOOS == "linux" {
		libdir = changeExtension(libdir, ".a")
	}
	cmd := exec.Command("gcc", "-o", outputFile, inputFile, libdir, "-lm") // -lm needed for math.h (don't know why I need this on linux)
	if out != nil {
		cmd.Stdout = out
		cmd.Stderr = out
		out.Write([]byte(cmd.String() + "\n"))
	}
	return cmd.Run()
}
