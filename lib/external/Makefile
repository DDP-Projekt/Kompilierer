.PHONY: all clean checkout-submodules clean-pcre2 clean-libarchive clean-xz cleanlibz clean-bz2 clean-lz4
.DEFAULT_GOAL = all

RM = rm -f
MKDIR = mkdir -p

CC = gcc

CMAKE=cmake
CMAKE_GENERATOR="MinGW Makefiles"
CMAKE_OPTIONS=-DCMAKE_BUILD_TYPE=Release -G$(CMAKE_GENERATOR) -DCMAKE_C_COMPILER=$(CC) -DCMAKE_COLOR_MAKEFILE=OFF

PCRE2_OUT_FILE_NAME = libpcre2-8.a
PCRE2_DIR = ./pcre2_build/
ZLIB_DIR = ./zlib_build/
XZ_DIR = ./xz_build/

LIBZ_BUILD_NAME = libz.a
LIBZ_NAME = libz.a
XZ_NAME = liblzma.a
BZ2_NAME = libbz2.a
LZ4_NAME = liblz4.a

LIBAR_OUT_FILE_NAME = libarchive.a
LIBAR_DIR = ./libarchive_build/
LIBAR_CMAKE_OPTIONS = -DCMAKE_COLOR_MAKEFILE=OFF -DBUILD_SHARED_LIBS=OFF -DENABLE_TEST=OFF -DENABLE_INSTALL=OFF \
	-DENABLE_LIBXML2=OFF -DENABLE_OPENSSL=OFF -DENABLE_CNG=OFF -DENABLE_ZSTD=OFF -DHAVE_LIBXML_XMLREADER_H=0 -DHAVE_LIBXML_XMLWRITER_H=0 \

LIBAR_DEP_INCLUDES = -DZLIB_LIBRARY=./$(LIBZ_NAME) -DZLIB_INCLUDE_DIR=$(ZLIB_DIR)include \
	-DLIBLZMA_LIBRARY=./$(XZ_NAME) -DLIBLZMA_INCLUDE_DIR=$(XZ_DIR)include -DLIBLZMA_HAS_AUTO_DECODER=TRUE -DLIBLZMA_HAS_EASY_ENCODER=TRUE -DLIBLZMA_HAS_LZMA_PRESET=TRUE \
	-DLZ4_LIBRARY=./$(LZ4_NAME) -DLZ4_INCLUDE_DIR=./lz4/lib/ \
	-DBZIP2_LIBRARIES=./$(BZ2_NAME) -DBZIP2_INCLUDE_DIR=./bzip2/

ifeq ($(OS),Windows_NT)
	LIBZ_BUILD_NAME=libzlibstatic.a
else
	CMAKE_GENERATOR="Unix Makefiles"
endif

SUBMODULES = pcre2 libarchive zlib xz bzip2 lz4

checkout-submodules:
	git submodule update --init $(SUBMODULES)

# pcre2 only needs to be compiled on windows
$(PCRE2_OUT_FILE_NAME): | checkout-submodules
	@echo "building pcre2"
	$(MKDIR) $(PCRE2_DIR)
	'$(CMAKE)' -S./pcre2/ -B$(PCRE2_DIR) $(CMAKE_OPTIONS)
	'$(MAKE)' -C $(PCRE2_DIR) pcre2-8-static
	cp $(PCRE2_DIR)$(PCRE2_OUT_FILE_NAME) .

$(LIBZ_NAME): | checkout-submodules
	@echo "building zlib"
	'$(CMAKE)' -S./zlib/ -B$(ZLIB_DIR) $(CMAKE_OPTIONS)
	'$(MAKE)' -C $(ZLIB_DIR) zlibstatic
	mkdir -p $(ZLIB_DIR)include
	cp ./zlib/*.h $(ZLIB_DIR)include/
	cp $(ZLIB_DIR)zconf.h $(ZLIB_DIR)include/
	cp $(ZLIB_DIR)$(LIBZ_BUILD_NAME) $(LIBZ_NAME)

$(XZ_NAME): | checkout-submodules
	@echo "building lzma"
	'$(CMAKE)' -S./xz/ -B$(XZ_DIR) $(CMAKE_OPTIONS) -DBUILD_SHARED_LIBS=OFF
	'$(MAKE)' -C $(XZ_DIR) liblzma
	mkdir -p $(XZ_DIR)include/
	cp -r ./xz/src/liblzma/api/* $(XZ_DIR)include/
	cp $(XZ_DIR)$(XZ_NAME) .

$(BZ2_NAME): | checkout-submodules
	@echo "building bzip2"
	'$(MAKE)' -C ./bzip2/
	cp ./bzip2/$(BZ2_NAME) .

$(LZ4_NAME): | checkout-submodules
	@echo "building lz4"
	'$(MAKE)' -C ./lz4/ 'CC=$(CC)' liblz4.a
	cp ./lz4/lib/$(LZ4_NAME) .

$(LIBAR_OUT_FILE_NAME): $(LIBZ_NAME) $(XZ_NAME) $(BZ2_NAME) $(LZ4_NAME) $(PCRE2_OUT_FILE_NAME)
	@echo "building libarchive"
	'$(CMAKE)' -S./libarchive/ -B$(LIBAR_DIR) $(CMAKE_OPTIONS) $(LIBAR_CMAKE_OPTIONS) $(LIBAR_DEP_INCLUDES)
	'$(MAKE)' -C $(LIBAR_DIR) archive_static
	cp $(LIBAR_DIR)libarchive/$(LIBAR_OUT_FILE_NAME) .

all: checkout-submodules $(LIBAR_OUT_FILE_NAME)

clean-pcre2:
	'$(MAKE)' -C $(PCRE2_DIR) clean
clean-libarchive:
	'$(MAKE)' -C ./libarchive_build/ clean
clean-xz:
	'$(MAKE)' -C $(XZ_DIR) clean
clean-libz:
	'$(MAKE)' -C $(ZLIB_DIR) clean
clean-bz2:
	'$(MAKE)' -C ./bzip2/ clean
clean-lz4:
	'$(MAKE)' -C ./lz4/ clean

clean: clean-pcre2 clean-libarchive clean-xz cleanlibz clean-bz2 clean-lz4
	$(RM) $(PCRE2_OUT_FILE_NAME) $(LIBAR_OUT_FILE_NAME) $(LIBZ_NAME) $(XZ_NAME) $(BZ2_NAME) $(LZ4_NAME)