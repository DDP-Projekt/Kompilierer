#include "DDP/error.h"
#include "DDP/ddpmemory.h"
#include "DDP/ddpos.h"
#include "DDP/ddptypes.h"
#include "DDP/ddpwindows.h"
#include "DDP/debug.h"
#include <stdarg.h>
#include <stdio.h>
#include <string.h>

#define ERROR_BUFFER_SIZE 1024

void ddp_error(const char *prefix, bool use_errno, ...) {
	char error_buffer[ERROR_BUFFER_SIZE];
	va_list args;
	va_start(args, use_errno);
	vsnprintf(error_buffer, ERROR_BUFFER_SIZE, prefix, args);
	va_end(args);

	if (use_errno) {
		char *error_message = strerror(errno);
		strncat(error_buffer, error_message, ERROR_BUFFER_SIZE - strlen(error_buffer) - 1);
	}

	ddpstring error;
	ddp_string_from_constant(&error, error_buffer);
	DDP_DBGLOG("Setze_Fehler: " DDP_STRING_FMT, DDP_STRING_DATA(&error));
	// will free error.str
	Setze_Fehler(&error);
}

#if DDPOS_WINDOWS
void ddp_error_win(const char *prefix, ...) {
	char error_buffer[ERROR_BUFFER_SIZE];
	va_list args;
	va_start(args, prefix);
	vsnprintf(error_buffer, ERROR_BUFFER_SIZE, prefix, args);
	va_end(args);

	LPSTR error_message = NULL;
	DWORD error_code = GetLastError();
	bool need_free = true;
	if (!FormatMessageA(FORMAT_MESSAGE_ALLOCATE_BUFFER | FORMAT_MESSAGE_FROM_SYSTEM | FORMAT_MESSAGE_IGNORE_INSERTS,
						NULL, error_code, MAKELANGID(LANG_NEUTRAL, SUBLANG_DEFAULT), (LPSTR)&error_message, 0, NULL)) {
		error_message = "Failed to get Error Message from WinAPI";
		need_free = false;
	}

	strncat(error_buffer, error_message, ERROR_BUFFER_SIZE - strlen(error_buffer) - 1);
	if (need_free) {
		LocalFree(error_message);
	}

	ddpstring error;
	ddp_string_from_constant(&error, error_buffer);
	DDP_DBGLOG("Setze_Fehler: " DDP_STRING_FMT, DDP_STRING_DATA(&error));
	// will free error.str
	Setze_Fehler(&error);
}
#endif // DDPOS_WINDOWS
