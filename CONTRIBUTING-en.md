# Contributing to DDP

DDP is and will always be open source and any contributions are welcome! You can contribute to this project by submitting pull requests.

Instructions for setting up a development environment are written below.

# Building the DDP Compiler

## Prerequisites

The DDP Compiler is written in Go. Therefore you need to have Golang version 1.18 or later installed. You can download it here: https://go.dev/dl/. <br>
NOTE: Some package managers don't install the correct version of Go. Please check your Go version with `go version`.

To build the DDP Compiler, you need to have a build of LLVM 12.0.0 installed.
On many linux distros, this is easily done by running `sudo apt install llvm-12`.
On Windows you need to build LLVM yourself (which you can also do on linux if you wish, but it is not recommended, because it can take several hours).

If you want to build LLVM yourself on linux anyway, skip to the *Building LLVM* section.

To run the makefile on Windows, you also need the following programs installed and added to your PATH:

- [Go](https://go.dev/dl/) (minimum version 1.18)
- [mingw64](https://winlibs.com/) (version 12.2.0 with the msvcrt runtime is tested and works, other versions might not work)
- make (can be downloaded via chocolatey, or you can use mingw32-make which comes with mingw64)
- [git](https://git-scm.com/download/win) (you need git-bash to run the makefiles)

### Note for Windows

Your final DDP-build is tightly coupled with the mingw64 version you used.
LLVM and the DDP runtime and stdlib must be built with the same version of mingw64 in order to work together.
Also, as DDP uses GCC as linker and libc-provider, the final DDP-build will need the same mingw64 version with which
it was built to compile DDP programs. Keep that in mind when setting DDP on any machine.

## On Linux

After you installed LLVM, simply run `make` from the root of the repo.
To run the tests, use `make test`.
To build LLVM from the llvm-project submodule, run `make llvm`.
If you built llvm from the submodule, make will use this llvm build instead of the global one.

## On Windows

After you installed LLVM (see *Building LLVM*), follow the same steps as on Linux, but use git-bash to run the commands (otherwise some unix commands will not work).

### Important
GNUMake will not work. If you have gnu make installed it has the name *make* on windows, so you should use mingw32-make instead when running the makefiles.

# Building LLVM

## Prerequisites

- [CMake](https://cmake.org/download/) (version 3.13.4 or above)
- [Python3](https://www.python.org/downloads/)

On Windows, the prerequisites above are also needed, and all the commands below need to be executed from git-bash and not cmd or powershell.

After you installed all the prerequisites open a terminal in the root repository and run `make llvm`.
This should download the llvm-project submodule and build it to llvm_build.
The download is about 1GB, so it might take a while.
Building LLVM also takes 2-3 hours, but it can be left to run in the background as it does not take up many resources.

# Complete Example on Windows

All commands are run from the root of the repo

```
$ make llvm
$ make
```
