/*
	This file implements extern functions from
	Duden/Umbegungsvariablen.ddp
*/

#include <stdlib.h>
#include "ddptypes.h"
#include "memory.h"
#ifdef _WIN32
#include <Windows.h>
#endif // _WIN32

void Hole_Umgebungsvariable(ddpstring* ret, ddpstring* Name) {
	ret->str = NULL;
	ret->cap = 0;

	const char* env = getenv(Name->str);
	if (env) {
		ret->cap = strlen(env) + 1;
		ret->str = ALLOCATE(char, ret->cap);
		strcpy(ret->str, env);
		ret->str[ret->cap-1] = '\0';
	} else {
		ret->cap = 1;
		ret->str = ALLOCATE(char, 1);
		ret->str[0] = '\0';
	}
}

void Setze_Umgebungsvariable(ddpstring* Name, ddpstring* Wert) {
#ifdef _WIN32
	_putenv_s(Name->str, Wert->str);
#else
	setenv(Name->str, Wert->str, 1);
#endif // _WIN32
}