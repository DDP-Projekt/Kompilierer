#include "gc.h"
#include "memory.h"
#include "debug.h"

static garbage_collected* gc_head = NULL;

static garbage_collected* allocate_gc(size_t size, uint8_t kind) {
	DBGLOG("allocate_gc");
	garbage_collected* gc = reallocate(NULL, 0, size);
	gc->type = kind;
	gc->marked = false;
	gc->next = gc_head;
	gc_head = gc;
	return gc;
}

#define ALLOCATE_GC(type, kind) \
	(type*)allocate_gc(sizeof(type), kind)

ddpstring* allocate_ddpstring() {
	DBGLOG("allocate_ddpstring");
	ddpstring* str = ALLOCATE_GC(ddpstring, GC_STRING);
	return str;
}

void mark_gc(garbage_collected* gc) {
	DBGLOG("mark_gc");
	gc->marked = true;
}

static void free_gc(garbage_collected* gc) {
	DBGLOG("free_gc");
	switch (gc->type)
	{
	case GC_STRING:
	{
		ddpstring* str = (ddpstring*)gc;
		reallocate(str->str, sizeof(ddpchar) * str->len, 0);
		reallocate(str, sizeof(ddpstring), 0);
		break;
	}
	default:
		break;
	}
}

void GC() {
	DBGLOG("GC begin (%d bytes allocated)", get_allocated_bytes());
	for (garbage_collected* it = gc_head, *previous = NULL; it != NULL;) {
		if (it->marked) {
			garbage_collected* toBeFreed = it;
			it = it->next;
			if (previous != NULL) {
				previous->next = it;
			} else {
				gc_head = it;
			}
			free_gc(toBeFreed);
		} else {
			previous = it;
			it = it->next;
		}
	}
	DBGLOG("GC end (%d bytes allocated)", get_allocated_bytes());
}