package gcc

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/DDP-Projekt/Kompilierer/pkg/scanner"
)

var gcc = "gcc"

func init() {
	if runtime.GOOS == "windows" {
		DDPPATH := scanner.DDPPATH
		_, err := os.Stat(filepath.Join(DDPPATH, "mingw64"))
		if err == nil || !os.IsNotExist(err) {
			gcc = filepath.Join(DDPPATH, "mingw64", "bin", "gcc.exe")
		}
	}
}

// creates a new command calling gcc with the specified args
// on windows it first searches the DDPPATH for a
// mingw64 installation
func New(args ...string) *exec.Cmd {
	cmd := exec.Command(gcc, args...)
	return cmd
}
