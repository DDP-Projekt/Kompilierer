#include <locale.h>
#include <stdio.h>
#include <stdlib.h>
#include <signal.h>
#include "ddptypes.h"

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

	initTable(get_ref_table());
}

extern int inbuilt_ddpmain(); // implicitly defined by the ddp code

int main() {
	init(); // initialize
	int ret = inbuilt_ddpmain(); // run the ddp code
	freeTable(get_ref_table());
	return ret;
}