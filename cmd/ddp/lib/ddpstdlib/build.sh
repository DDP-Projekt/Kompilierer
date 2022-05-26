#!/bin/bash -xe

objectfiles=""

for FILE in *.c; do
	[ -f "$FILE" ] || break
	gcc $FILE -c -o "${FILE%%.*}".o
	objectfiles=$objectfiles" ""${FILE%%.*}".o
done

if [[ "$OSTYPE" = "linux-gnu"* ]]; then
	ar cr ../ddpstdlib.a $objectfiles
elif [[ "$OSTYPE" = "win32" || "$OSTYPE" = "msys" ]]; then
	ar cr ../ddpstdlib.lib $objectfiles
else 
	echo unknown os
fi

for FILE in *.o; do
	rm $FILE
done