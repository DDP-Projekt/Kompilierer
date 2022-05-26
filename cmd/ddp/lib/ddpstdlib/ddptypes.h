#ifndef DDPTYPES_H
#define DDPTYPES_H

#include <stdint.h>
#include <stdbool.h>
#include "hashtable.h"

typedef unsigned char byte;
typedef int64_t ddpint;
typedef double ddpfloat;
typedef bool ddpbool;
typedef wchar_t ddpchar;

typedef struct {
	ddpchar* str;
	ddpint len;
} ddpstring;

ddpstring* inbuilt_string_from_constant(ddpchar* str, ddpint len);
void inbuilt_free_string(ddpstring* str);
ddpstring* inbuilt_deep_copy_string(ddpstring* str);

Table* get_ref_table();

#endif // DDPTYPES_H