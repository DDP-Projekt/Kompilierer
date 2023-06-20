OUT_FILE_NAME = libddpstdlib.a
OUT_FILE_NAME_DEBUG = $(OUT_FILE_NAME:.a=debug.a)

.PHONY = all clean debug
.DEFAULT_GOAL = all

CC = gcc
CCFLAGS = -c -Wall -Wno-format -O2 -std=c11 -pedantic
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
	$(CC) $(CCFLAGS) $(INC) $< -o $@

%_debug.o: %.c
	$(CC) $(CCFLAGS) $(DEBUGFLAGS) $(INC)  $< -o $@

all: $(OUT_FILE_NAME)

debug: $(OUT_FILE_NAME_DEBUG)

$(OUT_FILE_NAME): $(OBJS)
	$(AR) $(OUT_FILE_NAME) $(OBJS)

$(OUT_FILE_NAME_DEBUG): $(OBJS_DEBUG)
	$(AR) $(OUT_FILE_NAME_DEBUG) $(OBJS_DEBUG)

clean:
	$(RM) $(OBJS) $(OBJS_DEBUG) $(OUT_FILE_NAME) $(OUT_FILE_NAME_DEBUG) .depend

.depend: $(SRCS) $(HEADERS)
	$(CC) -MM $(INC) $(SRCS) > .depend

include .depend