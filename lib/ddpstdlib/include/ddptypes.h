/*
	declares types and functions to work with ddp types
*/
#ifndef DDP_TYPES_H
#define DDP_TYPES_H

#include "common.h"

// typedefs of primitive ddp types
typedef int64_t ddpint;
typedef double ddpfloat;
typedef bool ddpbool;
typedef int32_t ddpchar; // needs to be 32 bit to hold every possible unicode character

// a ddp string is a null-terminated utf8-encoded byte array
typedef struct {
	char* str; // the byte array
	ddpint cap; // the capacity of the array
} ddpstring;

// free a ddpstring
void free_string(ddpstring* str);

#endif // DDP_TYPES_H