#!/bin/sh -xe

objectfiles=""

for FILE in *.c; do
	[ -f "$FILE" ] || break
	gcc $FILE -c -o "${FILE%%.*}".o
	objectfiles=$objectfiles" ""${FILE%%.*}".o
done

ar cr ../ddpstdlib.lib $objectfiles

for FILE in *.o; do
	rm $FILE
done