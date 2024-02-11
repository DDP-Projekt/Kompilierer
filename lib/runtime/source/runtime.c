/*
	initialization for the ddp-c-runtime
	also defines the entry point of the executable
*/
#include "runtime.h"
#include <locale.h>
#include <signal.h>

#include "ddpwindows.h"

#include "ddpmemory.h"
#include "ddptypes.h"
#include "debug.h"

// should not be needed in production
// mainly for debugging
void SignalHandler(int signal) {
	if (signal == SIGSEGV) {
		ddp_end_runtime();
		ddp_runtime_error(1, "Segmentation fault\n");
	}
}

static ddpstringlist cmd_args; // holds the command line arguments as ddptype

// converts the command line arguments into a ddpstringlist
static void handle_args(int argc, char **argv) {
	cmd_args = (ddpstringlist){DDP_ALLOCATE(ddpstring, argc), (ddpint)argc, (ddpint)argc};
	DBGLOG("handle_args: %p", &cmd_args);

	for (int i = 0; i < argc; i++) {
		ddp_string_from_constant(&cmd_args.arr[i], argv[i]);
	}
}

// initialize runtime stuff
void ddp_init_runtime(int argc, char **argv) {
	DBGLOG("init_runtime");
#ifdef DDPOS_WINDOWS
	// the locales behaviour seems to change from time to time on windows
	// so this might change later
	setlocale(LC_ALL, "German_Germany.utf8");
	setlocale(LC_NUMERIC, "French_Canada.1252"); // somehow this is needed to get , instead of . as decimal seperator

	// enable utf-8 printing on windows
	// both of the functioncalls below are needed
	SetConsoleCP(CP_UTF8);
	SetConsoleOutputCP(CP_UTF8);
#else
	setlocale(LC_ALL, "de_DE.UTF-8");
#endif // DDPOS_WINDOWS

	signal(SIGSEGV, SignalHandler); // "catch" segfaults

	handle_args(argc, argv); // turn the commandline args into a ddpstringlist
}

// end the runtime
void ddp_end_runtime(void) {
	// to avoid stack overflows if a runtime_error causes another runtime_error
	static bool ending = false;
	if (ending) {
		return;
	}
	ending = true;

	DBGLOG("end_runtime");

	// free the cmd_args
	ddp_free_ddpstringlist(&cmd_args);
}

void Befehlszeilenargumente(ddpstringlist *ret) {
	ddp_deep_copy_ddpstringlist(ret, &cmd_args);
}
