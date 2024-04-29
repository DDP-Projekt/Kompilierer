/*
	This file implements extern functions from
	Duden/Umbegungsvariablen.ddp
*/

#include "ddpmemory.h"
#include "ddptypes.h"
#include "ddpwindows.h"
#include <stdlib.h>
#include <string.h>

void Hole_Umgebungsvariable(ddpstring *ret, ddpstring *Name) {
	*ret = DDP_EMPTY_STRING;

	const char *env = getenv(Name->str);
	if (env) {
		ret->cap = strlen(env) + 1;
		ret->str = DDP_ALLOCATE(char, ret->cap);
		strcpy(ret->str, env);
		ret->str[ret->cap - 1] = '\0';
	}
}

void Setze_Umgebungsvariable(ddpstring *Name, ddpstring *Wert) {
#ifdef DDPOS_WINDOWS
	_putenv_s(Name->str, Wert->str);
#else
	setenv(Name->str, Wert->str, 1);
#endif // DDPOS_WINDOWS
}
