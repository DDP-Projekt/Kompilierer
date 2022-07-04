# Contributing to DDP

Currently the DDP repos are private and no contributions are taken.

# Building the DDP Compiler

## Prerequisites

To build the DDP Compiler, you need to have a build of LLVM 12.0.0 installed.
On linux this is easily done by running `sudo apt install llvm-12`.
On Windows you need to build LLVM yourself (which you can also do on linux if you wish, but it is not recommended, because it can take several hours).

If you want to build LLVM yourself on linux anyway, skip to the *Building LLVM* section.

To run the makefile on Windows, you also need the following programs installed and added to your PATH:

- [mingw64](https://sourceforge.net/projects/mingw-w64/files/Toolchains%20targetting%20Win64/Personal%20Builds/mingw-builds/8.1.0/threads-posix/seh/x86_64-8.1.0-release-posix-seh-rt_v6-rev0.7z/download) (version 8.1.0 is tested and works, other versions might not work)
- make (comes with mingw64)
- [git](https://git-scm.com/download/win) (you need git-bash to run the makefiles)

## On Linux

After you installed LLVM, simply run `make` from the root of the repo.
To run the tests, use `make test`.
To build LLVM from the llvm-project submodule, run `make llvm`.
If you built llvm from the submodule, make will use this llvm build instead of the global one.

## On Windows

After you installed LLVM (see *Building LLVM*), follow the same steps as on Linux, but use git-bash to run the commands (otherwise some unix commands will not work).

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