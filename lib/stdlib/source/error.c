#include "error.h"
#include "ddpos.h"
#include "ddpwindows.h"
#include "ddpmemory.h"
#include <string.h>
#include <stdarg.h>
#include <stdio.h>

void ddp_error(const char *prefix, bool use_errno) {
	ddpstring error = { 0 };
	error.cap = strlen(prefix) + 1;

	if (use_errno) {
		char* error_message = strerror(errno);
		error.cap += strlen(error_message);
		error.str = ALLOCATE(char, error.cap);
		strcpy(error.str, prefix);
		strcat(error.str, error_message);
	}
	else {
		error.str = ALLOCATE(char, error.cap);
		strcpy(error.str, prefix);
	}

	// will free error.str
	Setze_Fehler(&error);
}

#if DDPOS_WINDOWS
void ddp_error_win(const char* prefix) {
	ddpstring error = { 0 };
	error.cap = strlen(prefix) + 1;

	LPSTR error_message = NULL;
	DWORD error_code = GetLastError();
	if (!FormatMessageA(FORMAT_MESSAGE_ALLOCATE_BUFFER | FORMAT_MESSAGE_FROM_SYSTEM | FORMAT_MESSAGE_IGNORE_INSERTS,
		NULL, error_code, MAKELANGID(LANG_NEUTRAL, SUBLANG_DEFAULT), (LPSTR)&error_message, 0, NULL)) {
		error_message = "Failed to get Error Message from WinAPI";
	}

	error.cap += strlen(error_message);
	error.str = ALLOCATE(char, error.cap);
	strcpy(error.str, prefix);
	strcat(error.str, error_message);
	LocalFree(error_message);

	// will free error.str
	Setze_Fehler(&error);
}
#endif // DDPOS_WINDOWS
