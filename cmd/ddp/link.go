package main

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

func invokeLLD(inputFile, outputFile string, out io.Writer) error {
	var args = make([]string, 0, 4)
	lldPath := DDPPATH + "/bin/lld-link.exe"
	libPath := DDPPATH + "/lib"
	args = append(args, "-out:"+outputFile, "-subsystem:console", "-libpath:"+libPath, inputFile)
	// append stdlib .o files
	err := filepath.WalkDir(libPath, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && (filepath.Ext(path) == ".o" || filepath.Ext(path) == ".lib") {
			args = append(args, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	cmd := exec.Command(lldPath, args...)
	if out != nil {
		cmd.Stdout = out
		cmd.Stderr = out
		out.Write([]byte(cmd.String() + "\n"))
	}
	return cmd.Run()
}
