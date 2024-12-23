#include "DDP/ddptypes.h"
#include "DDP/common.h"
#include "DDP/ddpmemory.h"
#include "DDP/debug.h"
#include <stdlib.h>
#include <string.h>

// allocate and create a ddpstring from a constant char array
// str must be null-terminated
void ddp_string_from_constant(ddpstring *ret, const char *str) {
	DDP_DBGLOG("ddp_string_from_constant: ret: %p, str: %s", ret, str);
	size_t size = strlen(str) + 1;
	if (size <= 1) {
		*ret = DDP_EMPTY_STRING;
		return;
	}

	// set the string fields
	ret->cap = size;

	char *string = &ret->small_string_buffer[0];
	if (size > DDP_SMALL_STRING_LIMIT) {
		ret->str = string = DDP_ALLOCATE(char, size);
		ret->refc = NULL;
	}

	// copy the passed char array
	memcpy(string, str, size);
}

// free a ddpstring
void ddp_free_string(ddpstring *str) {
	DDP_DBGLOG("free_string: %p", str);

	if (DDP_IS_SMALL_STRING(str)) {
		DDP_DBGLOG("not freeing small string");
		return;
	}

	if (str->refc == NULL) {
		DDP_DBGLOG("freeing str only");
		DDP_FREE_ARRAY(char, str->str, str->cap); // free the character array
		return;
	}

	DDP_DBGLOG("decrementing refc");
	if (--(*str->refc) == 0) {
		DDP_DBGLOG("freeing str and refc: " DDP_INT_FMT, *str->refc);
		DDP_FREE(ddpint, str->refc);
		DDP_DBGLOG("freed refc, freeing str");
		DDP_FREE_ARRAY(char, str->str, str->cap); // free the character array
	} else {
		DDP_DBGLOG("refc now at " DDP_INT_FMT, *str->refc);
	}
}

// allocate a new ddpstring as copy of str
void ddp_deep_copy_string(ddpstring *ret, ddpstring *str) {
	DDP_DBGLOG("ddp_deep_copy_string: %p, ret: %p, %s (%lld: %p)", str, ret, DDP_GET_STRING_PTR(str), str->cap, DDP_GET_STRING_PTR(str));
	if (ret == str) {
		return;
	}

	ret->cap = str->cap;

	char *string = &ret->small_string_buffer[0];
	if (!DDP_IS_SMALL_STRING(str)) {
		string = DDP_ALLOCATE(char, str->cap);
		ret->refc = NULL;
		ret->str = string;
	}

	// copy the passed char array
	memcpy(string, DDP_GET_STRING_PTR(str), str->cap);
}

void ddp_shallow_copy_string(ddpstring *ret, ddpstring *str) {
	DDP_DBGLOG("ddp_shallow_copy_string: %p, ret: %p", str, ret);
	if (ret == str) {
		return;
	}

	// for small strings shallow_copy == deep_copy
	if (DDP_IS_SMALL_STRING(str)) {
		ddp_deep_copy_string(ret, str);
		return;
	}

	if (str->refc == NULL) {
		DDP_DBGLOG("allocating refc");
		str->refc = DDP_ALLOCATE(ddpint, 1);
		DDP_DBGLOG("allocated refc: %p", str->refc);
		*str->refc = 1;
	}

	DDP_DBGLOG("incrementing refc: %p", str->refc);
	*str->refc += 1;
	DDP_DBGLOG("refc now at " DDP_INT_FMT, *str->refc);

	// set the fields of the copy
	*ret = *str;
}

// copies a string into itself
void ddp_perform_cow_string(ddpstring *str) {
	DDP_DBGLOG("ddp_perform_cow_string: %p", str);
	if (DDP_IS_SMALL_STRING(str)) {
		DDP_DBGLOG("not cowing small string");
		return;
	}

	if (str->refc == NULL) {
		return;
	}

	if (*str->refc == 1) {
		DDP_FREE(ddpint, str->refc);
		str->refc = NULL;
		return;
	}

	char *cpy = DDP_ALLOCATE(char, str->cap); // allocate the char array for the copy
	memcpy(cpy, str->str, str->cap);		  // copy the chars
	// set the fields of the copy
	*str->refc -= 1;
	str->refc = NULL;
	str->str = cpy;
}

// returns wether the length of str is 0
ddpbool ddp_string_empty(ddpstring *str) {
	return str->cap <= 0 || DDP_GET_STRING_PTR(str) == NULL || DDP_GET_STRING_PTR(str)[0] == '\0';
}

ddpint ddp_strlen(ddpstring *str) {
	if (ddp_string_empty(str)) {
		return 0;
	}
	return strlen(DDP_GET_STRING_PTR(str));
}

// allocates/frees/copies space if needed so that str has a capacity of at least size
void ddp_ensure_string_size(ddpstring *str, ddpint size) {
	ddp_perform_cow_string(str);
	if (size <= DDP_SMALL_STRING_LIMIT) {
		if (!DDP_IS_SMALL_STRING(str)) {
			char *old_str = str->str;
			ddpint old_cap = str->cap;
			str->cap = size;
			memcpy(DDP_GET_STRING_PTR(str), old_str, str->cap);
			DDP_FREE_ARRAY(char, old_str, old_cap);
			DDP_GET_STRING_PTR(str)
			[str->cap - 1] = '\0';
			DDP_DBGLOG("ensured shrank string");
			return;
		}

		str->cap = size;
		DDP_GET_STRING_PTR(str)
		[size - 1] = '\0';
		DDP_DBGLOG("ensured set new cap for already small string");
		return;
	}

	if (DDP_IS_SMALL_STRING(str)) {
		char *cpy = DDP_ALLOCATE(char, size);
		memcpy(cpy, DDP_GET_STRING_PTR(str), str->cap);
		str->str = cpy;
		str->refc = NULL;
		str->cap = size;
		DDP_DBGLOG("ensured allocated new space");
	} else {
		str->str = ddp_reallocate(str->str, str->cap, size);
		str->cap = size;
		str->str[str->cap - 1] = size;
		DDP_DBGLOG("ensured grew allocated space");
	}
}

static bool is_primitive_vtable(vtable *table) {
	DDP_DBGLOG("is_primitive: %lld, %p, %p, %p", table->type_size, table->free_func, table->deep_copy_func, table->equal_func);
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
	*ret = DDP_EMPTY_ANY;
	// copy metadata
	ret->vtable_ptr = any->vtable_ptr;

	if (any->vtable_ptr == NULL) {
		DDP_DBGLOG("vtable_ptr == NULL");
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
		DDP_DBGLOG("overwriting any value");
		memcpy(&ret->value, &any->value, DDP_SMALL_ANY_BUFF_SIZE);
	} else if (ret->vtable_ptr != NULL) {
		DDP_DBGLOG("shallow copying any value");
		// shallow copy the underlying value
		ret->vtable_ptr->shallow_copy_func(DDP_ANY_VALUE_PTR(ret), DDP_ANY_VALUE_PTR(any));
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
