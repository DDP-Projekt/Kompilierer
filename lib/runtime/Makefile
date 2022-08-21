OUT_FILE_NAME = libddpruntime.a

.PHONY = all build clean debug
.DEFAULT_GOAL := all

CC = gcc
CCFLAGS := -c -Wall -Wno-format -O2
DEBUGFLAGS := -DDDP_DEBUG
INC := -I./include/

SRCS := $(shell find source -name '*.c')
OBJS := $(notdir $(SRCS))
OBJS := $(OBJS:%.c=%.o)


all: build clean

build:
	$(CC) $(CCFLAGS) $(INC) $(SRCS)
	ar cr $(OUT_FILE_NAME) $(OBJS)

clean:
	rm -rf $(OBJS)

debug:
	$(CC) $(CCFLAGS) $(INC) $(DEBUGFLAGS) $(SRCS)
	ar cr $(OUT_FILE_NAME) $(OBJS)
	rm -rf $(OBJS)