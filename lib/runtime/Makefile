OUT_FILE_NAME = libddpruntime.a

.PHONY = all build clean debug
.DEFAULT_GOAL = all

CC = gcc
CCFLAGS = -c -Wall -Wno-format -O2
DEBUGFLAGS = -DDDP_DEBUG
INC = -I./include/

AR = ar

SRCS = $(shell find source -name '*.c')
OBJS := $(notdir $(SRCS))
OBJS := $(OBJS:%.c=%.o)


all: build clean-objs

build:
	$(CC) $(CCFLAGS) $(INC) $(SRCS)
	$(AR) cr $(OUT_FILE_NAME) $(OBJS)

clean-objs:
	rm -rf $(OBJS)

clean:
	rm -rf $(OBJS) $(OUT_FILE_NAME)

debug:
	$(CC) $(CCFLAGS) $(INC) $(DEBUGFLAGS) $(SRCS)
	$(AR) cr $(OUT_FILE_NAME) $(OBJS)
	rm -rf $(OBJS)