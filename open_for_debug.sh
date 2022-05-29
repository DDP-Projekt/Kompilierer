#!/bin/bash -xe

if [[ "$OSTYPE" = "linux-gnu"* ]]; then
    export CGO_CPPFLAGS="`llvm-config-12 --cppflags`"
    export CGO_CXXFLAGS=-std=c++14
    export CGO_LDFLAGS="`llvm-config-12 --ldflags --libs --system-libs all`"
elif [[ "$OSTYPE" = "win32" || "$OSTYPE" = "msys" ]]; then
    export CGO_CPPFLAGS="`D:/Hendrik/source/llvm/llvm-project-llvmort-12.0.0-build-mingw/bin/llvm-config.exe --cppflags`"
    export CGO_CXXFLAGS=-std=c++14
    export CGO_LDFLAGS="`D:/Hendrik/source/llvm/llvm-project-llvmort-12.0.0-build-mingw/bin/llvm-config.exe --ldflags --libs --system-libs all`"
else 
	echo unknown os
fi

code .