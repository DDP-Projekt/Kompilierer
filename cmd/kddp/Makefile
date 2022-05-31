OUT_FILE_NAME = ""
LLVM_CONFIG = ""
ifeq ($(OS),Windows_NT)
	OUT_FILE_NAME := kddp.exe
	LLVM_CONFIG := "llvm-config"
else
	OUT_FILE_NAME := kddp
	LLVM_CONFIG := "llvm-config-12"
endif

.PHONY = all build make_out_dir

.DEFAULT_GOAL := all

OUT_DIR := build/

build: export CGO_CPPFLAGS = $(shell $(LLVM_CONFIG) --cppflags)
build: export CGO_CXXFLAGS = -std=c++14
build: export CGO_LDFLAGS = $(shell $(LLVM_CONFIG) --ldflags --libs --system-libs all)

all: build

build: make_out_dir
	go build -o $(OUT_DIR)$(OUT_FILE_NAME) -tags byollvm

make_out_dir:
	mkdir -p $(OUT_DIR)