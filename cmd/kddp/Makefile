OUT_FILE_NAME = ""
LLVM_CONFIG = ""
EXPORTED_CGO_LDFLAGS = ""
ifeq ($(OS),Windows_NT)
	OUT_FILE_NAME := kddp.exe
	ifneq ("$(wildcard ../../llvm_build/bin/llvm-config.exe)","")
		LLVM_CONFIG := "../../llvm_build/bin/llvm-config.exe"
	else
		LLVM_CONFIG := "llvm-config"
	endif
	EXPORTED_CGO_LDFLAGS = $(shell $(LLVM_CONFIG) --ldflags --libs --system-libs all) -static -static-libgcc -static-libstdc++
else
	OUT_FILE_NAME := kddp
	ifneq ("$(wildcard ../../llvm_build/bin/llvm-config)","")
		LLVM_CONFIG := "../../llvm_build/bin/llvm-config"
	else
		LLVM_CONFIG := "llvm-config-12"
	endif
	EXPORTED_CGO_LDFLAGS = $(shell $(LLVM_CONFIG) --ldflags --libs --system-libs all)
endif

.PHONY = all clean

.DEFAULT_GOAL := all

OUT_DIR := build/

DDPVERSION := v0.0.4-pre
LLVMVERSION := $(shell $(LLVM_CONFIG) --version)
GCCVERSION := $(shell gcc -dumpfullversion)
GCCVERSIONFULL := $(shell gcc --version | head -n1)

LDFLAGS := "-s -w -X main.DDPVERSION=$(DDPVERSION) -X main.LLVMVERSION=$(LLVMVERSION) -X main.GCCVERSION=$(GCCVERSION) -X 'main.GCCVERSIONFULL=$(GCCVERSIONFULL)'"

kddp: export CGO_CPPFLAGS = $(shell $(LLVM_CONFIG) --cppflags)
kddp: export CGO_CXXFLAGS = -std=c++14
kddp: export CGO_LDFLAGS = $(EXPORTED_CGO_LDFLAGS)

all: kddp

clean:
	rm -rf $(OUT_DIR)

kddp: $(OUT_DIR)
	go build -o $(OUT_DIR)$(OUT_FILE_NAME) -tags byollvm -ldflags $(LDFLAGS)

$(OUT_DIR):
	mkdir -p $(OUT_DIR)