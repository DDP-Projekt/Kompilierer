/*
	initialization for the ddp-c-runtime
	also defines the entry point of the executable
*/
#include "main.h"
#include <locale.h>
#include <signal.h>
#ifdef _WIN32
#include <Windows.h>
#endif // _WIN32
#include "gc.h"
#include "debug.h"

// should not be needed in production
// mainly for debugging
static void SegfaultHandler(int signal) {
	end_runtime();
	runtime_error(1, "Segmentation fault\n");
}

// initialize runtime stuff
void init_runtime() {
	DBGLOG("init_runtime");
	setlocale(LC_ALL, "de_DE.UTF8");
#ifdef _WIN32
	setlocale(LC_NUMERIC, "French_Canada.1252"); // print floats with , instead of . as seperator
	// enable utf-8 printing on windows
	// both of the functioncalls below are needed
	SetConsoleCP(CP_UTF8);
	SetConsoleOutputCP(CP_UTF8);
#endif // _WIN32

	signal(SIGSEGV, SegfaultHandler); // "catch" segfaults

	initTable(get_ref_table()); // initialize the reference table
}

// end the runtime
void end_runtime() {
	DBGLOG("end_runtime");
	// free all possibly remaining entries (there should be none, but maybe we segfaulted or aborted...)
	Table* ref_table = get_ref_table();
	for (size_t i = 0; i < ref_table->capacity; i++) {
		Entry* entry = &ref_table->entries[i];
		if (entry->key != NULL && entry->value.reference_count > 0) {
			free_value(entry->key, &entry->value);
		}
	}
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