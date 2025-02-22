#include "DDP/ddptypes.h"
#include "DDP/ddpmemory.h"
#include "DDP/debug.h"
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

	if (any->vtable_ptr == NULL) {
		return;
	}

	if (!is_primitive_vtable(any->vtable_ptr) && any->value_ptr != NULL) {
		// free the underlying value
		any->vtable_ptr->free_func(DDP_ANY_VALUE_PTR(any));
	}

	// free the memory allocated for the value itself
	if (!DDP_IS_SMALL_ANY(any) && any->value_ptr != NULL) {
		ddp_reallocate(any->value_ptr, any->vtable_ptr->type_size, 0);
	}
}

// places a copy of any in ret
void ddp_deep_copy_any(ddpany *ret, ddpany *any) {
	DDP_DBGLOG("deep_copy_any: %p", any);
	// copy metadata
	ret->vtable_ptr = any->vtable_ptr;

	if (ret->vtable_ptr == NULL) {
		return;
	}

	// allocate space for the underlying value
	if (!DDP_IS_SMALL_ANY(any)) {
		DDP_DBGLOG("allocating space for any: %lld", ret->vtable_ptr->type_size);
		ret->value_ptr = ddp_reallocate(NULL, 0, ret->vtable_ptr->type_size);
	} else {
		DDP_DBGLOG("not allocating for small any");
	}

	if (is_primitive_vtable(ret->vtable_ptr)) {
		memcpy(&ret->value, &any->value, DDP_SMALL_ANY_BUFF_SIZE);
	} else if (ret->vtable_ptr != NULL) {
		// deep copy the underlying value
		ret->vtable_ptr->deep_copy_func(DDP_ANY_VALUE_PTR(ret), DDP_ANY_VALUE_PTR(any));
	}
}

// compares two any
ddpbool ddp_any_equal(ddpany *any1, ddpany *any2) {
	DDP_DBGLOG("any_equal: %p, %p", any1, any2);
	if (any1->vtable_ptr != any2->vtable_ptr) {
		return false;
	}

	// standardwert
	if (any1->vtable_ptr == NULL) {
		return true;
	}

	ddpint any2_size = any2->vtable_ptr == NULL ? -1 : any2->vtable_ptr->type_size;

	if (any1->vtable_ptr->type_size != any2_size) {
		return false;
	}

	if (is_primitive_vtable(any1->vtable_ptr)) {
		return memcmp(&any1->value, &any2->value, any1->vtable_ptr->type_size) == 0;
	}

	return any1->vtable_ptr->equal_func(DDP_ANY_VALUE_PTR(any1), DDP_ANY_VALUE_PTR(any2));
}
