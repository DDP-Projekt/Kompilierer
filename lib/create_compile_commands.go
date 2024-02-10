// Description: This script is used to generate compile_commands.json file for the runtime and stdlib
// example usage:
//
//	$ make all --always-make --dry-run | grep -w "gcc -c" | go run create_compile_commands.go > compile_commands.json
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	commands := make([]CompileCommand, 0)
	reader := bufio.NewReader(os.Stdin)
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF && len(line) == 0 {
			break
		} else if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		commands = append(commands, processLine(strings.TrimSpace(line)))
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "\t")
	if err := encoder.Encode(commands); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var cwd, _ = os.Getwd()

type CompileCommand struct {
	Arguments []string `json:"arguments"`
	Directory string   `json:"directory"`
	File      string   `json:"file"`
}

func processLine(line string) CompileCommand {
	args := strings.Split(line, " ")
	command, err := exec.LookPath(args[0])
	if err != nil {
		command = args[0]
	}
	args[0] = command
	dir, err := filepath.Abs(cwd)
	if err != nil {
		dir, err = filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			dir = cwd
		}
	}
	file, err := filepath.Abs(args[len(args)-1])
	if err != nil {
		file = args[len(args)-1]
	}
	if runtime.GOOS == "windows" {
		args = append(args, "--target=x86_64-pc-windows-gnu") // for mingw headers
	}
	return CompileCommand{
		Arguments: args,
		Directory: dir,
		File:      file,
	}
}
