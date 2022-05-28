#ifndef DEBUG_H
#define DEBUG_H

#include <stdio.h>

#define DBGLOG(...) \
	{ printf("\n\t"); \
		printf(__VA_ARGS__); \
		printf("\n"); \
	}
//#define DBGLOG(...)

#endif // DEBUG_H