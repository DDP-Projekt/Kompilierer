#include "gc.h"
#include "ddptypes.h"
#include "debug.h"
#include <stdlib.h>

// the reference table
// holds pointers as keys to every
// garbage-collected ddp object currently in use
static Table refTable;

// returns a pointer to the reference-table
Table* get_ref_table() {
	return &refTable;
}

// free a garbage colleted value
// reference_count of val must be 0
// key is the pointer to the value
// val is needed to determine the type of value
static void free_value(void* key, Value* val) {
	DBGLOG("free_value: %p", key);
	switch (val->kind)
	{
	case VK_STRING:
		free_string((ddpstring*)key);
		break;
	default:
		DBGLOG("invalid value kind");
		exit(1);
		break;
	}
	tableDelete(get_ref_table(), key); // delete the value from the ref table
}

// decrement the ref-count of a given garbage-collected object
void inbuilt_decrement_ref_count(void* key) {
	DBGLOG("inbuilt_decrement_ref_count: %p", key);
	Value val; // will hold the value from the ref-table
	// get the current value state
	if (tableGet(get_ref_table(), key, &val)) {
		val.reference_count--; // decrement the reference_count
		if (val.reference_count <= 0) free_value(key, &val); // if it is 0, free the value
		else tableSet(get_ref_table(), key, val); // otherwise override the value in the table with the new reference_count
	} else {
		DBGLOG("key %p not found in refTable", key);
		exit(1);
	}
}

// increment the ref-count of a given garbage-collected object
// if key is not already in the table, it is added (that's why kind is needed)
void inbuilt_increment_ref_count(void* key, uint8_t kind) {
	DBGLOG("inbuilt_increment_ref_count: %p", key);
	Value val; // will hold the value from the ref-table
	if (tableGet(get_ref_table(), key, &val)) { // key already in table
		val.reference_count++; // increment the reference_count
		tableSet(get_ref_table(), key, val); // otherwise override the value in the table with the new reference_count
	} else { // key must be added to the table
		DBGLOG("new key added");
		val.kind = kind;
		val.reference_count = 1;
		tableSet(get_ref_table(), key, val); // add the new Value to the table
	}
}