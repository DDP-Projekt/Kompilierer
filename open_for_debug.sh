#!/bin/sh -xe

export CGO_CPPFLAGS="`D:/Hendrik/source/llvm/llvm-project-llvmort-12.0.0-build-mingw/bin/llvm-config.exe --cppflags`"
export CGO_CXXFLAGS=-std=c++14
export CGO_LDFLAGS="`D:/Hendrik/source/llvm/llvm-project-llvmort-12.0.0-build-mingw/bin/llvm-config.exe --ldflags --libs --system-libs all`"
code .