DDP_BIN = ""
STD_BIN = libddpstdlib.a
RUN_BIN = libddpruntime.a

LLVM_SRC_DIR=./llvm-project/llvm
LLVM_BUILD_DIR=./llvm_build

CC=gcc
CXX=g++
LLVM_BUILD_TYPE=Release
LLVM_CMAKE_GENERATOR="MinGW Makefiles"
LLVM_CMAKE_BUILD_TOOL=$(MAKE)
LLVM_TARGETS="X86;AArch64;ARM"
LLVM_ADDITIONAL_CMAKE_VARIABLES= -DLLVM_BUILD_TOOLS=OFF -DLLVM_ENABLE_BINDINGS=OFF -DLLVM_ENABLE_UNWIND_TABLES=OFF -DLLVM_INCLUDE_BENCHMARKS=OFF -DLLVM_INCLUDE_EXAMPLES=OFF -DLLVM_INCLUDE_TESTS=OFF 

ifeq ($(OS),Windows_NT)
	DDP_BIN = kddp.exe
else
	DDP_BIN = kddp
	LLVM_CMAKE_GENERATOR="Unix Makefiles"
endif

# check if ninja is installed and use it
ifneq (, $(shell which ninja))
	LLVM_CMAKE_GENERATOR=Ninja
	LLVM_CMAKE_BUILD_TOOL=ninja
endif

OUT_DIR = ./build/DDP

.DEFAULT_GOAL = all

DDP_DIR = ./cmd/kddp
STD_DIR = ./lib/stdlib
RUN_DIR = ./lib/runtime
INCL_DIR = ./lib/runtime/include
DUDEN_DIR = ./lib/stdlib/Duden

DDP_DIR_OUT = $(OUT_DIR)/bin/
LIB_DIR_OUT = $(OUT_DIR)/lib/

CMAKE = cmake

SHELL = /bin/bash
.SHELLFLAGS = -o pipefail -c

.PHONY = all debug make_out_dir kddp stdlib stdlib-debug runtime runtime-debug test llvm help display_help_disclaimer test-complete

display_help_disclaimer:
	@echo "compiling the whole project"
	@echo 'run "make help" to get a list of all available targets'
	@echo ""

all: display_help_disclaimer make_out_dir kddp runtime stdlib

debug: make_out_dir kddp runtime-debug stdlib-debug

kddp:
	@echo "building kddp"
	cd $(DDP_DIR) ; $(MAKE)
	mv -f $(DDP_DIR)/build/$(DDP_BIN) $(DDP_DIR_OUT)

stdlib:
	@echo "building the ddp-stdlib"
	cd $(STD_DIR) ; $(MAKE)
	mv -f $(STD_DIR)/$(STD_BIN) $(LIB_DIR_OUT)
	cp -r $(STD_DIR) $(LIB_DIR_OUT)
	rm -rf $(OUT_DIR)/Duden || true
	mv -f $(LIB_DIR_OUT)stdlib/Duden $(OUT_DIR)

stdlib-debug:
	@echo "building the ddp-stdlib in debug mode"
	cd $(STD_DIR) ; $(MAKE) debug
	mv -f $(STD_DIR)/$(STD_BIN) $(LIB_DIR_OUT)
	cp -r $(STD_DIR) $(LIB_DIR_OUT)
	rm -rf $(OUT_DIR)/Duden || true
	mv -f $(LIB_DIR_OUT)stdlib/Duden $(OUT_DIR)

runtime:
	@echo "building the ddp-runtime"
	cd $(RUN_DIR) ; $(MAKE)
	mv -f $(RUN_DIR)/$(RUN_BIN) $(LIB_DIR_OUT)
	cp -r $(RUN_DIR) $(LIB_DIR_OUT)

runtime-debug:
	@echo "building the ddp-runtime in debug mode"
	cd $(RUN_DIR) ; $(MAKE) debug
	mv -f $(RUN_DIR)/$(RUN_BIN) $(LIB_DIR_OUT)
	cp -r $(RUN_DIR) $(LIB_DIR_OUT)

make_out_dir:
	@echo "creating output directories"
	mkdir -p $(OUT_DIR)
	mkdir -p $(DDP_DIR_OUT)
	mkdir -p $(LIB_DIR_OUT)
	cp LICENSE $(OUT_DIR)
	cp README.md $(OUT_DIR)

clean:
	@echo "deleting output directorie"
	rm -r $(OUT_DIR) || true
	cd $(STD_DIR) ; $(MAKE) clean
	cd $(RUN_DIR) ; $(MAKE) clean

llvm:
# clone the submodule
	@echo "cloning the llvm repo"
	git submodule init
	git submodule update

# ignore gopls errors
	cd ./llvm-project ; go mod init ignored || true

# generate cmake build files
	@echo "building llvm"
ifeq ($(LLVM_CMAKE_GENERATOR),Ninja)
	@echo "found ninja, using it as cmake generator"
endif
	$(CMAKE) -S$(LLVM_SRC_DIR) -B$(LLVM_BUILD_DIR) -DCMAKE_BUILD_TYPE=$(LLVM_BUILD_TYPE) -G$(LLVM_CMAKE_GENERATOR) -DCMAKE_C_COMPILER=$(CC) -DCMAKE_CXX_COMPILER=$(CXX) -DLLVM_TARGETS_TO_BUILD=$(LLVM_TARGETS) $(LLVM_ADDITIONAL_CMAKE_VARIABLES)

# build llvm
	cd $(LLVM_BUILD_DIR) ; $(LLVM_CMAKE_BUILD_TOOL) ; $(LLVM_CMAKE_BUILD_TOOL) llvm-config


# will hold the directories to run in the tests
# if empty, all directories are run
TEST_DIRS = 

test:
	go test -v ./tests '-run=(TestKDDP|TestStdlib)' -test_dirs="$(TEST_DIRS)" | sed ''/PASS/s//$$(printf "\033[32mPASS\033[0m")/'' | sed ''/FAIL/s//$$(printf "\033[31mFAIL\033[0m")/''

test-memory:
	go test -v ./tests '-run=(TestMemory)' -test_dirs="$(TEST_DIRS)" | sed ''/PASS/s//$$(printf "\033[32mPASS\033[0m")/'' | sed ''/FAIL/s//$$(printf "\033[31mFAIL\033[0m")/''


test-complete: clean all test clean debug test-memory

help:
	@echo "Targets:"
	@echo "    all (default target): compile kddp the runtime and the stdlib into $(OUT_DIR)"
	@echo "    debug: compile kddp the runtime and the stdlib into $(OUT_DIR) in debug mode"
	@echo "    kddp: compile kddp into $(OUT_DIR)"
	@echo "    stdlib: compile only the stdlib into $(OUT_DIR)"
	@echo "    stdlib-debug: compile only the stdlib in debug mode into $(OUT_DIR)"
	@echo "    runtime: compile only the runtime into $(OUT_DIR)"
	@echo "    runtime-debug: compile only the runtime in debug mode into $(OUT_DIR)"
	@echo "    clean: delete the output directorie $(OUT_DIR)"
	@echo "    llvm: clone the llvm-project repo at version 12.0.0 and build it"
	@echo "    test: run the ddp tests"
	@echo "          you can specifiy directorie names with the TEST_DIRS variable"
	@echo "          to only run those tests"
	@echo '          example: make test TEST_DIRS="slicing assignement if"'
	@echo "    test-memory: run the ddp tests and test for memory leaks"
	@echo '          the runtime and stdlib have to be compiled in debug mode beforehand'
	@echo "    test-complete: runs test and test-memory and automatically"
	@echo '          compiles everything correctly beforehand'
