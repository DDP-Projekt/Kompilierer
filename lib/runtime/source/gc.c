#include "gc.h"
#include "ddptypes.h"
#include "debug.h"

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
void free_value(void* key, Value* val) {
	DBGLOG("free_value: %p", key);
	switch (val->kind)
	{
	case VK_STRING:
		free_string((ddpstring*)key);
		break;
	case VK_INT_LIST:
		free_ddpintlist((ddpintlist*)key);
		break;
	case VK_FLOAT_LIST:
		free_ddpfloatlist((ddpfloatlist*)key);
		break;
	case VK_BOOL_LIST:
		free_ddpboollist((ddpboollist*)key);
		break;
	case VK_CHAR_LIST:
		free_ddpcharlist((ddpcharlist*)key);
		break;
	case VK_STRING_LIST:
		free_ddpstringlist((ddpstringlist*)key);
		break;
	default:
		runtime_error(1, "invalid value kind\n"); // unreachable
		break;
	}
	if(!tableDelete(get_ref_table(), key)) { // delete the value from the ref table
		runtime_error(1, "free_value: key %p not found in refTable\n", key);
	}
}

// decrement the ref-count of a given garbage-collected object
void inbuilt_decrement_ref_count(void* key) {
	DBGLOG("inbuilt_decrement_ref_count: %p", key);
	Value val; // will hold the value from the ref-table
	// get the current value state
	if (tableGet(get_ref_table(), key, &val)) {
		val.reference_count--; // decrement the reference_count
		DBGLOG("new ref count: %d", val.reference_count);
		if (val.reference_count <= 0) {
			tableSet(get_ref_table(), key, val); // if this key is reused sometime in the future, the ref_count must be 0
			free_value(key, &val); // if it is 0, free the value
		}
		else tableSet(get_ref_table(), key, val); // otherwise override the value in the table with the new reference_count
	} else {
		runtime_error(1, "key %p not found in refTable\n", key); // unreachable
	}
}

// increment the ref-count of a given garbage-collected object
// if key is not already in the table, it is added (that's why kind is needed)
void inbuilt_increment_ref_count(void* key, uint8_t kind) {
	DBGLOG("inbuilt_increment_ref_count: %p", key);
	Value val; // will hold the value from the ref-table
	if (tableGet(get_ref_table(), key, &val)) { // key already in table
		val.reference_count++; // increment the reference_count
		DBGLOG("new ref count: %d", val.reference_count);
		tableSet(get_ref_table(), key, val); // otherwise override the value in the table with the new reference_count
	} else { // key must be added to the table
		DBGLOG("new key added");
		val.kind = kind;
		val.reference_count = 1;
		tableSet(get_ref_table(), key, val); // add the new Value to the table
	}
}