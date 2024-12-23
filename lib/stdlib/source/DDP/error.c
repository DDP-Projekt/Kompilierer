#include "DDP/error.h"
#include "DDP/ddpmemory.h"
#include "DDP/ddpos.h"
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

	ddpstring error = {0};
	error.cap = strlen(error_buffer) + 1;
	error.str = DDP_ALLOCATE(char, error.cap);
	memcpy(error.str, error_buffer, error.cap);

	if (use_errno) {
		char *error_message = strerror(errno);
		size_t msg_len = strlen(error_message);
		error.str = DDP_GROW_ARRAY(char, error.str, error.cap, error.cap + msg_len);
		error.cap += msg_len;
		strcat(error.str, error_message);
	}

	DDP_DBGLOG("Setze_Fehler: " DDP_STRING_FMT, error.str);
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

	ddpstring error = {0};
	error.cap = strlen(prefix) + 1;
	error.str = DDP_ALLOCATE(char, error.cap);
	memcpy(error.str, error_buffer, error.cap);

	LPSTR error_message = NULL;
	DWORD error_code = GetLastError();
	if (!FormatMessageA(FORMAT_MESSAGE_ALLOCATE_BUFFER | FORMAT_MESSAGE_FROM_SYSTEM | FORMAT_MESSAGE_IGNORE_INSERTS,
						NULL, error_code, MAKELANGID(LANG_NEUTRAL, SUBLANG_DEFAULT), (LPSTR)&error_message, 0, NULL)) {
		error_message = "Failed to get Error Message from WinAPI";
	}

	size_t msg_len = strlen(error_message);
	error.str = DDP_GROW_ARRAY(char, error.str, error.cap, error.cap + msg_len);
	error.cap += msg_len;
	strcat(error.str, error_message);
	LocalFree(error_message);

	DDP_DBGLOG("Setze_Fehler: " DDP_STRING_FMT, error.str);
	// will free error.str
	Setze_Fehler(&error);
}
#endif // DDPOS_WINDOWS
