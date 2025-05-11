#!/bin/bash -xe

llvm_config="llvm-config"
if [[ -d "llvm_build" ]]; then
    llvm_config="llvm_build/bin/llvm-config"
elif [[ "$OSTYPE" = "linux-gnu"* ]]; then
    llvm_config="llvm-config-20"
fi

export CGO_CPPFLAGS="`$llvm_config --cppflags`"
export CGO_CXXFLAGS=-std=c++14
export CGO_LDFLAGS="`$llvm_config --ldflags --libs --system-libs all`"

"$1" .