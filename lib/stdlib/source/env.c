#include <stdlib.h>
#include "ddptypes.h"
#include "memory.h"
#ifdef _WIN32
#include <Windows.h>
#endif // _WIN32

ddpstring* Hole_Umgebungsvariable(ddpstring* Name) {
	ddpstring* value = ALLOCATE(ddpstring, 1);
	value->str = NULL;
	value->cap = 0;

	const char* env = getenv(Name->str);
	if (env) {
		value->cap = strlen(env) + 1;
		value->str = ALLOCATE(char, value->cap);
		strcpy(value->str, env);
		value->str[value->cap-1] = '\0';
	} else {
		value->cap = 1;
		value->str = ALLOCATE(char, 1);
		value->str[0] = '\0';
	}

	return value;
}

void Setze_Umgebungsvariable(ddpstring* Name, ddpstring* Wert) {
#ifdef _WIN32
	_putenv_s(Name->str, Wert->str);
#else
	setenv(Name->str, Wert->str, 1);
#endif // _WIN32
}