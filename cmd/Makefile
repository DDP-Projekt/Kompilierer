KDDP_DIR = kddp/
KDDP_OUT_FILE_NAME = ""
DDP_SETUP_DIR = ddp-setup/
DDP_SETUP_OUT_FILE_NAME = ""
LLVM_CONFIG = ""
EXPORTED_CGO_LDFLAGS = ""
ifeq ($(OS),Windows_NT)
	KDDP_OUT_FILE_NAME = kddp.exe
	DDP_SETUP_OUT_FILE_NAME = ddp-setup.exe
	ifneq ("$(wildcard ../llvm_build/bin/llvm-config.exe)","")
		LLVM_CONFIG := "../llvm_build/bin/llvm-config.exe"
	else
		LLVM_CONFIG := "llvm-config"
	endif
	EXPORTED_CGO_LDFLAGS = $(shell $(LLVM_CONFIG) --ldflags --libs --system-libs all) -static -static-libgcc -static-libstdc++
else
	KDDP_OUT_FILE_NAME = kddp
	DDP_SETUP_OUT_FILE_NAME = ddp-setup
	ifneq ("$(wildcard ../llvm_build/bin/llvm-config)","")
		LLVM_CONFIG := "../llvm_build/bin/llvm-config"
	else
		LLVM_CONFIG := "llvm-config-20"
	endif
	EXPORTED_CGO_LDFLAGS = $(shell $(LLVM_CONFIG) --ldflags --libs --system-libs all)
endif

.PHONY: all clean kddp kddp-debug ddp-setup gc_strategy

.DEFAULT_GOAL := all

DDPVERSION := v1.0.0
LLVMVERSION := $(shell $(LLVM_CONFIG) --version)
GCCVERSION := $(shell gcc -dumpfullversion)
GCCVERSIONFULL := $(shell gcc --version | head -n1)

KDDP_LDFLAGS := -s -w -X main.DDPVERSION=$(DDPVERSION) -X main.LLVMVERSION=$(LLVMVERSION) -X main.GCCVERSION=$(GCCVERSION) -X 'main.GCCVERSIONFULL=$(GCCVERSIONFULL)'
# TODO: use build tags instead of -X
KDDP_LDFLAGS_DEBUG := $(KDDP_LDFLAGS) -X 'github.com/DDP-Projekt/Kompilierer/src/compiler.DDP_COMPILER_DEBUG=true'

kddp: export CGO_CPPFLAGS = $(shell $(LLVM_CONFIG) --cppflags)
kddp: export CGO_CXXFLAGS = $(shell $(LLVM_CONFIG) --cxxflags)
kddp: export CGO_LDFLAGS = $(EXPORTED_CGO_LDFLAGS) -L../../lib/gc_strategy -lddp-gc-strategy

kddp-debug: export CGO_CPPFLAGS = $(shell $(LLVM_CONFIG) --cppflags)
kddp-debug: export CGO_CXXFLAGS = $(shell $(LLVM_CONFIG) --cxxflags)
kddp-debug: export CGO_LDFLAGS = $(EXPORTED_CGO_LDFLAGS) -L../../lib/gc_strategy -lddp-gc-strategy

all: kddp kddp-debug ddp-setup

RM = rm -rf

clean:
	$(RM) $(KDDP_DIR)$(KDDP_OUT_FILE_NAME)
	$(RM) $(DDP_SETUP_DIR)$(DDP_SETUP_OUT_FILE_NAME)
	'$(MAKE)' -C '../lib/gc_strategy/' clean

kddp: gc_strategy
	cd $(KDDP_DIR) ; go build -o $(KDDP_OUT_FILE_NAME) -ldflags "$(KDDP_LDFLAGS)"

kddp-debug: gc_strategy
	cd $(KDDP_DIR) ; go build -o $(KDDP_OUT_FILE_NAME) -ldflags "$(KDDP_LDFLAGS_DEBUG)"

gc_strategy:
	'$(MAKE)' -C '../lib/gc_strategy/'

SETUP_LDFLAGS := "-s -w"

ddp-setup:
	cd $(DDP_SETUP_DIR) ; go build -o $(DDP_SETUP_OUT_FILE_NAME) -ldflags $(SETUP_LDFLAGS)
