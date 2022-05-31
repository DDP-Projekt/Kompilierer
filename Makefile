DDP_BIN = ""
STD_BIN = ""
ifeq ($(OS),Windows_NT)
	DDP_BIN := ddp.exe
	STD_BIN := ddpstdlib.lib
else
	DDP_BIN := ddp
	STD_BIN := ddpstdlib.a
endif

OUT_DIR := build/

.DEFAULT_GOAL = all

DDP_DIR = ./cmd/ddp
STD_DIR = ./lib/ddpstdlib

MAKE = make

.PHONY = all make_out_dir make_sub_dirs

all: make_out_dir make_sub_dirs
	cp $(DDP_DIR)/build/$(DDP_BIN) $(OUT_DIR)
	cp $(STD_DIR)/$(STD_BIN) $(OUT_DIR)

make_sub_dirs:
	$(MAKE) -C $(DDP_DIR)
	$(MAKE) -C $(STD_DIR)

make_out_dir:
	mkdir -p $(OUT_DIR)