#!/bin/bash

set -euxo pipefail

if [ "$OSTYPE" = "win32" ] || [ "$OSTYPE" = "msys" ] || [ "$OSTYPE" = "cygwin" ]; then
	is_windows=true
else
	is_windows=false
fi

fail_usage() {
	if [ "$is_windows" = true ]; then
		echo "Usage: release.sh <path to DDPLS> <optional path to MinGW zip>"
	else
		echo "Usage: release.sh <path to DDPLS>"
	fi
	exit 1
}

# ======= Safe Guards =======

if [ "$(basename $(pwd))" != "Kompilierer" ]; then
	echo "This script must be run from the root of the Kompilierer repo"
	fail_usage
fi

if [ -z "$1" ] || [ ! -f "$1" ]; then
	echo "Path to DDPLS is not a file"
	fail_usage
fi

if [ $# -ge 2 ] && [ ! -f "$2" ]; then
	echo "Path to MinGW is not a file"
	fail_usage
fi

# ===========================

make -j$(nproc)

ship_mingw=true
if [ "$is_windows" = true ] && [ $# -eq 1 ]; then
	ship_mingw=false
fi

release_folder_path="DDP-"
release_folder_path+=$(eval ./build/DDP/bin/kddp version | head -n 1 | tr " " "-")
if [ "$ship_mingw" != true ]; then
	release_folder_path+="-no-mingw"
fi

echo "Output folder: $release_folder_path"

rm -rf $release_folder_path
cp -rf ./build/DDP/ "$release_folder_path"

ls_dest="$release_folder_path/bin/DDPLS"
if [ "$is_windows" = true ]; then
	ls_dest+=".exe"
fi
cp -rf "$1" "$ls_dest"

if [ "$ship_mingw" = true ]; then
	cp -f "$2" "$release_folder_path/mingw64.zip"
fi

zip -r "$release_folder_path.zip" "$release_folder_path"
