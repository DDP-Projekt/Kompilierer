package main

import (
	"io"
	"os/exec"
)

func invokeGCC(inputFile, outputFile string, out io.Writer) error {
	cmd := exec.Command("gcc", "-o", outputFile, inputFile, "ddpstdlib.lib")
	if out != nil {
		cmd.Stdout = out
		cmd.Stderr = out
		out.Write([]byte(cmd.String() + "\n"))
	}
	return cmd.Run()
}
