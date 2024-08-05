/*
	This file implements extern functions from 
	Duden/Ausgabe.ddp
	Duden/Eingabe.ddp
	Duden/Datei_Ausgabe.ddp
	Duden/Datei_Eingabe.ddp
*/
#include "ddpmemory.h"
#include "ddptypes.h"
#include "ddpwindows.h"
#include "debug.h"
#include "utf8/utf8.h"
#include <math.h>
#include <stdarg.h>
#include <stdio.h>
#include <string.h>

#ifdef DDPOS_WINDOWS
#include <io.h>
#endif // DDPOS_WINDOWS

/*
	print functions
*/

void Schreibe_Zahl(ddpint p1) {
	printf(DDP_INT_FMT, p1);
}

void Schreibe_Kommazahl(ddpfloat p1) {
	if (isinf(p1)) {
		printf(p1 > 0 ? "Unendlich" : "-Unendlich");
	} else if (isnan(p1)) {
		printf("Keine Zahl (NaN)");
	} else {
		printf(DDP_FLOAT_FMT, p1);
	}
}

void Schreibe_Wahrheitswert(ddpbool p1) {
	printf(p1 ? "wahr" : "falsch");
}

void Schreibe_Buchstabe(ddpchar p1) {
	char temp[5];
	utf8_char_to_string(temp, p1);
	printf(DDP_STRING_FMT, temp);
}

void Schreibe_Text(ddpstring *p1) {
	// {NULL, 0} is a valid string, so we need to check for NULL
	printf(DDP_STRING_FMT, p1->str ? p1->str : "");
}

void Schreibe_Fehler(ddpstring *fehler) {
	fprintf(stderr, DDP_STRING_FMT, fehler->str);
}

#ifdef DDPOS_WINDOWS
// wrapper to get an error message for GetLastError()
// expects fmt to be of format "<message>%s"
static void runtime_error_getlasterror(int exit_code, const char *fmt) {
	char error_buffer[1024];
	DWORD error_code = GetLastError();
	if (!FormatMessageA(FORMAT_MESSAGE_FROM_SYSTEM | FORMAT_MESSAGE_IGNORE_INSERTS,
						NULL, error_code, MAKELANGID(LANG_NEUTRAL, SUBLANG_DEFAULT), error_buffer, sizeof(error_buffer), NULL)) {
		sprintf(error_buffer, "WinAPI Error Code %d (FormatMessageA failed with code %d)", error_code, GetLastError());
	}
	ddp_runtime_error(exit_code, fmt, error_buffer);
}

static HANDLE *get_stdin_handle(void) {
	static HANDLE stdin_hndl;
	static bool initialized = false;
	if (!initialized) {
		stdin_hndl = GetStdHandle(STD_INPUT_HANDLE);
		if (stdin_hndl == INVALID_HANDLE_VALUE) {
			runtime_error_getlasterror(1, "GetStdHandle failed: %s");
		}
	}
	return &stdin_hndl;
}
#endif // DDPOS_WINDOWS

ddpchar extern_lies_buchstabe(ddpboolref war_eof) {
	DDP_DBGLOG("extern_lies_buchstabe");
#ifdef DDPOS_WINDOWS // if stdin is a terminal type on windows
	if (_isatty(_fileno(stdin))) {
		wchar_t buff[2];
		char mbStr[5];
		unsigned long read;
		if (ReadConsoleW(*get_stdin_handle(), buff, 1, &read, NULL) == 0) {
			runtime_error_getlasterror(1, "ReadConsoleW failed: %s");
		}
		int size = WideCharToMultiByte(CP_UTF8, 0, buff, read, mbStr, sizeof(mbStr), NULL, NULL);
		if (size == 0) {
			runtime_error_getlasterror(1, "WideCharToMultiByte (1) failed: %s");
		}
		mbStr[size] = '\0';
		ddpchar ch;
		utf8_string_to_char(mbStr, (uint32_t *)&ch);
		if (ch == 26) {
			*war_eof = true; // set eof for ctrl+Z
		}
		return ch;
	} else {
#endif // DDPOS_WINDOWS
		char temp[5];
		temp[0] = getchar();
		if (temp[0] == EOF) {
			clearerr(stdin);
			*war_eof = true;
			return 0;
		}
		int i = utf8_indicated_num_bytes(temp[0]);
		for (int j = 1; j < i; j++) {
			temp[j] = getchar();
		}
		temp[i] = '\0';
		ddpchar result;
		utf8_string_to_char(temp, (uint32_t *)&result);
		return result;
#ifdef DDPOS_WINDOWS
	}
#endif // DDPOS_WINDOWS
}
