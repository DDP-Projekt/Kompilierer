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

LIBAR_OUT_FILE_NAME = libarchive.a
LIBAR_DIR = ./libarchive_build/
LIBAR_ADDITIONAL_CMAKE_OPTIONS = -DCMAKE_COLOR_MAKEFILE=OFF -DBUILD_SHARED_LIBS=OFF -DENABLE_TEST=OFF -DENABLE_INSTALL=OFF \
	-DENABLE_LIBXML2=OFF -DENABLE_OPENSSL=OFF -DENABLE_CNG=OFF -DHAVE_LIBXML_XMLREADER_H=0 -DHAVE_LIBXML_XMLWRITER_H=0 \
	-DZLIB_LIBRARY=./libz.a -DZLIB_INCLUDE_DIR=$(ZLIB_DIR) -DLIBLZMA_LIBRARY=./liblzma.a -DLIBLZMA_INCLUDE_DIR=./xz/lib/ -DPCRE2_LIBRARY=./libpcre2-8.a

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

libz.a:
	@echo "building zlib"
	cmake -S./zlib/ -B$(ZLIB_DIR) $(CMAKE_OPTIONS)
	cd $(ZLIB_DIR) ; make zlibstatic
	cp ./zlib/*.h $(ZLIB_DIR)
	cp $(ZLIB_DIR)libzlibstatic.a ./libz.a

liblzma.a:
	@echo "building lzma"
	cmake -S./xz/ -B$(XZ_DIR) $(CMAKE_OPTIONS)
	cd $(XZ_DIR) ; make liblzma
	cp $(XZ_DIR)liblzma.a .

libbz2.a:
	@echo "building bzip2"
	cd ./bzip2/ ; make bzip2
	cp ./bzip2/libbz2.a .

liblz4.a:
	@echo "building lz4"
	cd ./lz4/ ; make CC=$(CC) liblz4.a
	cp ./lz4/lib/liblz4.a .

$(LIBAR_OUT_FILE_NAME): | checkout-submodules libz.a liblzma.a libbz2.a liblz4.a
	@echo "building libarchive"
	cmake -S./libarchive/ -B$(LIBAR_DIR) $(CMAKE_OPTIONS) $(LIBAR_ADDITIONAL_CMAKE_OPTIONS)
	cd $(LIBAR_DIR) ; make archive_static
	cp $(LIBAR_DIR)/libarchive/$(LIBAR_OUT_FILE_NAME) .

all: checkout-submodules $(PCRE2_OUT_FILE_NAME) $(LIBAR_OUT_FILE_NAME)

clean:
ifeq ($(OS),Windows_NT)
	$(RM) $(PCRE2_OUT_FILE_NAME)
	cd $(PCRE2_DIR) ; $(CMAKE_BUILD_TOOL) clean
endif
	$(RM) $(LIBAR_OUT_FILE_NAME)
