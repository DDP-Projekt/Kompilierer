/*
	This file implements extern functions from
	Duden/Umbegungsvariablen.ddp
*/

#include "DDP/ddptypes.h"
#include "DDP/ddpwindows.h"
#include <stdlib.h>

void Hole_Umgebungsvariable(ddpstring *ret, ddpstring *Name) {
	*ret = DDP_EMPTY_STRING;

	const char *env = getenv(DDP_GET_STRING_PTR(Name));
	if (env) {
		ddp_string_from_constant(ret, env);
	}
}

void Setze_Umgebungsvariable(ddpstring *Name, ddpstring *Wert) {
#ifdef DDPOS_WINDOWS
	_putenv_s(DDP_GET_STRING_PTR(Name), DDP_GET_STRING_PTR(Wert));
#else
	setenv(DDP_GET_STRING_PTR(Name), DDP_GET_STRING_PTR(Wert), 1);
#endif // DDPOS_WINDOWS
}
