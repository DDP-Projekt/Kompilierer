#include <string.h>
#include <stdlib.h>
#include "ddptypes.h"
#include "memory.h"
#include "debug.h"

static Table refTable;

Table* get_ref_table() {
	return &refTable;
}

ddpstring* inbuilt_string_from_constant(ddpchar* str, ddpint len) {
	DBGLOG("inbuilt_string_from_constant");
	ddpchar* string = ALLOCATE(ddpchar, len);
	for (int i = 0; i < len; i++) {
		string[i] = str[i];
	}
	ddpstring* dstr = ALLOCATE(ddpstring, 1);
	dstr->str = string;
	dstr->len = len;
	return dstr;
}

void inbuilt_free_string(ddpstring* str) {
	DBGLOG("inbuilt_free_string");
	FREE_ARRAY(ddpchar, str->str, str->len);
	FREE(ddpstring, str);
}

ddpstring* inbuilt_deep_copy_string(ddpstring* str) {
	DBGLOG("inbuilt_deep_copy_string");
	ddpchar* cpy = ALLOCATE(ddpchar, str->len);
	memcpy(cpy, str->str, sizeof(ddpchar) * str->len);
	ddpstring* cpystr = ALLOCATE(ddpstring, 1);
	cpystr->str = cpy;
	cpystr->len = str->len;
	return cpystr;
}

static void free_value(void* key, Value* val) {
	DBGLOG("free_value");
	switch (val->kind)
	{
	case VK_STRING:
		inbuilt_free_string((ddpstring*)key);
		break;
	default:
		DBGLOG("invalid value kind");
		exit(1);
		break;
	}
	tableDelete(get_ref_table(), key);
}

void inbuilt_decrement_ref_count(void* key) {
	DBGLOG("inbuilt_decrement_ref_count: %p", key);
	Value val;
	if (tableGet(get_ref_table(), key, &val)) {
		val.reference_count--;
		if (val.reference_count <= 0) free_value(key, &val); 
		else tableSet(get_ref_table(), key, val);
	} else {
		DBGLOG("key %p not found in refTable", key);
		exit(1);
	}
}

void inbuilt_increment_ref_count(void* key, uint8_t kind) {
	DBGLOG("inbuilt_increment_ref_count: %p", key);
	Value val;
	if (tableGet(get_ref_table(), key, &val)) {
		val.reference_count++;
		tableSet(get_ref_table(), key, val);
	} else {
		DBGLOG("new key added");
		val.kind = kind;
		val.reference_count = 1;
		tableSet(get_ref_table(), key, val);
	}
}