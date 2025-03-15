CMD_DIR = ./cmd/
STD_DIR = ./lib/stdlib/
RUN_DIR = ./lib/runtime/
EXT_DIR = ./lib/external/

DDP_SETUP_BUILD_DIR = $(CMD_DIR)ddp-setup/build/

KDDP_BIN = ""
DDP_SETUP_BIN = ""

KDDP_DIR = $(CMD_DIR)kddp/
DDP_SETUP_DIR = $(CMD_DIR)ddp-setup/

STD_BIN = libddpstdlib.a
EXT_BIN_PCRE2 = libpcre2-8.a
EXT_BIN_LIBAR = libarchive.a
EXT_BIN_LIBZ = libz.a
EXT_BIN_LIBLZMA = liblzma.a
EXT_BIN_LIBBZ2 = libbz2.a
EXT_BIN_LIBLZ4 = liblz4.a
PCRE2_DIR = $(EXT_DIR)pcre2_build/
PCRE2_HEADERS = $(PCRE2_DIR)pcre2.h
PCRE2_HEADERS_OUT_DIR = $(STD_DIR_OUT)include/
LIBAR_DIR = $(EXT_DIR)libarchive/libarchive/
LIBAR_HEADERS = $(wildcard $(LIBAR_DIR)*.h)
LIBAR_HEADERS_OUT_DIR = $(STD_DIR_OUT)include/
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
SED = sed -u
TAR = tar

define cp_if_exists
	@if [ -f $(1) ]; then \
		echo copying $(1) to $(2); \
		$(CP) $(1) $(2); \
	fi
endef

LLVM_BUILD_TYPE=Release
LLVM_CMAKE_GENERATOR="MinGW Makefiles"
LLVM_CMAKE_BUILD_TOOL=$(MAKE)
LLVM_TARGETS="X86"
LLVM_ADDITIONAL_CMAKE_VARIABLES= -DCMAKE_INSTALL_PREFIX=llvm_build/  -DLLVM_BUILD_TOOLS=OFF -DLLVM_ENABLE_BINDINGS=OFF -DLLVM_ENABLE_UNWIND_TABLES=OFF -DLLVM_INCLUDE_BENCHMARKS=OFF -DLLVM_INCLUDE_EXAMPLES=OFF -DLLVM_INCLUDE_TESTS=OFF

ifeq ($(OS),Windows_NT)
	KDDP_BIN = kddp.exe
	DDP_SETUP_BIN = ddp-setup.exe
else
	KDDP_BIN = kddp
	DDP_SETUP_BIN = ddp-setup
	LLVM_CMAKE_GENERATOR="Unix Makefiles"
endif

OUT_DIR = ./build/DDP/

.DEFAULT_GOAL = all

KDDP_DIR_OUT = $(OUT_DIR)bin/
DDP_SETUP_DIR_OUT = $(OUT_DIR)
LIB_DIR_OUT = $(OUT_DIR)lib/
STD_DIR_OUT = $(LIB_DIR_OUT)stdlib/
RUN_DIR_OUT = $(LIB_DIR_OUT)runtime/

CMAKE = cmake

SHELL = /bin/bash
.SHELLFLAGS = -o pipefail -c

.PHONY: all debug kddp ddp-setup stdlib-copies stdlib stdlib-debug runtime-copies runtime runtime-debug external-compile external external-headers clean-cmd clean-runtime clean-stdlib clean clean-outdir format-stdlib format-runtime format checkout-llvm llvm test-normal test-memory test-normal-memory test-sumtypes coverage test test-with-optimizations help

all: kddp runtime stdlib ddp-setup $(OUT_DIR)LICENSE $(OUT_DIR)README.md ## compiles kdddp, the runtime, the stdlib and ddp-setup into the build/DDP/ directory 

debug: kddp runtime-debug stdlib-debug ## same as all but the runtime and stdlib print debugging information

%/:
	$(MKDIR) $@

