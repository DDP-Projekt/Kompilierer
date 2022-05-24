#ifndef DDP_HASH_TABLE_H
#define DDP_HASH_TABLE_H

#include <stdint.h>
#include <stdbool.h>

typedef struct {
	void* key;
	int value;
} Entry;

typedef struct {
	int count;
	int capacity;
	Entry* entries;
} Table;

void initTable(Table* table);
void freeTable(Table* table);
bool tableSet(Table* table, void* key, int value);
bool tableGet(Table* table, void* key, int* value);
bool tableDelete(Table* table, void* key);


uint32_t hashPointer(const char* key);

#endif