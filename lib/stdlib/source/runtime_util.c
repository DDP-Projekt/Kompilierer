/*
	This file implements extern functions from 
	Duden/Laufzeit.ddp
*/
#include "ddpmemory.h"
#include "ddptypes.h"
#include "ddpwindows.h"
#include "runtime.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#ifdef DDPOS_WINDOWS
#include <io.h>
#else
#include <unistd.h>
#endif // DDPOS_WINDOWS

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
	char buffer[PATH_MAX];
	if (getcwd(buffer, sizeof(buffer)) != NULL) {
		char *string = DDP_ALLOCATE(char, PATH_MAX + 1);
		memcpy(string, buffer, PATH_MAX);
		
		ret->str = string;
		ret->str[PATH_MAX] = '\0';
		ret->cap = PATH_MAX + 1;
		return;
	}
	
	ret->str = "";
	ret->cap = 0;
}