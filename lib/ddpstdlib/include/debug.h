/*
	defines useful macros and helper functions
	for debugging
*/
#ifndef DDP_DEBUG_H
#define DDP_DEBUG_H

#include <stdio.h>

//#define DEBUG // undef to remove logs and similar debug stuff

#ifdef DEBUG
// helper macro to log stuff in debug mode
#define DBGLOG(...) \
	{ printf("\n\t"); \
		printf(__VA_ARGS__); \
		printf("\n"); \
	}
#else
#define DBGLOG(...)
#endif // DEBUG

#endif // DDP_DEBUG_H