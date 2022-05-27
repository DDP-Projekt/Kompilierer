#ifndef MEMORY_H
#define MEMORY_H

#include <stddef.h>

void* reallocate(void* pointer, size_t oldSize, size_t newSize);
size_t get_allocated_bytes();

#endif // MEMORY_H