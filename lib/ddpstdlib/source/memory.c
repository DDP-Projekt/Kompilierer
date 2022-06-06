#include "memory.h"
#include "debug.h"
#include <stdlib.h>
#include <stdio.h>

// used for allocation/reallocation and freeing of memory
// to allocate call reallocate(NULL, 0, size)
// to free call reallocate(ptr, oldsize, 0)
// to reallocate call reallocate(ptr, oldsize, newsize)
void* reallocate(void* pointer, size_t oldSize, size_t newSize) {
#ifdef DEBUG
	static unsigned long long allocatedBytes = 0;
#endif // DEBUG

	// newSize == 0 means free
	if (newSize == 0) {
		free(pointer);
#ifdef DEBUG
		allocatedBytes -= oldSize;
		DBGLOG("freed %llu bytes, now at %llu bytesAllocated", oldSize, allocatedBytes);
#endif // DEBUG
		return NULL;
	}

	// realloc can act as realloc or malloc
	// if pointer is NULL it acts as malloc
	// otherwise as realloc
	void* result = realloc(pointer, newSize);
#ifdef DEBUG
		allocatedBytes += newSize - oldSize;
		DBGLOG("allocated %llu bytes, now at %llu bytesAllocated", newSize - oldSize, allocatedBytes);
#endif // DEBUG
	if (result == NULL) { // out of memory
		printf("out of memory");
		exit(1);
	}

	return result;
}