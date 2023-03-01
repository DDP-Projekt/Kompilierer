#include "common.h"
#include "main.h"
#include <stdarg.h>

// print the error message to stderr and exit with exit_code
void runtime_error(int exit_code, const char* fmt, ...) {
	va_list argptr;
	va_start(argptr, fmt);

	fprintf(stderr, "\nLaufzeitfehler: ");
	vfprintf(stderr, fmt, argptr);

	va_end(argptr);

	end_runtime();
	exit(exit_code);
}