#!/bin/bash -xe

objectfiles=""
outdir=$1

if [[ -z "$outdir" ]]; then
	outdir="./"
fi

for FILE in source/*.c; do
	[ -f "$FILE" ] || break
	filename=$(basename ${FILE%%.*})
	gcc -I./include/ $FILE -c -o "$outdir/$filename".o
	objectfiles=$objectfiles" ""$outdir/$filename".o
done

if [[ "$OSTYPE" = "linux-gnu"* ]]; then
	ar cr "$outdir""/ddpstdlib.a" $objectfiles
elif [[ "$OSTYPE" = "win32" || "$OSTYPE" = "msys" ]]; then
	ar cr "$outdir""/ddpstdlib.lib" $objectfiles
else 
	echo unknown os
fi

for FILE in "$outdir"/*.o; do
	rm $FILE
done