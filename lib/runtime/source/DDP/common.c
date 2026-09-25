#include "DDP/common.h"
#include "DDP/ddpwindows.h"
#include "DDP/debug.h"
#include "DDP/runtime.h"
#include <inttypes.h>
#include <stdarg.h>
#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>

#define UNW_LOCAL_ONLY
#include "libunwind.h"

#include "backtrace.h"

// print the error message to stderr and exit with exit_code
void ddp_runtime_error(int exit_code, const char *fmt, ...) {
	DDP_DBGLOG("runtime_error: %d, %s", exit_code, fmt);

	va_list argptr;
	va_start(argptr, fmt);

	fprintf(stderr, "\nLaufzeitfehler: ");
	vfprintf(stderr, fmt, argptr);

	va_end(argptr);

	print_backtrace();

	ddp_end_runtime();
	exit(exit_code);
}

// lazily created once and cached, as required by backtrace_create_state()
static struct backtrace_state *bt_state = NULL;
static bool bt_state_initialized = false;

static void bt_error_callback(void *data, const char *msg, int errnum) {
	(void)data;
	(void)msg;
	(void)errnum;
	DDP_DBGLOG("libbacktrace error (%d): %s", errnum, msg);
}

// data passed through backtrace_pcinfo() to bt_pcinfo_callback()
struct bt_pcinfo_data {
	int frame;
	uintptr_t pc;
	bool found;
};

// called once per (possibly inlined) frame backtrace_pcinfo() resolves for a
// given pc; prints directly instead of collecting into a struct first, so an
// inlined call chain (multiple calls for a single pc) is printed in full
static int bt_pcinfo_callback(void *data, uintptr_t pc, const char *filename, int lineno, const char *function) {
	(void)pc;
	struct bt_pcinfo_data *info = data;
	if (filename == NULL && function == NULL) {
		return 0;
	}
	info->found = true;
	fprintf(stderr, "#%d 0x%" PRIxPTR ": %s at %s:%d\n", info->frame, info->pc,
			function != NULL ? function : "??", filename != NULL ? filename : "??", lineno);
	return 0;
}

void print_backtrace(void) {
	if (!bt_state_initialized) {
		// filename NULL: libbacktrace determines the running executable's own
		// path (via GetModuleFileName on PE/COFF, /proc/self/exe on ELF, ...)
		bt_state = backtrace_create_state(NULL, /*threaded=*/1, bt_error_callback, NULL);
		bt_state_initialized = true;
	}

	unw_cursor_t cursor;
	unw_context_t context;
	unw_getcontext(&context);
	unw_init_local(&cursor, &context);

	fprintf(stderr, "--- backtrace ---\n");
	int frame = 0;
	while (unw_step(&cursor) > 0 && frame < 64) {
		unw_word_t pc, off;
		unw_get_reg(&cursor, UNW_REG_IP, &pc);

		// try to resolve name + file:line from the DWARF debug info first
		struct bt_pcinfo_data info = {frame, (uintptr_t)pc, false};
		if (bt_state != NULL) {
			backtrace_pcinfo(bt_state, (uintptr_t)pc, bt_pcinfo_callback, bt_error_callback, &info);
		}

		if (!info.found) {
			char proc_name[255];

			// pc/off are unw_word_t (uintptr_t): on Windows (LLP64) `long` is only
			// 32 bits and would silently truncate the address, so PRIxPTR is used
			// instead to always match uintptr_t's actual width
			if (unw_get_proc_name(&cursor, proc_name, sizeof(proc_name), &off) == 0) {
				fprintf(stderr, "#%d 0x%" PRIxPTR ": %s + 0x%" PRIxPTR "\n", frame, (uintptr_t)pc, proc_name, (uintptr_t)off);
			} else {
				fprintf(stderr, "#%d 0x%" PRIxPTR ": -- <unknown>\n", frame, (uintptr_t)pc);
			}
		}

		frame++;
	}
	fprintf(stderr, "-----------------\n");
}
