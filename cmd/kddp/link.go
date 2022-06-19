package main

import (
	"io"
	"os/exec"
	"runtime"
)

// invokes gcc on the input file and links it with the ddpstdlib
func invokeGCC(inputFile, outputFile string, out io.Writer) error {
	libdir := "ddpstdlib.lib"
	if runtime.GOOS == "linux" {
		libdir = "ddpstdlib.a"
	}
	cmd := exec.Command("gcc", "-o", outputFile, inputFile, libdir, "-lm") // -lm needed for math.h (don't know why I need this on linux)
	if out != nil {
		cmd.Stdout = out
		cmd.Stderr = out
		out.Write([]byte(cmd.String() + "\n"))
	}
	return cmd.Run()
}
