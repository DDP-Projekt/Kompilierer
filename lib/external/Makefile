.PHONY: all clean checkout-submodules
.DEFAULT_GOAL = all

RM = rm -f
MKDIR = mkdir -p

CC = gcc

CMAKE_BUILD_TOOL=$(MAKE)
CMAKE_GENERATOR="MinGW Makefiles"
CMAKE_OPTIONS=-DCMAKE_BUILD_TYPE=Release -G$(CMAKE_GENERATOR) -DCMAKE_C_COMPILER=$(CC) -DCMAKE_COLOR_MAKEFILE=OFF

PCRE2_OUT_FILE_NAME = libpcre2-8.a
PCRE2_DIR = ./pcre2_build/
ZLIB_DIR = ./zlib_build/
XZ_DIR = ./xz_build/

LIBZ_NAME = libz.a
XZ_NAME = liblzma.a
BZ2_NAME = libbz2.a
LZ4_NAME = liblz4.a

LIBAR_OUT_FILE_NAME = libarchive.a
LIBAR_DIR = ./libarchive_build/
LIBAR_CMAKE_OPTIONS = -DCMAKE_COLOR_MAKEFILE=OFF -DBUILD_SHARED_LIBS=OFF -DENABLE_TEST=OFF -DENABLE_INSTALL=OFF \
	-DENABLE_LIBXML2=OFF -DENABLE_OPENSSL=OFF -DENABLE_CNG=OFF -DHAVE_LIBXML_XMLREADER_H=0 -DHAVE_LIBXML_XMLWRITER_H=0 \

LIBAR_DEP_INCLUDES = -DZLIB_LIBRARY=./$(LIBZ_NAME) -DZLIB_INCLUDE_DIR=$(ZLIB_DIR)include \
	-DLIBLZMA_LIBRARY=./$(XZ_NAME) -DLIBLZMA_INCLUDE_DIR=$(XZ_DIR)include \
	-DLZ4_LIBRARY=./$(LZ4_NAME) -DLZ4_INCLUDE_DIR=./lz4/lib/ \
	-DBZIP2_LIBRARIES=./$(BZ2_NAME) -DBZIP2_INCLUDE_DIR=./bzip2/

ifeq ($(OS),Windows_NT)
# windows
else
	CMAKE_GENERATOR="Unix Makefiles"
endif

SUBMODULES = pcre2 libarchive zlib xz bzip2 lz4

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
	cmake -S./pcre2/ -B$(PCRE2_DIR) $(CMAKE_OPTIONS)
	cd $(PCRE2_DIR) && $(CMAKE_BUILD_TOOL) pcre2-8-static
	cp $(PCRE2_DIR)$(PCRE2_OUT_FILE_NAME) .
endif

$(LIBZ_NAME):
	@echo "building zlib"
	cmake -S./zlib/ -B$(ZLIB_DIR) $(CMAKE_OPTIONS)
	cd $(ZLIB_DIR) ; make zlibstatic
	mkdir -p $(ZLIB_DIR)include
	cp ./zlib/*.h $(ZLIB_DIR)include/
	mv $(ZLIB_DIR)zconf.h $(ZLIB_DIR)include/
	cp $(ZLIB_DIR)libzlibstatic.a $(LIBZ_NAME)

$(XZ_NAME):
	@echo "building lzma"
	cmake -S./xz/ -B$(XZ_DIR) $(CMAKE_OPTIONS) -DBUILD_SHARED_LIBS=OFF
	cd $(XZ_DIR) ; make liblzma
	mkdir -p $(XZ_DIR)include
	cp -r ./xz/src/liblzma/api/* $(XZ_DIR)include/
	cp $(XZ_DIR)$(XZ_NAME) .

$(BZ2_NAME):
	@echo "building bzip2"
	cd ./bzip2/ ; make bzip2
	cp ./bzip2/$(BZ2_NAME) .

$(LZ4_NAME):
	@echo "building lz4"
	cd ./lz4/ ; make CC=$(CC) liblz4.a
	cp ./lz4/lib/$(LZ4_NAME) .

$(LIBAR_OUT_FILE_NAME): | checkout-submodules $(LIBZ_NAME) $(XZ_NAME) $(BZ2_NAME) $(LZ4_NAME)
	@echo "building libarchive"
	cmake -S./libarchive/ -B$(LIBAR_DIR) $(CMAKE_OPTIONS) $(LIBAR_CMAKE_OPTIONS) $(LIBAR_DEP_INCLUDES)
	cd $(LIBAR_DIR) ; make archive_static
	cp $(LIBAR_DIR)/libarchive/$(LIBAR_OUT_FILE_NAME) .

all: checkout-submodules $(PCRE2_OUT_FILE_NAME) $(LIBAR_OUT_FILE_NAME)

clean:
ifeq ($(OS),Windows_NT)
	$(RM) $(PCRE2_OUT_FILE_NAME)
	cd $(PCRE2_DIR) ; $(CMAKE_BUILD_TOOL) clean
endif
	$(RM) $(LIBAR_OUT_FILE_NAME)
	$(RM) $(XZ_NAME)
	$(RM) $(BZ2_NAME)
	$(RM) $(LZ4_NAME)