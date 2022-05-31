package main

import (
	"io"
	"os/exec"
	"runtime"
)

func invokeGCC(inputFile, outputFile string, out io.Writer) error {
	libdir := "ddpstdlib.lib"
	if runtime.GOOS == "linux" {
		libdir = "ddpstdlib.a"
	}
	cmd := exec.Command("gcc", "-o", outputFile, inputFile, libdir)
	if out != nil {
		cmd.Stdout = out
		cmd.Stderr = out
		out.Write([]byte(cmd.String() + "\n"))
	}
	return cmd.Run()
}
