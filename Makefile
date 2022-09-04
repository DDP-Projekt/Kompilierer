DDP_BIN = ""
STD_BIN = libddpstdlib.a
RUN_BIN = libddpruntime.a
ifeq ($(OS),Windows_NT)
	DDP_BIN := kddp.exe
else
	DDP_BIN := kddp
endif

LLVM_SRC_DIR=./llvm-project/llvm
LLVM_BUILD_DIR=./llvm_build

CC=gcc
CXX=g++
LLVM_BUILD_TYPE=Release
LLVM_CMAKE_GENERATOR="MinGW Makefiles"

OUT_DIR := build/kddp

.DEFAULT_GOAL := all

DDP_DIR = ./cmd/kddp
STD_DIR = ./lib/stdlib
RUN_DIR = ./lib/runtime
INCL_DIR = ./lib/runtime/include
DUDEN_DIR = ./lib/stdlib/Duden

DDP_DIR_OUT = $(OUT_DIR)/bin/
LIB_DIR_OUT = $(OUT_DIR)/lib/
INCL_DIR_OUT =  $(OUT_DIR)/include/DDP_Runtime

CMAKE = cmake
MAKE = make

.PHONY = all debug make_out_dir kddp ddpstdlib ddpstdlib-debug duden runtime-include ddpruntime ddpruntime-debug test llvm

all: make_out_dir kddp ddpruntime ddpstdlib duden runtime-include

debug: make_out_dir kddp ddpruntime-debug ddpstdlib-debug duden

kddp:
	cd $(DDP_DIR) ; $(MAKE)
	mv $(DDP_DIR)/build/$(DDP_BIN) $(DDP_DIR_OUT)

ddpstdlib:
	cd $(STD_DIR) ; $(MAKE)
	mv $(STD_DIR)/$(STD_BIN) $(LIB_DIR_OUT)

ddpstdlib-debug:
	cd $(STD_DIR) ; $(MAKE) debug
	mv $(STD_DIR)/$(STD_BIN) $(LIB_DIR_OUT)

duden:
	cp -r $(DUDEN_DIR) $(OUT_DIR)

runtime-include:
	cp -a $(INCL_DIR)/. $(INCL_DIR_OUT)

ddpruntime:
	cd $(RUN_DIR) ; $(MAKE)
	mv $(RUN_DIR)/$(RUN_BIN) $(LIB_DIR_OUT)

ddpruntime-debug:
	cd $(RUN_DIR) ; $(MAKE) debug
	mv $(RUN_DIR)/$(RUN_BIN) $(LIB_DIR_OUT)

make_out_dir:
	mkdir -p $(OUT_DIR)
	mkdir -p $(DDP_DIR_OUT)
	mkdir -p $(LIB_DIR_OUT)
	mkdir -p $(INCL_DIR_OUT)

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