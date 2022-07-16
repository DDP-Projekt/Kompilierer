DDP_BIN = ""
STD_BIN = ""
RUN_BIN = ""
ifeq ($(OS),Windows_NT)
	DDP_BIN := kddp.exe
	STD_BIN := ddpstdlib.lib
	RUN_BIN := ddpruntime.lib
else
	DDP_BIN := kddp
	STD_BIN := ddpstdlib.a
	RUN_BIN := ddpruntime.a
endif

LLVM_SRC_DIR=./llvm-project/llvm
LLVM_BUILD_DIR=./llvm_build

CC=gcc
CXX=g++
LLVM_BUILD_TYPE=Release
LLVM_CMAKE_GENERATOR="MinGW Makefiles"

OUT_DIR := build/

.DEFAULT_GOAL := all

DDP_DIR = ./cmd/kddp
STD_DIR = ./lib/stdlib
RUN_DIR = ./lib/runtime
DUDEN_DIR = ./lib/stdlib/Duden

CMAKE = cmake
MAKE = make

.PHONY = all debug make_out_dir kddp ddpstdlib ddpstdlib-debug copy-duden ddpruntime ddpruntime-debug test llvm

all: make_out_dir kddp ddpruntime ddpstdlib copy-duden

debug: make_out_dir kddp ddpruntime-debug ddpstdlib-debug copy-duden

kddp:
	cd $(DDP_DIR) ; $(MAKE)
	mv $(DDP_DIR)/build/$(DDP_BIN) $(OUT_DIR)

ddpstdlib:
	cd $(STD_DIR) ; $(MAKE)
	mv $(STD_DIR)/$(STD_BIN) $(OUT_DIR)

ddpstdlib-debug:
	cd $(STD_DIR) ; $(MAKE) debug
	mv $(STD_DIR)/$(STD_BIN) $(OUT_DIR)

copy-duden:
	cp -r $(DUDEN_DIR) $(OUT_DIR)

ddpruntime:
	cd $(RUN_DIR) ; $(MAKE)
	mv $(RUN_DIR)/$(RUN_BIN) $(OUT_DIR)

ddpruntime-debug:
	cd $(RUN_DIR) ; $(MAKE) debug
	mv $(RUN_DIR)/$(RUN_BIN) $(OUT_DIR)

make_out_dir:
	mkdir -p $(OUT_DIR)

test:
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