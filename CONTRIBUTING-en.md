# Contributing to DDP

DDP is and will always be open source and any contributions are welcome! You can contribute to this project by submitting pull requests to the `dev` branch.

Instructions for setting up a development environment are written below.

# Building the DDP Compiler

## Prerequisites

The DDP Compiler is written in Go. Therefore you need to have Golang version 1.22.2 or later installed. You can download it here: https://go.dev/dl/. <br>
NOTE: Some package managers don't install the correct version of Go. Please check your Go version with `go version`.

To build the DDP Compiler, you need to have a build of LLVM 12.0.0 installed.
On many linux distros, this is easily done by running `sudo apt install llvm-12` (or `sudo apt install llvm-12-dev`).
On Windows you need to build LLVM yourself (which you can also do on linux if you wish, but it is not recommended, because it can take several hours).

If you want to build LLVM yourself on linux anyway, skip to the *Building LLVM* section.

To run the makefile on Windows, you also need the following programs installed and added to your PATH:

- [Go](https://go.dev/dl/) (minimum version 1.22.2)
- [mingw64](https://winlibs.com/) (version 12.2.0 with the msvcrt runtime is tested and works, other versions might not work)
- make (can be downloaded via chocolatey, or you can use mingw32-make which comes with mingw64)
- [git](https://git-scm.com/download/win) (you need git-bash to run the makefiles)
- [CMake](https://cmake.org/download/) (version 3.13.4 or above)

### Note for Windows

Your final DDP-build is tightly coupled with the mingw64 version you used.
Also, as DDP uses GCC as linker and libc-provider, the final DDP-build will need the same mingw64 version with which
it was built to compile DDP programs. Keep that in mind when setting DDP on any machine.

## On Linux

After you installed LLVM, simply run `make` from the root of the repo.
To run the tests, use `make test`.
To display a short explanation of all make targets run `make help`.
To build LLVM from the llvm-project submodule, run `make llvm`.
If you built llvm from the submodule, make will use this llvm build instead of the global one.

It is recommended to use the j option of make to use more than one thread. Example: `make -j8`.

## On Windows

After you installed LLVM (see *Building LLVM* or *Precompiled LLVM*), follow the same steps as on Linux, but use git-bash to run the commands (otherwise some unix commands will not work).

### Important

GNUMake will not work. If you have gnu make installed it has the name *make* on windows, so you should use mingw32-make instead when running the makefiles.

# Building LLVM

# Precompiled LLVM

For certain mingw versions, there are precompiled (LLVM libraries)[https://github.com/DDP-Projekt/Kompilierer/releases/tag/llvm-binaries].
These can be downloaded and extracted into the `llvm_build` folder.

The mingw version used must match the LLVM version, otherwise there will be errors.

## Complete Windows Example with Precompiled LLVM and MinGW via Chocolatey

All commands are run from the root of the repo.

```bash
$ choco install mingw --version 12.2.0
# download and extract llvm binaries
$ curl -L -o ./llvm_build.tar.gz https://github.com/DDP-Projekt/Kompilierer/releases/download/llvm-binaries/llvm_build-mingw-12.2.0-x86_64-ucrt-posix-seh.tar.gz
$ mkdir -p ./llvm_build/
$ tar -xzf ./llvm_build.tar.gz -C ./ --force-local
$ rm ./llvm_build.tar.gz
# build the compiler
$ make -j8
# run the tests
$ make test-complete -j8
```

## Prerequisites

- [Python3](https://www.python.org/downloads/)

On Windows, the prerequisites above are also needed, and all the commands below need to be executed from git-bash and not cmd or powershell.

After you installed all the prerequisites open a terminal in the root repository and run `make llvm`.
This should download the llvm-project submodule and build it to llvm_build.
The download is about 1GB, so it might take a while.
Building LLVM also takes 2-3 hours, but it can be left to run in the background as it does not take up many resources.

# Complete Example on Windows

All commands are run from the root of the repo

```
$ make llvm -j8
$ make -j8
```

# Working on the Compiler

## Debugging the Compiler

Because the compiler depends on the C-Bindings of LLVM it cannot be debugged like a normal Go program.
Before debugging some environment variables need to be set. To simplify this process the shell script `open_for_debug.sh` can be used.
It sets the necessary environment variables and then opens VSCode.
For other IDEs the script can be adapted.
