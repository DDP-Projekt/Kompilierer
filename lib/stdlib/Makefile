OUT_FILE_NAME = libddpstdlib.a
OUT_FILE_NAME_DEBUG = $(OUT_FILE_NAME:.a=debug.a)
PCRE2_OUT_FILE_NAME = libpcre2-8.a

.PHONY = all clean debug
.DEFAULT_GOAL = all

CC = gcc
CCFLAGS = -c -Wall -Werror -Wno-format -O2 -std=c11 -pedantic -D_POSIX_C_SOURCE=200809L
DEBUGFLAGS = -DDDP_DEBUG
INC = -I./include/ -I../runtime/include/ -I./pcre2/src/

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

# pcre2 only needs to be compiled on windows
ifneq ($(OS),Windows_NT)
	undefine PCRE2_OUT_FILE_NAME
else
$(PCRE2_OUT_FILE_NAME):
	cd ./pcre2 ; $(MAKE)
	cp ./pcre2/.libs/$(PCRE2_OUT_FILE_NAME) .
endif

%.o: %.c
	$(CC) $(CCFLAGS) $(INC) $< -o $@

%_debug.o: %.c
	$(CC) $(CCFLAGS) $(DEBUGFLAGS) $(INC)  $< -o $@

all: $(OUT_FILE_NAME) $(PCRE2_OUT_FILE_NAME)

debug: $(OUT_FILE_NAME_DEBUG) $(PCRE2_OUT_FILE_NAME)

$(OUT_FILE_NAME): $(OBJS)
	$(AR) $(OUT_FILE_NAME) $(OBJS)

$(OUT_FILE_NAME_DEBUG): $(OBJS_DEBUG)
	$(AR) $(OUT_FILE_NAME_DEBUG) $(OBJS_DEBUG)

clean:
	$(RM) $(OBJS) $(OBJS_DEBUG) $(OUT_FILE_NAME) $(PCRE2_OUT_FILE_NAME) $(OUT_FILE_NAME_DEBUG) .depend
	@if [ -d ./pcre2 ]; then \
		cd ./pcre2 && $(MAKE) clean; \
	fi

.depend: $(SRCS) $(HEADERS)
	$(CC) -MM $(INC) $(SRCS) > .depend

include .depend
