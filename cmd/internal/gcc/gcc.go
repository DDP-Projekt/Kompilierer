package gcc

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/DDP-Projekt/Kompilierer/src/ddppath"
)

var gcc = "gcc"

func init() {
	if runtime.GOOS == "windows" {
		_, err := os.Stat(ddppath.Mingw64)
		if err == nil || !os.IsNotExist(err) {
			gcc = filepath.Join(ddppath.Mingw64, "bin", "gcc.exe")
		}
	}
}

// returns the gcc command used by New
// either "gcc" or $(DDPPATH)/mingw64/bin/gcc.exe
func Cmd() string {
	return gcc
}

// creates a new command calling gcc with the specified args
// on windows it first searches the DDPPATH for a
// mingw64 installation
func New(args ...string) *exec.Cmd {
	cmd := exec.Command(gcc, args...)
	return cmd
}
