/*
	This file implements extern functions from 
	Duden/Laufzeit.ddp
*/
#include "main.h"
#include "ddptypes.h"
#include "memory.h"
#include "ddpwindows.h"
#include <stdlib.h>
#include <stdio.h>

#ifdef DDPOS_WINDOWS
#include <io.h>
#else
#include <unistd.h>
#endif // DDPOS_WINDOWS

void Programm_Beenden(ddpint code) {
	ddp_end_runtime();
	exit(code);
}

void Laufzeitfehler(ddpstring* Nachricht, ddpint code) {
	ddp_runtime_error(code, Nachricht->str);
}

ddpbool Ist_Befehlszeile() {
#ifdef DDPOS_WINDOWS
	return _isatty(_fileno(stdin));
#else
	return isatty(STDOUT_FILENO);
#endif
}

void Betriebssystem(ddpstring* ret) {
#ifdef DDPOS_WINDOWS 
	#define OS "Windows"
#else 
	#define OS "Linux"
#endif
	ret->cap = sizeof(OS);
	ret->str = ALLOCATE(char, sizeof(OS));
	memcpy(ret->str, OS, sizeof(OS));
#undef OS
}