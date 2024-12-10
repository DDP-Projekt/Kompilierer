OUT_FILE_NAME = libddpruntime.a
OUT_FILE_NAME_DEBUG = $(OUT_FILE_NAME:.a=debug.a)

.PHONY: all clean debug
.DEFAULT_GOAL = all

CC = gcc
CCFLAGS = -c -Wall -Werror -Wextra -Wno-format -O2 -std=c11 -pedantic -D_POSIX_C_SOURCE=200809L
DEBUGFLAGS = -DDDP_DEBUG
INC = -I./include/

AR = ar rcs
RM = rm -f

# these wildcards work only to depth 1
# so include/utf8/utf8.c is matched but include/utf8/test_dir/test.c would NOT be matched
# if you want to add more subdirectories, simply add a nesting more here (aka: include/*/*/*.h)
HEADERS = $(wildcard include/DDP/*.h include/DDP/*/*.h)
SRCS = $(wildcard source/DDP/*.c source/DDP/*/*.c)
OBJS = $(SRCS:.c=.o)
OBJS_DEBUG = $(OBJS:.o=_debug.o)

LIB_OBJS = $(filter-out source/main.o, $(OBJS))
LIB_OBJS_DEBUG = $(filter-out source/main_debug.o, $(OBJS_DEBUG))

%.o: %.c
	$(CC) $(CCFLAGS) $(INC) -o $@ $< 

%_debug.o: %.c
	$(CC) $(CCFLAGS) $(DEBUGFLAGS) $(INC) -o $@ $<

all: $(OUT_FILE_NAME) source/main.o

debug: $(OUT_FILE_NAME_DEBUG) source/main_debug.o

$(OUT_FILE_NAME): $(LIB_OBJS)
	$(AR) $(OUT_FILE_NAME) $(LIB_OBJS)

$(OUT_FILE_NAME_DEBUG): $(LIB_OBJS_DEBUG)
	$(AR) $(OUT_FILE_NAME_DEBUG) $(LIB_OBJS_DEBUG)

compile_commands.json: $(SRCS) $(HEADERS)
	@if [ ! -f bear ]; then \
		'$(MAKE)' $(OUT_FILE_NAME) --always-make --dry-run \
		| grep -w '$(CC) -c' \
		| go run ../create_compile_commands.go > compile_commands.json || true; \
	else \
		'$(MAKE)' clean && bear -- '$(MAKE)' all || true; \
	fi

clean:
	$(RM) $(OBJS) $(OBJS_DEBUG) $(OUT_FILE_NAME) $(OUT_FILE_NAME_DEBUG) .depend

.depend: $(SRCS) $(HEADERS)
	$(CC) -MM $(INC) $(SRCS) > .depend

include .depend
