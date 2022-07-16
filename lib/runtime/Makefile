OUT_FILE_NAME = ""
ifeq ($(OS),Windows_NT)
	OUT_FILE_NAME := ddpruntime.lib
else
	OUT_FILE_NAME := ddpruntime.a
endif

.PHONY = all build clean debug
.DEFAULT_GOAL := all

CC = gcc
DEBUGFLAGS := -DDDP_DEBUG
INC := -I./include/

SRCS := $(shell find source -name '*.c')
OBJS := $(notdir $(SRCS))
OBJS := $(OBJS:%.c=%.o)


all: build clean

build:
	$(CC) -c $(INC) $(SRCS)
	ar cr $(OUT_FILE_NAME) $(OBJS)

clean:
	rm -rf $(OBJS)

debug:
	$(CC) -c $(INC) $(DEBUGFLAGS) $(SRCS)
	ar cr $(OUT_FILE_NAME) $(OBJS)
	rm -rf $(OBJS)