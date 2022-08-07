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
#include "ddptypes.h"
#include "memory.h"

// should not be needed in production
// mainly for debugging
static void SegfaultHandler(int signal) {
	end_runtime();
	runtime_error(1, "Segmentation fault\n");
}

static ddpstringlist* cmd_args = NULL; // holds the command line arguments as ddptype

// converts the command line arguments into a ddpstringlist
static void handle_args(int argc, char** argv) {
	cmd_args = ALLOCATE(ddpstringlist, 1);
	DBGLOG("handle_args: %p", cmd_args);
	inbuilt_increment_ref_count(cmd_args, VK_STRING_LIST);
	cmd_args->cap = argc;
	cmd_args->len = cmd_args->cap;
	cmd_args->arr = ALLOCATE(ddpstring*, cmd_args->cap);
	for (size_t i = 0; i < argc; i++) {
		cmd_args->arr[i] = ALLOCATE(ddpstring, 1);
		cmd_args->arr[i]->cap = strlen(argv[i]) + 1;
		cmd_args->arr[i]->str = ALLOCATE(char, cmd_args->arr[i]->cap);
		cmd_args->arr[i]->str[cmd_args->arr[i]->cap-1] = '\0';
		strcpy(cmd_args->arr[i]->str, argv[i]);
		inbuilt_increment_ref_count(cmd_args->arr[i], VK_STRING);
	}
}

// initialize runtime stuff
void init_runtime(int argc, char** argv) {
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

	handle_args(argc, argv); // turn the commandline args into a ddpstringlist
}

static void free_ref_table() {
	// free all possibly remaining entries (there should be none, but maybe we segfaulted or aborted...)
	Table* ref_table = get_ref_table();
	// free all lists first to avoid freeing elements before their list has been freed
	for (size_t i = 0; i < ref_table->capacity; i++) {
		Entry* entry = &ref_table->entries[i];
		if (entry->value.kind > 0 && entry->key != NULL && entry->value.reference_count > 0) {
			free_value(entry->key, &entry->value);
		}
	}
	// now free all remaining elements
	for (size_t i = 0; i < ref_table->capacity; i++) {
		Entry* entry = &ref_table->entries[i];
		if (entry->key != NULL && entry->value.reference_count > 0) {
			free_value(entry->key, &entry->value);
		}
	}
	freeTable(get_ref_table()); // free the reference table (not the remaining entries, it should be empty)
}

// end the runtime
void end_runtime() {
	// to avoid stack overflows if a runtime_error causes another runtime_error
	static bool ending = false;
	if (ending) {
		return;
	}
	ending = true;

	DBGLOG("end_runtime");

	// free the cmd_args
	inbuilt_decrement_ref_count(cmd_args);

	free_ref_table();
}

extern int inbuilt_ddpmain(); // implicitly defined by the ddp code

// entry point of the final executable (needed by gcc)
int main(int argc, char** argv) {
	init_runtime(argc, argv); // initialize the runtime
	int ret = inbuilt_ddpmain(); // run the ddp code
	end_runtime(); // end the runtime
	return ret; // return the exit status of the ddp-code
}

extern ddpstringlist* inbuilt_deep_copy_ddpstringlist(ddpstringlist* list);

ddpstringlist* ddpextern_Befehlszeilenargumente() {
	return inbuilt_deep_copy_ddpstringlist(cmd_args);
}