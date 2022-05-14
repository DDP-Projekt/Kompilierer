#include "memory.h"
#include <stdlib.h>
#include <stdio.h>
#include "gc.h"

#define NEXT_GC (1024 * 1024)
//#define DEBUG_STRESS_GC

static size_t bytes_allocated = 0;
size_t get_allocated_bytes() { return bytes_allocated; }

void* reallocate(void* pointer, size_t oldSize, size_t newSize) {
	bytes_allocated += newSize - oldSize;

	if (newSize > oldSize) {
#ifdef DEBUG_STRESS_GC
		GC();
#endif // DEBUG_STRESS_GC
		if (bytes_allocated > NEXT_GC) GC();
	}

	if (newSize == 0) {
		free(pointer);
		return NULL;
	}

	void* result = realloc(pointer, newSize);
	if (result == NULL) { // out of memory
		printf("out of memory");
		exit(1);
	}

  return result;
}