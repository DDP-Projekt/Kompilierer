OUT_FILE_NAME = libddpstdlib.a

.PHONY = all build clean debug
.DEFAULT_GOAL := all

CC = gcc
CCFLAGS := -c -Wall -Wno-format -pedantic -O2
DEBUGFLAGS := -DDDP_DEBUG
INC := -I./include/ -I../runtime/include/

AR = ar

SRCS := $(shell find source -name '*.c')
OBJS := $(notdir $(SRCS))
OBJS := $(OBJS:%.c=%.o)


all: build clean

build:
	$(CC) $(CCFLAGS) $(INC) $(SRCS)
	$(AR) cr $(OUT_FILE_NAME) $(OBJS)

clean:
	rm -rf $(OBJS)

debug:
	$(CC) $(CCFLAGS) $(INC) $(DEBUGFLAGS) $(SRCS)
	$(AR) cr $(OUT_FILE_NAME) $(OBJS)
	rm -rf $(OBJS)