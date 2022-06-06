OUT_FILE_NAME = ""
ifeq ($(OS),Windows_NT)
	OUT_FILE_NAME := ddpstdlib.lib
else
	OUT_FILE_NAME := ddpstdlib.a
endif

.PHONY = all build clean debug var

CC = gcc
DEBUGFLAGS := -D DEBUG
INC := -I./include/ # set on the debug phony

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
	$(CC) -c $(INC) $(DEBUGFLAGS) $(CCFLAGS) $(SRCS)
	ar cr $(OUT_FILE_NAME) $(OBJS)
	rm -rf $(OBJS)

var:
	@echo $(SRCS)
	@echo $(OBJS)