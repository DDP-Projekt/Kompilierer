#include "runtime.h"

extern int ddp_ddpmain(); // implicitly defined by the ddp code

// entry point of the final executable (needed by gcc)
int main(int argc, char** argv) {
	ddp_init_runtime(argc, argv); // initialize the runtime
	int ret = ddp_ddpmain(); // run the ddp code
	ddp_end_runtime(); // end the runtime
	return ret; // return the exit status of the ddp-code
}