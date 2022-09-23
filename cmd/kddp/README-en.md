# kddp
This executeable contains commands to parse and build .ddp files.
## Commands
|Command name|Command syntax|Command description|Command options|Option description|
|------------|--------------|-------------------|---------------|------------------|
|help|`help <command>`|displays usage information|-|-|
|build|`build <filename> <options>`|build the given .ddp file into a executable|`-o <filepath>`<br>`--verbose`<br>`--nodeletes`<br>`--gcc_flags`<br>`--extern_gcc_flags`|specify the name of the output file<br>print verbose output<br>don't delete intermediate files<br>custom flags that are passed to gcc<br>custom flags that are passed to gcc when compiling extern .c files|
|parse|`parse <filepath> <options>`|parse the specified ddp file into a ddp ast|`-o <filepath>`|specify the name of the output file; if none is set output is written to the terminal|
|version|`version <options>`|display version information for kddp|`--verbose`<br>`--build_info`|show verbose output for all versions<br>show go build info|
|run|`run <filename> <options>`|compile and run the given .ddp file|`--verbose`<br>`--gcc_flags`<br>`--extern_gcc_flags`|print verbose output<br>custom flags that are passed to gcc<br>custom flags that are passed to gcc when compiling extern .c files|