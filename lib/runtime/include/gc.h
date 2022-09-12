/*
	declares functions to interact with the garbage collector (reference counter)
*/
#ifndef DDP_GC_H
#define DDP_GC_H

#include "hashtable.h"

Table* get_ref_table(); // returns a pointer to the reference-table

// free a garbage colleted value
// reference_count of val must be 0
// key is the pointer to the value
// val is needed to determine the type of value
void free_value(void* key, Value* val);

// decrement the ref-count of a given garbage-collected object
void _ddp_decrement_ref_count(void* key);
// increment the ref-count of a given garbage-collected object
// if key is not already in the table, it is added (that's why kind is needed)
void _ddp_increment_ref_count(void* key, uint8_t kind);

#endif // DDP_GC_H