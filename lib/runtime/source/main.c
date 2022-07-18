/*
	initialization for the ddp-c-runtime
	also defines the entry point of the executable
*/
#include <locale.h>
#include <signal.h>
#ifdef _WIN32
#include <Windows.h>
#endif // _WIN32
#include "gc.h"
#include "mt19937-64.h"

// should not be needed in production
// mainly for debugging
static void SegfaultHandler(int signal) {
	runtime_error(1, "Segmentation fault\n");
}

// initialize runtime stuff
static void init_runtime() {
	setlocale(LC_ALL, "de_DE.UTF8");
#ifdef _WIN32
	setlocale(LC_NUMERIC, "French_Canada.1252"); // print floats with , instead of . as seperator
	// enable utf-8 printing on windows
	// both of the functioncalls below are needed
	SetConsoleCP(CP_UTF8);
	SetConsoleOutputCP(CP_UTF8);
#endif // _WIN32

	signal(SIGSEGV, SegfaultHandler); // "catch" segfaults

	init_mt19937_64(); // seed the random generator

	initTable(get_ref_table()); // initialize the reference table
}

// end the runtime
static void end_runtime() {
	freeTable(get_ref_table()); // free the reference table (not the remaining entries, it should be empty)
}

extern int inbuilt_ddpmain(); // implicitly defined by the ddp code

// entry point of the final executable (needed by gcc)
int main() {
	init_runtime(); // initialize the runtime
	int ret = inbuilt_ddpmain(); // run the ddp code
	end_runtime(); // end the runtime
	return ret; // return the exit status of the ddp-code
}