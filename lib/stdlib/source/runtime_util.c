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
	end_runtime();
	exit(code);
}

void Laufzeitfehler(ddpstring* Nachricht, ddpint code) {
	runtime_error(code, Nachricht->str);
}

ddpbool Ist_Befehlszeile() {
#ifdef _WIN32
	return _isatty(_fileno(stdin));
#else
	return isatty(STDOUT_FILENO);
#endif
}

ddpstring* Betriebssystem() {
#ifdef _WIN32 
	#define OS "Windows"
#else 
	#define OS "Linux"
#endif
	ddpstring* os = ALLOCATE(ddpstring, 1);
	os->cap = sizeof(OS);
	os->str = ALLOCATE(char, sizeof(OS));
	memcpy(os->str, OS, sizeof(OS));
	return os;
#undef OS
}