OUT_FILE_NAME = libddpruntime.a
OUT_FILE_NAME_DEBUG = $(OUT_FILE_NAME:.a=debug.a)

.PHONY = all clean debug
.DEFAULT_GOAL = all

CC = gcc
CCFLAGS = -c -Wall -Wextra -Wno-format -O2 -std=c11 -pedantic
DEBUGFLAGS = -DDDP_DEBUG
INC = -I./include/

AR = ar rcs
RM = rm -f

HEADERS := $(shell find include -name '*.h') 
SRCS := $(shell find source -name '*.c')
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