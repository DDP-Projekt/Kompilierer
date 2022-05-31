OUT_FILE_NAME = ""
ifeq ($(OS),Windows_NT)
	OUT_FILE_NAME := ddpstdlib.lib
else
	OUT_FILE_NAME := ddpstdlib.a
endif

.PHONY = all build clean debug

CC = gcc
DEBUGFLAGS := -D DEBUG 
INC := -I./include/ # set on the debug phony

SRCS := $(wildcard source/*.c)
OBJS := $(patsubst source/%.c, %.o, $(SRCS))

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