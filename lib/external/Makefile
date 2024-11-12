.PHONY = all clean checkout-submodules
.DEFAULT_GOAL = all

RM = rm -f
MKDIR = mkdir -p

PCRE2_CMAKE_BUILD_TOOL=$(MAKE)
PCRE2_OUT_FILE_NAME = libpcre2-8.a
PCRE2_DIR = ./pcre2_build/

SUBMODULES = pcre2

checkout-submodules:
	git submodule update --init $(SUBMODULES)

# pcre2 only needs to be compiled on windows
ifneq ($(OS),Windows_NT)
$(PCRE2_OUT_FILE_NAME):
	@echo "pcre2 is already provided on linux systems"
else
$(PCRE2_OUT_FILE_NAME): | checkout-pcre2
	@echo "building pcre2"
	$(MKDIR) $(PCRE2_DIR)
	cmake -S./pcre2/ -B$(PCRE2_DIR) -DCMAKE_BUILD_TYPE=Release -G"MinGW Makefiles" -DCMAKE_C_COMPILER=$(CC) -DCMAKE_COLOR_MAKEFILE=OFF
	cd $(PCRE2_DIR) && $(PCRE2_CMAKE_BUILD_TOOL) pcre2-8-static
	cp $(PCRE2_DIR)$(PCRE2_OUT_FILE_NAME) .
endif

all: checkout-submodules $(PCRE2_OUT_FILE_NAME)

clean:
ifeq ($(OS),Windows_NT)
	$(RM) $(PCRE2_OUT_FILE_NAME)
	cd $(PCRE2_DIR) ; $(PCRE2_CMAKE_BUILD_TOOL) clean
endif