kddp: $(KDDP_DIR_OUT) $(LIB_DIR_OUT) ## compiles kddp into build/DDP/bin/
	@echo "building kddp"
	'$(MAKE)' -C $(CMD_DIR) kddp
	$(CP) $(KDDP_DIR)$(KDDP_BIN) $(KDDP_DIR_OUT)$(KDDP_BIN)
	$(KDDP_DIR_OUT)$(KDDP_BIN) dump-list-defs -o $(LIB_DIR_OUT)$(DDP_LIST_DEFS_NAME) $(DDP_LIST_DEFS_OUTPUT_TYPES)

ddp-setup: $(DDP_SETUP_DIR_OUT) ## compiles ddp-setup into build/DDP/bin/
	@echo "building ddp-setup"
	'$(MAKE)' -C $(CMD_DIR) ddp-setup
	$(CP) $(DDP_SETUP_DIR)$(DDP_SETUP_BIN) $(DDP_SETUP_DIR_OUT)$(DDP_SETUP_BIN)

stdlib-copies: $(STD_DIR_OUT)
	$(CP) $(STD_DIR)include/ $(STD_DIR)source/ $(STD_DIR)Makefile $(STD_DIR_OUT)
	$(CP) $(STD_DIR)Duden/ $(OUT_DIR)

stdlib: external stdlib-copies $(LIB_DIR_OUT) ## compiles the stdlib and the Duden into build/DDP/lib/stdlib and build/DDP/Duden
	'$(MAKE)' -C $(STD_DIR)
	$(CP) $(STD_DIR)$(STD_BIN) $(LIB_DIR_OUT)$(STD_BIN)

stdlib-debug: external stdlib-copies $(LIB_DIR_OUT) ## same as stdlib but will print debugging information
	'$(MAKE)' -C $(STD_DIR) debug
	$(CP) $(STD_DIR)$(STD_BIN_DEBUG) $(LIB_DIR_OUT)$(STD_BIN)

runtime-copies: $(RUN_DIR_OUT)
	$(CP) $(RUN_DIR)include/ $(RUN_DIR)source/ $(RUN_DIR)Makefile $(RUN_DIR_OUT)

runtime: runtime-copies $(LIB_DIR_OUT) ## compiles the runtime into build/DDP/lib/stdlib
	'$(MAKE)' -C $(RUN_DIR)
	$(CP) $(RUN_DIR)$(RUN_BIN) $(LIB_DIR_OUT)$(RUN_BIN)
	$(CP) $(RUN_BIN_MAIN_DIR)$(RUN_BIN_MAIN) $(LIB_DIR_OUT)$(RUN_BIN_MAIN)

runtime-debug: runtime-copies $(LIB_DIR_OUT) ## same as runtime but prints debugging information
	'$(MAKE)' -C $(RUN_DIR) debug
	$(CP) $(RUN_DIR)$(RUN_BIN_DEBUG) $(LIB_DIR_OUT)$(RUN_BIN)
	$(CP) $(RUN_BIN_MAIN_DIR)$(RUN_BIN_MAIN_DEBUG) $(LIB_DIR_OUT)$(RUN_BIN_MAIN)

external-compile:
	@echo "building all external libraries"
	'$(MAKE)' -C $(EXT_DIR)

external-headers: external-compile $(LIBAR_HEADERS_OUT_DIR) $(PCRE2_HEADERS_OUT_DIR)
	$(CP) $(LIBAR_HEADERS) $(LIBAR_HEADERS_OUT_DIR)
	$(CP) $(PCRE2_HEADERS) $(PCRE2_HEADERS_OUT_DIR)

external: external-headers
	$(CP) $(EXT_DIR)$(EXT_BIN_PCRE2) $(EXT_DIR)$(EXT_BIN_LIBAR) $(EXT_DIR)$(EXT_BIN_LIBZ) $(EXT_DIR)$(EXT_BIN_LIBLZMA) $(EXT_DIR)$(EXT_BIN_LIBBZ2) $(EXT_DIR)$(EXT_BIN_LIBLZ4) $(LIB_DIR_OUT)

$(OUT_DIR)LICENSE: LICENSE $(OUT_DIR)
	$(CP) LICENSE $(OUT_DIR)

