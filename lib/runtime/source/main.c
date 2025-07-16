#include "DDP/common.h"
#include "DDP/debug.h"
#include "DDP/runtime.h"
#include <stdarg.h>
#include <stdio.h>
#include <stdlib.h>

extern int ddp_ddpmain(void); // implicitly defined by the ddp code

// entry point of the final executable (needed by gcc)
int main(int argc, char **argv) {
	ddp_init_runtime(argc, argv); // initialize the runtime
	int ret = ddp_ddpmain();	  // run the ddp code
	ddp_end_runtime();			  // end the runtime
	return ret;					  // return the exit status of the ddp-code
}

// print the error message to stderr and exit with exit_code
void ddp_runtime_error(int exit_code, const char *fmt, ...) {
	DDP_DBGLOG("runtime_error: %d, %s", exit_code, fmt);

	va_list argptr;
	va_start(argptr, fmt);

	fprintf(stderr, "\nLaufzeitfehler: ");
	vfprintf(stderr, fmt, argptr);

	va_end(argptr);

	ddp_end_runtime();
	exit(exit_code);
}
