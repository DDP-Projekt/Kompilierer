/*
	declares types and functions to work with ddp types
*/
#ifndef DDP_TYPES_H
#define DDP_TYPES_H

#include <stddef.h>
#include <stdint.h>
#include <stdbool.h>

// typedefs of primitive ddp types
typedef int64_t ddpint;
typedef double ddpfloat;
typedef bool ddpbool;
typedef wchar_t ddpchar;

// a ddp string is basically a ddpchar array
typedef struct {
	ddpchar* str; // the ddpchar array
	ddpint len; // number of characters in the string and the array
	//ddpint cap; // added later
} ddpstring;

// allocate and create a ddpstring from a constant ddpchar array
ddpstring* inbuilt_string_from_constant(ddpchar* str, ddpint len);
// free a ddpstring
void free_string(ddpstring* str);
// allocate a new ddpstring as copy of str
ddpstring* inbuilt_deep_copy_string(ddpstring* str);

#endif // DDP_TYPES_H