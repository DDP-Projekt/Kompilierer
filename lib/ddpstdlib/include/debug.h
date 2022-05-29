#ifndef DEBUG_H
#define DEBUG_H

#include <stdio.h>

#define DEBUG // undef to remove logs and similar debug stuff

#ifdef DEBUG
#define DBGLOG(...) \
	{ printf("\n\t"); \
		printf(__VA_ARGS__); \
		printf("\n"); \
	}
#else
#define DBGLOG(...)
#endif // DEBUG

#endif // DEBUG_H