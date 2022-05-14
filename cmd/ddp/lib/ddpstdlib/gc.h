#ifndef GC_H
#define GC_H

#include <stdint.h>
#include <stdbool.h>

enum {
	GC_STRING
};

struct garbage_collected {
	struct garbage_collected* next;
	bool marked;
	uint8_t type;
};

typedef struct garbage_collected garbage_collected;

typedef unsigned char byte;

typedef int64_t ddpint;

typedef double ddpfloat;

typedef bool ddpbool;

typedef wchar_t ddpchar;

typedef struct {
	garbage_collected gc;
	ddpchar* str;
	ddpint len;
} ddpstring;

ddpstring* allocate_ddpstring();
void mark_gc(garbage_collected* gc);

void GC();

#endif // GC_H