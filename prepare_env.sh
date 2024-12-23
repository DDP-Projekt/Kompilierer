#!/bin/bash

[[ $_ != $0 ]] || (echo "usage: \$source prepare_env.sh" && exit 1);

llvm_config="llvm-config"
if [[ -d "llvm_build" ]]; then
    llvm_config="llvm_build/bin/llvm-config"
elif [[ "$OSTYPE" = "linux-gnu"* ]]; then
    llvm_config="llvm-config-14"
fi

export CGO_CPPFLAGS="`$llvm_config --cppflags`"
export CGO_CXXFLAGS=-std=c++14
export CGO_LDFLAGS="`$llvm_config --ldflags --libs --system-libs all`"

"$1" .