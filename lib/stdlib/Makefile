OUT_FILE_NAME = libddpstdlib.a
OUT_FILE_NAME_DEBUG = $(OUT_FILE_NAME:.a=debug.a)

.PHONY: all clean debug format $(OUT_FILE_NAME) $(OUT_FILE_NAME_DEBUG)
.DEFAULT_GOAL = all

ifeq ($(OS),Windows_NT)
	TARGET_TRIPLE 	  = x86_64-pc-windows-gnu
	CARGO_FLAGS 	  = --release --target $(TARGET_TRIPLE)
	CARGO_FLAGS_DEBUG = --target $(TARGET_TRIPLE)
else
	TARGET_TRIPLE	  =
	CARGO_FLAGS  	  = --release
	CARGO_FLAGS_DEBUG =
endif

CARGO = cargo
RM = rm -f

all: $(OUT_FILE_NAME)

debug: $(OUT_FILE_NAME_DEBUG)

clean:
	$(RM) $(OUT_FILE_NAME) $(OUT_FILE_NAME_DEBUG)
	$(CARGO) clean

format:
	$(CARGO) fmt

$(OUT_FILE_NAME):
	$(CARGO) build $(CARGO_FLAGS)
	cp ../target/$(TARGET_TRIPLE)/release/$(OUT_FILE_NAME) $(OUT_FILE_NAME)

$(OUT_FILE_NAME_DEBUG):
	$(CARGO) build $(CARGO_FLAGS_DEBUG)
	cp ../target/$(TARGET_TRIPLE)/debug/$(OUT_FILE_NAME) $(OUT_FILE_NAME_DEBUG)
