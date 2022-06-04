DDP_BIN = ""
STD_BIN = ""
ifeq ($(OS),Windows_NT)
	DDP_BIN := kddp.exe
	STD_BIN := ddpstdlib.lib
else
	DDP_BIN := kddp
	STD_BIN := ddpstdlib.a
endif

OUT_DIR := build/

.DEFAULT_GOAL = all

DDP_DIR = ./cmd/kddp
STD_DIR = ./lib/ddpstdlib

MAKE = make

.PHONY = all debug make_out_dir kddp ddpstdlib ddpstdlib-debug

all: make_out_dir kddp ddpstdlib
	mv $(DDP_DIR)/build/$(DDP_BIN) $(OUT_DIR)
	mv $(STD_DIR)/$(STD_BIN) $(OUT_DIR)

debug: make_out_dir kddp ddpstdlib-debug
	$(MAKE) -C $(DDP_DIR)
	$(MAKE) -C $(STD_DIR) debug
	mv $(DDP_DIR)/build/$(DDP_BIN) $(OUT_DIR)
	mv $(STD_DIR)/$(STD_BIN) $(OUT_DIR)

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