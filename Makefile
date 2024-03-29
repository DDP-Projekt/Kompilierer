KDDP_DIR = ./cmd/kddp/
STD_DIR = ./lib/stdlib/
RUN_DIR = ./lib/runtime/

KDDP_BIN = ""
STD_BIN = libddpstdlib.a
STD_BIN_PCRE2 = libpcre2-8.a
STD_BIN_DEBUG = $(STD_BIN:.a=debug.a)
RUN_BIN = libddpruntime.a
RUN_BIN_DEBUG = $(RUN_BIN:.a=debug.a)
RUN_BIN_MAIN_DIR = $(RUN_DIR)source/
RUN_BIN_MAIN = main.o
RUN_BIN_MAIN_DEBUG = $(RUN_BIN_MAIN:.o=_debug.o)
DDP_LIST_DEFS_NAME = ddp_list_types_defs

DDP_LIST_DEFS_OUTPUT_TYPES = --llvm-ir --object

LLVM_SRC_DIR=./llvm-project/llvm/
LLVM_BUILD_DIR=./llvm_build/

CC=gcc
CXX=g++

RM = rm -rf
CP = cp -rf
MKDIR = mkdir -p

LLVM_BUILD_TYPE=Release
LLVM_CMAKE_GENERATOR="MinGW Makefiles"
LLVM_CMAKE_BUILD_TOOL=$(MAKE)
LLVM_TARGETS="X86;AArch64"
LLVM_ADDITIONAL_CMAKE_VARIABLES= -DLLVM_BUILD_TOOLS=OFF -DLLVM_ENABLE_BINDINGS=OFF -DLLVM_ENABLE_UNWIND_TABLES=OFF -DLLVM_INCLUDE_BENCHMARKS=OFF -DLLVM_INCLUDE_EXAMPLES=OFF -DLLVM_INCLUDE_TESTS=OFF 

ifeq ($(OS),Windows_NT)
	KDDP_BIN = kddp.exe
else
	KDDP_BIN = kddp
	LLVM_CMAKE_GENERATOR="Unix Makefiles"
endif

# check if ninja is installed and use it
ifneq (, $(shell which ninja))
	LLVM_CMAKE_GENERATOR=Ninja
	LLVM_CMAKE_BUILD_TOOL=ninja
endif

OUT_DIR = ./build/DDP/

.DEFAULT_GOAL = all

KDDP_DIR_OUT = $(OUT_DIR)bin/
LIB_DIR_OUT = $(OUT_DIR)lib/
STD_DIR_OUT = $(LIB_DIR_OUT)stdlib/
RUN_DIR_OUT = $(LIB_DIR_OUT)runtime/

CMAKE = cmake

SHELL = /bin/bash
.SHELLFLAGS = -o pipefail -c

.PHONY = all clean clean-outdir debug kddp stdlib stdlib-debug runtime runtime-debug test test-memory download-llvm llvm help test-complete download-pcre2

all: $(OUT_DIR) kddp runtime stdlib

debug: $(OUT_DIR) kddp runtime-debug stdlib-debug

kddp:
	@echo "building kddp"
	cd $(KDDP_DIR) ; '$(MAKE)'
	$(CP) $(KDDP_DIR)build/$(KDDP_BIN) $(KDDP_DIR_OUT)$(KDDP_BIN)
	$(KDDP_DIR_OUT)$(KDDP_BIN) dump-list-defs -o $(LIB_DIR_OUT)$(DDP_LIST_DEFS_NAME) $(DDP_LIST_DEFS_OUTPUT_TYPES)

stdlib:
	@echo "building the ddp-stdlib"
	cd $(STD_DIR) ; '$(MAKE)'
	$(CP) $(STD_DIR)$(STD_BIN) $(LIB_DIR_OUT)$(STD_BIN)
	@if [ -f $(STD_DIR)$(STD_BIN_PCRE2) ]; then \
		$(CP) $(STD_DIR)$(STD_BIN_PCRE2) $(LIB_DIR_OUT)$(STD_BIN_PCRE2); \
	fi
	$(CP) $(STD_DIR)include/ $(STD_DIR_OUT)
	$(CP) $(STD_DIR)source/ $(STD_DIR_OUT)
	$(CP) $(STD_DIR)Duden/ $(OUT_DIR)
	@if [ -d ./lib/stdlib/pcre2 ]; then \
		$(CP) $(STD_DIR)pcre2/ $(STD_DIR_OUT); \
	fi
	$(CP) $(STD_DIR)Makefile $(STD_DIR_OUT)Makefile

