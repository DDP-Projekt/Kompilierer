/*
	This file implements extern functions from 
	Duden/Laufzeit.ddp
*/
#include "DDP/ddpmemory.h"
#include "DDP/ddptypes.h"
#include "DDP/ddpwindows.h"
#include "DDP/runtime.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#ifdef DDPOS_WINDOWS
#include <io.h>
#else
#include <unistd.h>
#endif // DDPOS_WINDOWS

#ifdef DDPOS_WINDOWS
#define DDP_MAX_PATH 260
#else
#define DDP_MAX_PATH 4096
#endif

void Programm_Beenden(ddpint code) {
	ddp_end_runtime();
	exit(code);
}

void Laufzeitfehler(ddpstring *Nachricht, ddpint code) {
	ddp_runtime_error(code, Nachricht->str);
}

ddpbool Ist_Befehlszeile(void) {
#ifdef DDPOS_WINDOWS
	return _isatty(_fileno(stdin));
#else
	return isatty(STDOUT_FILENO);
#endif
}

void Betriebssystem(ddpstring *ret) {
#ifdef DDPOS_WINDOWS
#define OS "Windows"
#else
#define OS "Linux"
#endif
	ret->cap = sizeof(OS);
	ret->str = DDP_ALLOCATE(char, sizeof(OS));
	memcpy(ret->str, OS, sizeof(OS));
#undef OS
}

void Arbeitsverzeichnis(ddpstring *ret) {
	char buffer[DDP_MAX_PATH];
	if (getcwd(buffer, sizeof(buffer)) != NULL) {
		int len = strlen(buffer) + 1;
		char *string = DDP_ALLOCATE(char, len);
		memcpy(string, buffer, len);

		ret->str = string;
		ret->cap = len;
		return;
	}

	// TODO: Error Handling
	*ret = DDP_EMPTY_STRING;
}
