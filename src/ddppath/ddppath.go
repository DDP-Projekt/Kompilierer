package ddppath

import (
	"os"
	"path/filepath"
)

const LIST_DEFS_NAME = "ddp_list_types_defs"

var (
	// path to the directory where DDP is installed
	InstallDir string
	// path to the directory of the Duden
	Duden string
	// path to the DDP/bin dir (contains kddp.exe and probably DDPLS.exe)
	Bin string
	// path to the DDP/lib dir (contains ddpstdlib.a and ddpruntime.a and probably their sources)
	Lib string
	// path to the main.o file which contains the main function
	Main_O string
	// path to the ddp_list_types_defs.ll file which contains the textual llvm ir definitions of the inbuilt ddp list types
	DDP_List_Types_Defs_LL string
	// path to the ddp_list_types_defs.ll file which is an object file containing the definitions of the inbuilt ddp list types
	DDP_List_Types_Defs_O string
	// path to the mingw64 directory in the DDP installation directory
	// might not be present
	Mingw64 string
)

func init() {
	// get the path to the ddp install directory
	if ddppath := os.Getenv("DDPPATH"); ddppath != "" {
		InstallDir = ddppath
	} else if exeFolder, err := executableFolder(); err != nil { // fallback if the environment variable is not set, might fail though
		panic(err)
	} else {
		InstallDir, err = filepath.Abs(filepath.Join(exeFolder, "../"))
		if err != nil {
			panic(err)
		}
	}
	Duden = filepath.Join(InstallDir, "Duden")
	Bin = filepath.Join(InstallDir, "bin")
	Lib = filepath.Join(InstallDir, "lib")
	Main_O = filepath.Join(Lib, "main.o")
	DDP_List_Types_Defs_LL = filepath.Join(Lib, LIST_DEFS_NAME+".ll")
	DDP_List_Types_Defs_O = filepath.Join(Lib, LIST_DEFS_NAME+".o")
	Mingw64 = filepath.Join(InstallDir, "mingw64")
}

// Returns same path as Executable, returns just the folder
// path. Excludes the executable name and any trailing slash.
func executableFolder() (string, error) {
	p, err := os.Executable()
	if err != nil {
		return "", err
	}

	return filepath.Dir(filepath.Clean(p)), nil
}