stdlib-debug:
	@echo "building the ddp-stdlib in debug mode"
	cd $(STD_DIR) ; '$(MAKE)' debug
	$(CP) $(STD_DIR)$(STD_BIN_DEBUG) $(LIB_DIR_OUT)$(STD_BIN)
	@if [ -f $(STD_DIR)$(STD_BIN_PCRE2) ]; then \
		$(CP) $(STD_DIR)$(STD_BIN_PCRE2) $(LIB_DIR_OUT)$(STD_BIN_PCRE2); \
	fi
	$(CP) $(STD_DIR)include/ $(STD_DIR_OUT)
	$(CP) $(STD_DIR)source/ $(STD_DIR_OUT)
	$(CP) $(STD_DIR)Duden/ $(OUT_DIR)
	@if [ -d ./lib/stdlib/pcre2 ]; then \
		$(CP) $(STD_DIR)pcre2/ $(STD_DIR_OUT); \
	fi
	$(CP) $(STD_DIR)Makefile $(STD_DIR_OUT)Makefile

runtime:
	@echo "building the ddp-runtime"
	cd $(RUN_DIR) ; '$(MAKE)'
	$(CP) $(RUN_DIR)$(RUN_BIN) $(LIB_DIR_OUT)$(RUN_BIN)
	$(CP) $(RUN_BIN_MAIN_DIR)$(RUN_BIN_MAIN) $(LIB_DIR_OUT)$(RUN_BIN_MAIN)
	$(CP) $(RUN_DIR)include/ $(RUN_DIR_OUT)
	$(CP) $(RUN_DIR)source/ $(RUN_DIR_OUT)
	$(CP) $(RUN_DIR)Makefile $(RUN_DIR_OUT)Makefile

runtime-debug:
	@echo "building the ddp-runtime in debug mode"
	cd $(RUN_DIR) ; '$(MAKE)' debug
	@echo copying $(RUN_DIR)$(RUN_BIN_DEBUG) to $(LIB_DIR_OUT)$(RUN_BIN)
	$(CP) $(RUN_DIR)$(RUN_BIN_DEBUG) $(LIB_DIR_OUT)$(RUN_BIN)
	$(CP) $(RUN_BIN_MAIN_DIR)$(RUN_BIN_MAIN_DEBUG) $(LIB_DIR_OUT)$(RUN_BIN_MAIN)
	$(CP) $(RUN_DIR)include/ $(RUN_DIR_OUT)
	$(CP) $(RUN_DIR)source/ $(RUN_DIR_OUT)
	$(CP) $(RUN_DIR)Makefile $(RUN_DIR_OUT)Makefile

$(OUT_DIR): LICENSE README.md
	@echo "creating output directories"
	$(MKDIR) $(OUT_DIR)
	$(MKDIR) $(KDDP_DIR_OUT)
	$(MKDIR) $(OUT_DIR)Duden/
	$(MKDIR) $(STD_DIR_OUT)include/
	$(MKDIR) $(STD_DIR_OUT)source/
	$(MKDIR) $(RUN_DIR_OUT)include/
	$(MKDIR) $(RUN_DIR_OUT)source/
	$(CP) LICENSE $(OUT_DIR)
	$(CP) README.md $(OUT_DIR)

clean: clean-outdir
	cd $(KDDP_DIR) ; '$(MAKE)' clean
	cd $(STD_DIR) ; '$(MAKE)' clean
	cd $(RUN_DIR) ; '$(MAKE)' clean

clean-outdir:
	@echo "deleting output directorie"
	$(RM) $(OUT_DIR)

download-pcre2:
	@echo "downloading pcre2.tar.gz"
	curl -L https://github.com/PCRE2Project/pcre2/releases/download/pcre2-10.42/pcre2-10.42.tar.gz -o pcre2.tar.gz
	mkdir $(STD_DIR)pcre2
	tar -xzf pcre2.tar.gz -C $(STD_DIR)pcre2 --strip-components=1
	rm ./pcre2.tar.gz
	cd $(STD_DIR)pcre2 ; ./configure

download-llvm:
# clone the submodule
	@echo "cloning the llvm repo"
	git submodule init
	git submodule update

# ignore gopls errors
	cd ./llvm-project ; go mod init ignored || true

llvm: download-llvm
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
	go test -v ./tests '-run=(TestKDDP|TestStdlib|TestBuildExamples)' -test_dirs="$(TEST_DIRS)" | sed ''/PASS/s//$$(printf "\033[32mPASS\033[0m")/'' | sed ''/FAIL/s//$$(printf "\033[31mFAIL\033[0m")/''

test-memory:
	go test -v ./tests '-run=(TestMemory)' -test_dirs="$(TEST_DIRS)" | sed ''/PASS/s//$$(printf "\033[32mPASS\033[0m")/'' | sed ''/FAIL/s//$$(printf "\033[31mFAIL\033[0m")/''

# runs all tests and test-memory
# everything is done manually to ensure the build is finished
# before the tests even with -j n
test-complete: 
	'$(MAKE)' all 
	'$(MAKE)' test 
	'$(MAKE)' debug 
	'$(MAKE)' test-memory

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
