/*
	This file implements extern functions from 
	Duden/Ausgabe.ddp
	Duden/Eingabe.ddp
	Duden/Datei_Ausgabe.ddp
	Duden/Datei_Eingabe.ddp
*/
#include "ddptypes.h"
#include "utf8/utf8.h"
#include "memory.h"
#include "debug.h"
#include <math.h>
#include <stdarg.h>
#ifdef _WIN32
#include <io.h>
#include <Windows.h>
#endif // _WIN32

/*
	print functions
*/

void Schreibe_Zahl(ddpint p1) {
	printf("%lld", p1);
}

void Schreibe_Kommazahl(ddpfloat p1) {
	if (isinf(p1)){
		printf("Unendlich");
	}
	else if (isnan(p1)) {
		printf("Keine Zahl (NaN)");
	}
	else {
		printf("%.16g", p1);
	}
}

void Schreibe_Boolean(ddpbool p1) {
	printf(p1 ? "wahr" : "falsch");
}

void Schreibe_Buchstabe(ddpchar p1) {
	char temp[5];
	utf8_char_to_string(temp, p1);
	printf("%s", temp);
}

void Schreibe_Text(ddpstring* p1) {
	printf("%s", p1->str);
}

#ifdef _WIN32
// wrapper to get an error message for GetLastError()
// expects fmt to be of format "<message>%s"
static void runtime_error_getlasterror(int exit_code, const char* fmt) {
	char error_buffer[1024];
	DWORD error_code = GetLastError();
	if (!FormatMessageA(FORMAT_MESSAGE_FROM_SYSTEM | FORMAT_MESSAGE_IGNORE_INSERTS,
	NULL, error_code, MAKELANGID(LANG_NEUTRAL, SUBLANG_DEFAULT), error_buffer, 1024, NULL)) {
		sprintf(error_buffer, "WinAPI Error Code %d (FormatMessageA failed with code %d)", GetLastError());
	}
	runtime_error(exit_code, fmt, error_buffer);
}

static HANDLE* get_stdin_handle() {
	static HANDLE stdin_hndl;
	static bool initialized = false;
	if (!initialized) {
		stdin_hndl = GetStdHandle(STD_INPUT_HANDLE);
		if (stdin_hndl == INVALID_HANDLE_VALUE) runtime_error_getlasterror(1, "GetStdHandle failed: %s");
	}
	return &stdin_hndl;
}
#endif // _WIN32

ddpchar __Extern_Lies_Buchstabe(ddpboolref __war_eof) {
	DBGLOG("__Extern_Lies_Buchstabe");
#ifdef _WIN32 // if stdin is a terminal type on windows
	if (_isatty(_fileno(stdin))) {
		wchar_t buff[2];
		char mbStr[5];
		unsigned long read;
		if (ReadConsoleW(*get_stdin_handle(), buff, 1, &read, NULL) == 0) runtime_error_getlasterror(1, "ReadConsoleW failed: %s");
		int size = WideCharToMultiByte(CP_UTF8, 0, buff, read, mbStr, sizeof(mbStr), NULL, NULL);
		if (size == 0) runtime_error_getlasterror(1, "WideCharToMultiByte (1) failed: %s");
		mbStr[size] = '\0';
		ddpchar ch = utf8_string_to_char(mbStr);
		if (ch == 26) *__war_eof = true; // set eof for ctrl+Z
		return ch;
	} else {
#endif
	char temp[5];
	temp[0] = getchar();
	if (temp[0] == EOF)  {
		clearerr(stdin);
		*__war_eof = true;
		return 0;
	}
	int i = utf8_indicated_num_bytes(temp[0]);
	for (int j = 1; j < i; j++) temp[j] = getchar();
	temp[i] = '\0';
	return utf8_string_to_char(temp);
#ifdef _WIN32
	}
#endif
}

/*
	File IO
*/

static void write_error(ddpstringref ref, const char* fmt, ...) {
	char errbuff[1024];

	va_list argptr;
	va_start(argptr, fmt);

	int len = vsprintf(errbuff, fmt, argptr);
	
	va_end(argptr);

	(*ref)->str = reallocate((*ref)->str, (*ref)->cap, len+1);
	memcpy((*ref)->str, errbuff, len);
	(*ref)->cap = len+1;
	(*ref)->str[(*ref)->cap-1] = '\0';
}

ddpint Lies_Text_Datei(ddpstring* Pfad, ddpstringref ref) {
	FILE* file = fopen(Pfad->str, "r");
	ddpint ret = -1;
	if (file) {
		fseek(file, 0, SEEK_END); // seek the last byte in the file
		size_t string_size = ftell(file) + 1; // file_size + '\0'
		rewind(file); // go back to file start
		(*ref)->str = reallocate((*ref)->str, (*ref)->cap, string_size);
		(*ref)->cap = string_size;
		size_t read = fread((*ref)->str, sizeof(char), string_size-1, file);
		(*ref)->str[(*ref)->cap-1] = '\0';
		if (read != string_size-1) {
			ret = -1;
			write_error(ref, "Fehler beim Lesen von '%s': %s", Pfad->str, strerror(errno));
		} else {
			ret = read;
		}
		fclose(file);
	} else {
		ret = -1;
		write_error(ref, "Fehler beim Lesen von '%s': %s", Pfad->str, strerror(errno));
	}

	return ret;
}

ddpint Schreibe_Text_Datei(ddpstring* Pfad, ddpstring* text, ddpstringref fehler) {
	FILE* file = fopen(Pfad->str, "w");
	ddpint ret = -1;
	if (file) {
		ret = fprintf(file, text->str);
		fclose(file);
		if (ret < 0) {
			ret = -1;
			write_error(fehler, "Fehler beim Schreiben zu '%s': %s", Pfad->str, strerror(errno));
		}
	} else {
		write_error(fehler, "Fehler beim Schreiben zu '%s': %s", Pfad->str, strerror(errno));
	}
	return ret;
}