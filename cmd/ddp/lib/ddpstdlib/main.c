#include <locale.h>
#include <stdio.h>
#include <stdlib.h>
#include <signal.h>
#include "gc.h"

static void SegfaultHandler(int signal) {
	printf("Segmentation fault");
	exit(1);
}

// initializes stuff
static void init() {
	setlocale(LC_ALL, "en_US.UTF-8");
	//setlocale(LC_NUMERIC, "de_DE.utf8");
	setlocale(LC_NUMERIC, "French_Canada.1252"); // somehow the above did not work

	signal(SIGSEGV, SegfaultHandler);
}

extern int inbuilt_ddpmain(); // implicitly defined by the ddp code

int main() {
	init(); // initialize
	int ret = inbuilt_ddpmain(); // run the ddp code
	GC(); // collect garbage
	return ret;
}