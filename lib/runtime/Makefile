OUT_FILE_NAME = libddpruntime.a
OUT_FILE_NAME_DEBUG = $(OUT_FILE_NAME:.a=debug.a)

.PHONY: all clean debug format $(OUT_FILE_NAME) $(OUT_FILE_NAME_DEBUG)
.DEFAULT_GOAL = all

CC = gcc
CCFLAGS = -c -Wall -Werror -Wextra -Wno-format -O2 -std=c11 -pedantic -D_POSIX_C_SOURCE=200809L
DEBUGFLAGS = -DDDP_DEBUG
INC = -I./include/


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

CLANG_FORMAT_FILE= ../../.clang-format
CLANG_FORMAT= clang-format -i -style='file:$(CLANG_FORMAT_FILE)'

# these wildcards work only to depth 1
# so include/utf8/utf8.c is matched but include/utf8/test_dir/test.c would NOT be matched
# if you want to add more subdirectories, simply add a nesting more here (aka: include/*/*/*.h)
HEADERS = $(wildcard include/DDP/*.h include/DDP/*/*.h)

%.o: %.c
	$(CC) $(CCFLAGS) $(INC) -o $@ $< 

%.o: %.c
	$(CC) $(CCFLAGS) $(DEBUGFLAGS) $(INC) -o $@ $<

all: $(OUT_FILE_NAME) source/main.o compile_commands.json

debug: $(OUT_FILE_NAME_DEBUG) source/main_debug.o

compile_commands.json: $(SRCS) $(HEADERS)
	@if [ ! -f bear ]; then \
		'$(MAKE)' $(OUT_FILE_NAME) --always-make --dry-run \
		| grep -w '$(CC) -c' \
		| go run ../create_compile_commands.go > compile_commands.json || true; \
	else \
		'$(MAKE)' clean && bear -- '$(MAKE)' all || true; \
	fi

clean:
	$(RM) $(OUT_FILE_NAME) $(OUT_FILE_NAME_DEBUG) main.o main_debug.o .depend
	$(CARGO) clean

format:
	$(CLANG_FORMAT) $(HEADERS)
	$(CARGO) fmt

$(OUT_FILE_NAME):
	$(CARGO) build $(CARGO_FLAGS)
	cp ../target/$(TARGET_TRIPLE)/release/$(OUT_FILE_NAME) $(OUT_FILE_NAME)

$(OUT_FILE_NAME_DEBUG):
	$(CARGO) build $(CARGO_FLAGS_DEBUG)
	cp ../target/$(TARGET_TRIPLE)/debug/$(OUT_FILE_NAME) $(OUT_FILE_NAME_DEBUG)
