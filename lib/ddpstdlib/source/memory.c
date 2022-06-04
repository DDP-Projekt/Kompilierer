#include "memory.h"
#include <stdlib.h>
#include <stdio.h>

// used for allocation/reallocation and freeing of memory
// to allocate call reallocate(NULL, 0, size)
// to free call reallocate(ptr, oldsize, 0)
// to reallocate call reallocate(ptr, oldsize, newsize)
void* reallocate(void* pointer, size_t oldSize, size_t newSize) {
	// newSize == 0 means free
	if (newSize == 0) {
		free(pointer);
		return NULL;
	}

	// realloc can act as realloc or malloc
	// if pointer is NULL it acts as malloc
	// otherwise as realloc
	void* result = realloc(pointer, newSize);
	if (result == NULL) { // out of memory
		printf("out of memory");
		exit(1);
	}

	return result;
}