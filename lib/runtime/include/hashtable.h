/*
	inspired (stolen) from craftinginterpreters:
	https://craftinginterpreters.com/hash-tables.html
*/
#ifndef DDP_HASH_TABLE_H
#define DDP_HASH_TABLE_H

#include "common.h"

// type of the pointer in a Value
typedef enum {
	VK_STRING, // ValueKind_STRING
	VK_INT_LIST,
	VK_FLOAT_LIST,
	VK_BOOL_LIST,
	VK_CHAR_LIST,
	VK_STRING_LIST,
} ValueKind;

// a single garbage-collected value
typedef struct {
	int reference_count; // references to this value
	uint8_t kind; // kind of the Value (from the enum ValueKind)
} Value;

// a single Entry in a HashTable
typedef struct {
	void* key; // pointer to the Actual Value (also the key)
	Value value; // information about the value which key points to
} Entry;

// HashTable struct
typedef struct {
	int count; // count of Entries
	int capacity; // capacity of the entries array
	Entry* entries; // array of entries
} Table;

// these functions DO NOT take care of freeing the values
// pointed to by the keys
void initTable(Table* table); // initialize a HashTable
void freeTable(Table* table); // free the HashTable (does not free the Values pointed to by the keys)
bool tableSet(Table* table, void* key, Value value); // overwrites/adds a value to the table. returns true if a new entry was created
bool tableGet(Table* table, void* key, Value* value); // returns true if the key is in the table and stores the value in value. returns false otherwise
bool tableDelete(Table* table, void* key); // removes a key from the table. returns true if something was removed

#endif // DDP_HASH_TABLE_H