$(OUT_DIR)README.md: README.md $(OUT_DIR)
	$(CP) README.md $(OUT_DIR)

clean-cmd:
	'$(MAKE)' -C $(CMD_DIR) clean
clean-runtime:
	'$(MAKE)' -C $(RUN_DIR) clean
clean-stdlib:
	'$(MAKE)' -C $(STD_DIR) clean

clean: clean-outdir clean-cmd clean-runtime clean-stdlib ## deletes everything produced by this Makefile

clean-outdir: ## deletes build/DDP/
	$(RM) $(OUT_DIR)

clean-all: clean
	'$(MAKE)' -C $(EXT_DIR) clean

format-stdlib:
	'$(MAKE)' -C $(STD_DIR) format

format-runtime:
	'$(MAKE)' -C $(STD_DIR) format

format: format-stdlib format-runtime

checkout-llvm: ## clones the llvm-project submodule
# clone the submodule
	@echo "cloning the llvm repo"
	git submodule update --init llvm-project

# ignore gopls errors
	cd ./llvm-project ; go mod init ignored || true

llvm: checkout-llvm ## compiles llvm
# generate cmake build files
	@echo "building llvm"
	$(CMAKE) -S$(LLVM_SRC_DIR) -B$(LLVM_BUILD_DIR) -DCMAKE_BUILD_TYPE=$(LLVM_BUILD_TYPE) -G$(LLVM_CMAKE_GENERATOR) -DCMAKE_C_COMPILER=$(CC) -DCMAKE_CXX_COMPILER=$(CXX) -DLLVM_TARGETS_TO_BUILD=$(LLVM_TARGETS) $(LLVM_ADDITIONAL_CMAKE_VARIABLES)

# build llvm
	cd $(LLVM_BUILD_DIR) ; MAKEFLAGS='$(MAKEFLAGS)' $(CMAKE) --build . --target llvm-libraries llvm-config llvm-headers install-llvm-headers

# will hold the directories to run in the tests
# if empty, all directories are run
TEST_DIRS = 
# will hold additional arguments to pass to kddp
KDDP_ARGS = 

test-unit:
	go test $(shell go list ./src/... | grep -v compiler)

test-normal: all ## runs the tests
	go test -v ./tests '-run=(TestKDDP|TestStdlib|TestBuildExamples|TestStdlibCoverage)' -test_dirs="$(TEST_DIRS)" -kddp_args="$(KDDP_ARGS)" | $(SED) ''/PASS/s//$$(printf "\033[32mPASS\033[0m")/'' | $(SED) ''/FAIL/s//$$(printf "\033[31mFAIL\033[0m")/''

test-memory: debug ## runs the tests checking for memory leaks
	go test -v ./tests '-run=(TestMemory)' -test_dirs="$(TEST_DIRS)" -kddp_args="$(KDDP_ARGS)" | $(SED) -u ''/PASS/s//$$(printf "\033[32mPASS\033[0m")/'' | $(SED) -u ''/FAIL/s//$$(printf "\033[31mFAIL\033[0m")/''

test-normal-memory: ## runs test-normal and test-memory in the correct order
	'$(MAKE)' test-normal 
	'$(MAKE)' test-memory
	'$(MAKE)' all

test-sumtypes: ## validates that sumtypes in the source tree are correctly used
	go run github.com/BurntSushi/go-sumtype@latest $(shell go list ./... | grep -v vendor )

coverage: all ## creates a coverage report for tests/testdata/stdlib
	go test -v ./tests '-run=TestStdlibCoverage' | $(SED) -u ''/PASS/s//$$(printf "\033[32mPASS\033[0m")/'' | $(SED) -u ''/FAIL/s//$$(printf "\033[31mFAIL\033[0m")/''

test: test-unit test-normal-memory coverage ## runs all the tests

test-with-optimizations: ## runs all tests with full optimizations enabled
	'$(MAKE)' KDDP_ARGS="-O 2" test

help: ## Show this help.
	@egrep -h '\s##\s' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m  %-30s\033[0m %s\n", $$1, $$2}'
