/*
	This file implements extern functions from 
	Duden/Laufzeit.ddp
*/
#include "main.h"
#include "ddptypes.h"
#include "memory.h"

#ifdef _WIN32
#include <io.h>
#include <Windows.h>
#else
#include <unistd.h>
#endif // _WIN32

void Programm_Beenden(ddpint code) {
	ddp_end_runtime();
	exit(code);
}

void Laufzeitfehler(ddpstring* Nachricht, ddpint code) {
	ddp_runtime_error(code, Nachricht->str);
}

ddpbool Ist_Befehlszeile() {
#ifdef _WIN32
	return _isatty(_fileno(stdin));
#else
	return isatty(STDOUT_FILENO);
#endif
}

void Betriebssystem(ddpstring* ret) {
#ifdef _WIN32 
	#define OS "Windows"
#else 
	#define OS "Linux"
#endif
	ret->cap = sizeof(OS);
	ret->str = ALLOCATE(char, sizeof(OS));
	memcpy(ret->str, OS, sizeof(OS));
#undef OS
}