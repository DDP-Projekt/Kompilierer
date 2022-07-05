DDP_BIN = ""
STD_BIN = ""
ifeq ($(OS),Windows_NT)
	DDP_BIN := kddp.exe
	STD_BIN := ddpstdlib.lib
else
	DDP_BIN := kddp
	STD_BIN := ddpstdlib.a
endif

LLVM_SRC_DIR=./llvm-project/llvm
LLVM_BUILD_DIR=./llvm_build

CC=gcc
CXX=g++
LLVM_BUILD_TYPE=Release
LLVM_CMAKE_GENERATOR="MinGW Makefiles"

OUT_DIR := build/

.DEFAULT_GOAL = all

DDP_DIR = ./cmd/kddp
STD_DIR = ./lib/ddpstdlib

CMAKE = cmake
MAKE = make

.PHONY = all debug make_out_dir kddp ddpstdlib ddpstdlib-debug test llvm

all: make_out_dir kddp ddpstdlib

debug: make_out_dir kddp ddpstdlib-debug

kddp:
	$(MAKE) -C $(DDP_DIR)
	mv $(DDP_DIR)/build/$(DDP_BIN) $(OUT_DIR)

ddpstdlib:
	$(MAKE) -C $(STD_DIR)
	mv $(STD_DIR)/$(STD_BIN) $(OUT_DIR)

ddpstdlib-debug:
	$(MAKE) -C $(STD_DIR) debug
	mv $(STD_DIR)/$(STD_BIN) $(OUT_DIR)

make_out_dir:
	mkdir -p $(OUT_DIR)

test: all
	go test -v ./tests | sed ''/PASS/s//$$(printf "\033[32mPASS\033[0m")/'' | sed ''/FAIL/s//$$(printf "\033[31mFAIL\033[0m")/''

llvm:
# clone the submodule
	git submodule init
	git submodule update

# ignore gopls errors
	cd ./llvm-project ; go mod init ignored || true

# generate cmake build files
	$(CMAKE) -S$(LLVM_SRC_DIR) -B$(LLVM_BUILD_DIR) -DCMAKE_BUILD_TYPE=$(LLVM_BUILD_TYPE) -G$(LLVM_CMAKE_GENERATOR) -DCMAKE_C_COMPILER=$(CC) -DCMAKE_CXX_COMPILER=$(CXX)

# build llvm
	cd $(LLVM_BUILD_DIR) ; $(MAKE)