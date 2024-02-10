#include "ddpmemory.h"
#include "debug.h"
#include <stdlib.h>

// used for allocation/reallocation and freeing of memory
// to allocate call reallocate(NULL, 0, size)
// to free call reallocate(ptr, oldsize, 0)
// to reallocate call reallocate(ptr, oldsize, newsize)
void *ddp_reallocate(void *pointer, size_t oldSize, size_t newSize) {
#ifdef DDP_DEBUG
	static unsigned long long allocatedBytes = 0;
#endif // DDP_DEBUG

	// newSize == 0 means free
	if (newSize == 0) {
		free(pointer);
#ifdef DDP_DEBUG
		allocatedBytes -= oldSize;
		DBGLOG("freed %lu bytes, now at %llu bytesAllocated", oldSize, allocatedBytes);
#endif // DDP_DEBUG
		return NULL;
	}

	// GNU libc's realloc should already do this, but just to be sure
	if (oldSize == newSize) {
		return pointer;
	}

	// realloc can act as realloc or malloc
	// if pointer is NULL it acts as malloc
	// otherwise as realloc
	void *result = realloc(pointer, newSize);
#ifdef DDP_DEBUG
	allocatedBytes += newSize - oldSize;
	DBGLOG("allocated %lu bytes, now at %llu bytesAllocated", newSize - oldSize, allocatedBytes);
#endif					  // DDP_DEBUG
	if (result == NULL) { // out of memory
		ddp_runtime_error(1, "out of memory\n");
	}

	return result;
}
