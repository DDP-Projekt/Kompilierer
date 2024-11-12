.PHONY = all clean checkout-submodules
.DEFAULT_GOAL = all

RM = rm -f
MKDIR = mkdir -p

CC = gcc

CMAKE_BUILD_TOOL=$(MAKE)
CMAKE_GENERATOR="MinGW Makefiles"

PCRE2_OUT_FILE_NAME = libpcre2-8.a
PCRE2_DIR = ./pcre2_build/

LIBAR_OUT_FILE_NAME = libarchive.a
LIBAR_DIR = ./libarchive_build/

ifeq ($(OS),Windows_NT)
# windows
else
	CMAKE_GENERATOR="Unix Makefiles"
endif

SUBMODULES = pcre2 libarchive

checkout-submodules:
	git submodule update --init $(SUBMODULES)

# pcre2 only needs to be compiled on windows
ifneq ($(OS),Windows_NT)
$(PCRE2_OUT_FILE_NAME):
	@echo "pcre2 is already provided on linux systems"
else
$(PCRE2_OUT_FILE_NAME): | checkout-submodules
	@echo "building pcre2"
	$(MKDIR) $(PCRE2_DIR)
	cmake -S./pcre2/ -B$(PCRE2_DIR) -DCMAKE_BUILD_TYPE=Release -G$(CMAKE_GENERATOR) -DCMAKE_C_COMPILER=$(CC) -DCMAKE_COLOR_MAKEFILE=OFF
	cd $(PCRE2_DIR) && $(CMAKE_BUILD_TOOL) pcre2-8-static
	cp $(PCRE2_DIR)$(LIBAR_OUT_FILE_NAME) .
endif

$(LIBAR_OUT_FILE_NAME): | checkout-submodules
	@echo "building libarchive"
	cmake -S./libarchive/ -B$(LIBAR_DIR) -DCMAKE_BUILD_TYPE=Release -G$(CMAKE_GENERATOR) -DCMAKE_C_COMPILER=$(CC)
	cd $(LIBAR_DIR) ; make
	cp $(LIBAR_DIR)/libarchive/$(LIBAR_OUT_FILE_NAME) .

all: checkout-submodules $(PCRE2_OUT_FILE_NAME) $(LIBAR_OUT_FILE_NAME)

clean:
ifeq ($(OS),Windows_NT)
	$(RM) $(PCRE2_OUT_FILE_NAME)
	cd $(PCRE2_DIR) ; $(CMAKE_BUILD_TOOL) clean
endif
