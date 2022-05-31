#!/bin/bash -xe

if [[ "$OSTYPE" = "linux-gnu"* ]]; then
    export CGO_CPPFLAGS="`llvm-config-12 --cppflags`"
    export CGO_CXXFLAGS=-std=c++14
    export CGO_LDFLAGS="`llvm-config-12 --ldflags --libs --system-libs all`"
elif [[ "$OSTYPE" = "win32" || "$OSTYPE" = "msys" ]]; then
    export CGO_CPPFLAGS="`llvm-config --cppflags`"
    export CGO_CXXFLAGS=-std=c++14
    export CGO_LDFLAGS="`llvm-config --ldflags --libs --system-libs all`"
else 
	echo unknown os
fi

code .