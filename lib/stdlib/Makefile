OUT_FILE_NAME = libddpstdlib.a
OUT_FILE_NAME_DEBUG = $(OUT_FILE_NAME:.a=debug.a)

.PHONY = all clean debug
.DEFAULT_GOAL = all

CC = gcc
CCFLAGS = -c -Wall -Werror -Wno-format -O2 -std=c11 -pedantic -D_POSIX_C_SOURCE=200809L
DEBUGFLAGS = -DDDP_DEBUG
INC = -I./include/ -I../runtime/include/

AR = ar rcs
RM = rm -f

# these wildcards work only to depth 1
# so include/utf8/utf8.c is matched but include/utf8/test_dir/test.c would NOT be matched
# if you want to add more subdirectories, simply add a nesting more here (aka: include/*/*/*.h)
HEADERS = $(wildcard ../runtime/include/*.h ../runtime/include/*/*.h)
HEADERS = $(wildcard include/*.h include/*/*.h)
SRCS = $(wildcard source/*.c source/*/*.c)
OBJS = $(SRCS:.c=.o)
OBJS_DEBUG = $(OBJS:.o=_debug.o)

%.o: %.c
	$(CC) $(CCFLAGS) $(INC) -o $@ $<

%_debug.o: %.c
	$(CC) $(CCFLAGS) $(DEBUGFLAGS) $(INC) -o $@ $<

all: $(OUT_FILE_NAME) compile_commands.json

debug: $(OUT_FILE_NAME_DEBUG)

$(OUT_FILE_NAME): $(OBJS)
	$(AR) $(OUT_FILE_NAME) $(OBJS)

$(OUT_FILE_NAME_DEBUG): $(OBJS_DEBUG)
	$(AR) $(OUT_FILE_NAME_DEBUG) $(OBJS_DEBUG)

compile_commands.json: $(SRCS) $(HEADERS)
	@if [ ! -f bear ]; then \
		$(MAKE) $(OUT_FILE_NAME) --always-make --dry-run \
		| grep -w '$(CC) -c' \
		| go run ../create_compile_commands.go > compile_commands.json; \
	else \
		$(MAKE) clean && bear -- $(MAKE) all; \
	fi

clean:
	$(RM) $(OBJS) $(OBJS_DEBUG) $(OUT_FILE_NAME) $(OUT_FILE_NAME_DEBUG) .depend compile_commands.json

.depend: $(SRCS) $(HEADERS)
	$(CC) -MM $(INC) $(SRCS) > .depend

include .depend
