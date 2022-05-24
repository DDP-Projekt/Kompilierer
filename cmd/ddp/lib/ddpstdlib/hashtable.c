#include "hashtable.h"

#include "memory.h"

void initTable(Table* table) {
	table->count = 0;
	table->capacity = 0;
	table->entries = NULL;
}

void freeTable(Table* table) {
	reallocate(table->entries, sizeof(Entry) * table->capacity, 0); // free the array
	initTable(table); // reset the table
}

static Entry* findEntry(Entry* entries, int capacity, void* key) {
	uint32_t hash = hashPointer(key, sizeof(key));
	uint32_t index = hash % capacity;
	Entry* tombstone = NULL;
	for (;;) {
		Entry* entry = &entries[index];
		if (entry->key == NULL) {
			if (entry->value == -1) {
				// Empty entry.
				return tombstone != NULL ? tombstone : entry;
			} else {
				// We found a tombstone.
				if (tombstone == NULL) tombstone = entry;
			}
			} else if (entry->key == key) {
			// We found the key.
			return entry;
		}

		index = (index + 1) % capacity;
	}
}

static void adjustCapacity(Table* table, int capacity) {
	Entry* entries = (Entry*)reallocate(NULL, 0, sizeof(Entry) * (capacity));
	for (int i = 0; i < capacity; i++) {
		entries[i].key = NULL;
		entries[i].value = -1;
	}

	table->count = 0;
	for (int i = 0; i < table->capacity; i++) {
		Entry* entry = &table->entries[i];
		if (entry->key == NULL) continue;

		Entry* dest = findEntry(entries, capacity, entry->key);
		dest->key = entry->key;
		dest->value = entry->value;
		table->count++;
	}

	reallocate(table->entries, sizeof(Entry) * (table->capacity), 0);
	table->entries = entries;
	table->capacity = capacity;
}

bool tableSet(Table* table, void* key, int value) {
	if (table->count + 1 > table->capacity * 0.75) {
		int capacity = ((table->capacity) < 8 ? 8 : (table->capacity) * 2);
		adjustCapacity(table, capacity);
	}

	Entry* entry = findEntry(table->entries, table->capacity, key);
	bool isNewKey = entry->key == NULL;
	if (isNewKey && entry->value == -1) table->count++;

	entry->key = key;
	entry->value = value;
	return isNewKey;
}

bool tableGet(Table* table, void* key, int* value) {
	if (table->count == 0) return false;

	Entry* entry = findEntry(table->entries, table->capacity, key);
	if (entry->key == NULL) return false;

	*value = entry->value;
	return true;
}

bool tableDelete(Table* table, void* key) {
	if (table->count == 0) return false;

	// Find the entry.
	Entry* entry = findEntry(table->entries, table->capacity, key);
	if (entry->key == NULL) return false;

	// Place a tombstone in the entry.
	entry->key = NULL;
	entry->value = 0;
	return true;
}

uint32_t hashPointer(const char* key, int length) {
	uint32_t hash = 2166136261u;
	for (int i = 0; i < length; i++) {
		hash ^= (uint8_t)key[i];
		hash *= 16777619;
	}
	return hash;
}