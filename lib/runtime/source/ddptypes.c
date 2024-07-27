#include "ddptypes.h"
#include "ddpmemory.h"
#include "debug.h"
#include <stdlib.h>
#include <string.h>

// allocate and create a ddpstring from a constant char array
// str must be null-terminated
void ddp_string_from_constant(ddpstring *ret, char *str) {
	DDP_DBGLOG("_ddp_string_from_constant: ret: %p, str: %s", ret, str);
	size_t size = strlen(str) + 1;
	if (size == 1) {
		*ret = DDP_EMPTY_STRING;
		return;
	}

	char *string = DDP_ALLOCATE(char, size); // the char array of the string (plus null terminator)
	// copy the passed char array
	memcpy(string, str, size);

	// set the string fields
	ret->str = string;
	ret->cap = size;
}

// free a ddpstring
void ddp_free_string(ddpstring *str) {
	DDP_DBGLOG("free_string: %p", str);
	DDP_FREE_ARRAY(char, str->str, str->cap); // free the character array
}

// allocate a new ddpstring as copy of str
void ddp_deep_copy_string(ddpstring *ret, ddpstring *str) {
	DDP_DBGLOG("_ddp_deep_copy_string: %p, ret: %p", str, ret);
	if (ret == str) {
		return;
	}
	char *cpy = DDP_ALLOCATE(char, str->cap); // allocate the char array for the copy
	memcpy(cpy, str->str, str->cap);		  // copy the chars
	// set the fields of the copy
	ret->str = cpy;
	ret->cap = str->cap;
}

// returns wether the length of str is 0
ddpbool ddp_string_empty(ddpstring *str) {
	return str->str == NULL || str->cap <= 0 || (str->str[0] == '\0');
}

ddpint ddp_strlen(ddpstring *str) {
	if (str == NULL || str->str == NULL) {
		return 0;
	}
	return strlen(str->str);
}

extern vtable ddpint_vtable;
extern vtable ddpfloat_vtable;
extern vtable ddpbool_vtable;
extern vtable ddpchar_vtable;

static bool is_primitive_vtable(vtable *table) {
	return table != NULL && table->free_func == NULL;
}

// frees the given any
void ddp_free_any(ddpany *any) {
	DDP_DBGLOG("free_any: %p, vtable: %p", any, any->vtable_ptr);

	if (!is_primitive_vtable(any->vtable_ptr) && any->value_ptr != NULL) {
		// free the underlying value
		any->vtable_ptr->free_func(any->value_ptr);
	}

	// free the memory allocated for the value itself
	if (any->value_ptr != NULL) {
		ddp_reallocate(any->value_ptr, any->value_size, 0);
	}
}

// places a copy of any in ret
void ddp_deep_copy_any(ddpany *ret, ddpany *any) {
	DDP_DBGLOG("deep_copy_any: %p", any);
	// copy metadata
	ret->value_size = any->value_size;
	ret->vtable_ptr = any->vtable_ptr;

	// allocate space for the underlying value
	ret->value_ptr = ddp_reallocate(NULL, 0, ret->value_size);
	if (is_primitive_vtable(ret->vtable_ptr)) {
		memcpy(ret->value_ptr, any->value_ptr, ret->value_size);
	} else if (ret->vtable_ptr != NULL) {
		// deep copy the underlying value
		ret->vtable_ptr->deep_copy_func(ret->value_ptr, any->value_ptr);
	}
}

// compares two any
ddpbool ddp_any_equal(ddpany *any1, ddpany *any2) {
	DDP_DBGLOG("any_equal: %p, %p", any1, any2);
	if (any1->value_size != any2->value_size) {
		return false;
	}

	if (any1->vtable_ptr != any2->vtable_ptr) {
		return false;
	}

	// standardwert
	if (any1->vtable_ptr == NULL) {
		return true;
	}

	if (is_primitive_vtable(any1->vtable_ptr)) {
		return memcmp(any1->value_ptr, any2->value_ptr, any1->value_size) == 0;
	}

	return any1->vtable_ptr->equal_func(any1->value_ptr, any2->value_ptr);
}
