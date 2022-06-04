/*
	declares functions to interact with the garbage collector (reference counter)
*/
#ifndef DDP_GC_H
#define DDP_GC_H

#include "hashtable.h"

Table* get_ref_table(); // returns a pointer to the reference-table

#endif // DDP_GC_H