#ifndef DDP_HASH_TABLE_H
#define DDP_HASH_TABLE_H

#include <stdint.h>
#include <stdbool.h>

typedef enum {
	VK_STRING
} ValueKind;

typedef struct {
	int reference_count;
	uint8_t kind;
} Value;

typedef struct {
	void* key;
	Value value;
} Entry;

typedef struct {
	int count;
	int capacity;
	Entry* entries;
} Table;

void initTable(Table* table);
void freeTable(Table* table);
bool tableSet(Table* table, void* key, Value value);
bool tableGet(Table* table, void* key, Value* value);
bool tableDelete(Table* table, void* key);


uint32_t hashPointer(const char* key);

#endif