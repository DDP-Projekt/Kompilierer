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
	ddp_runtime_error(code, DDP_STRING_DATA(Nachricht));
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
	ddp_string_from_constant(ret, OS);
#undef OS
}

void Arbeitsverzeichnis(ddpstring *ret) {
	char buffer[DDP_MAX_PATH];
	if (getcwd(buffer, sizeof(buffer)) != NULL) {
		ddp_string_from_constant(ret, buffer);
		return;
	}

	// TODO: Error Handling
	*ret = DDP_EMPTY_STRING;
}
