#!/bin/bash -xe

./build.sh
cd ../../lib/ddpstdlib
./build.sh ../../cmd/ddp/build