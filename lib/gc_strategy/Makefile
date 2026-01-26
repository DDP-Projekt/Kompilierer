LLVM_CONFIG=""
LLVM_CFLAGS=""

ifeq ($(OS),Windows_NT)
	ifneq ("$(wildcard ../../llvm_build/bin/llvm-config.exe)","")
		LLVM_CONFIG := "../../llvm_build/bin/llvm-config.exe"
	else
		LLVM_CONFIG := "llvm-config"
	endif
	LLVM_CFLAGS = $(shell $(LLVM_CONFIG) --cppflags --cxxflags)
else
	ifneq ("$(wildcard ../../llvm_build/bin/llvm-config)","")
		LLVM_CONFIG := "../../llvm_build/bin/llvm-config"
	else
		LLVM_CONFIG := "llvm-config-20"
	endif
	LLVM_CFLAGS = $(shell $(LLVM_CONFIG) --cppflags --cxxflags)
endif

TARGET = libddp-gc-strategy.a

CXX = g++
CCFLAGS = -c -fPIC -O2 $(LLVM_CFLAGS)
SO_FLAGS = -shared

SRCS = $(wildcard *.cpp)
OBJS = $(SRCS:.cpp=.o)
HEADERS = $(wildcard *.h)

CLANG_FORMAT_FILE= ../../.clang-format
CLANG_FORMAT= clang-format -i -style='file:$(CLANG_FORMAT_FILE)'

AR = ar rcs
RM = rm -f

.PHONY: all clean

.DEFAULT_GOAL := all

all: $(TARGET) compile_commands.json

%.o: %.cpp
	$(CXX) $(CCFLAGS) -o $@ $<

$(TARGET): $(OBJS) $(HEADERS)
	$(AR) $@ $<

compile_commands.json: $(SRCS) $(HEADERS)
	@if [ ! -f bear ]; then \
		'$(MAKE)' $(TARGET) --always-make --dry-run \
		| grep -w '$(CXX) -c' \
		| go run ../create_compile_commands.go > compile_commands.json || true; \
	else \
		'$(MAKE)' clean && bear -- '$(MAKE)' all || true; \
	fi

clean:
	$(RM) $(TARGET) $(OBJS)

format:
	$(CLANG_FORMAT) $(SRCS) $(HEADERS)
