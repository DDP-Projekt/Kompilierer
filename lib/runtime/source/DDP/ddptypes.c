#include "DDP/ddptypes.h"
#include "DDP/ddpmemory.h"
#include "DDP/debug.h"
#include <stdlib.h>
#include <string.h>

// allocate and create a ddpstring from a constant char array
// str must be null-terminated
void ddp_string_from_constant(ddpstring *ret, const char *str) {
	DDP_DBGLOG("_ddp_string_from_constant: ret: %p, str: %s", ret, str);
	const size_t size = strlen(str) + 1;
	if (size == 1) {
		*ret = DDP_EMPTY_STRING;
		return;
	}

	ret->len = size - 1;

	if (size <= DDP_SMALL_STRING_BUFF_SIZE) {
		memcpy(ret->small.str, str, size);
		return;
	}

	// allocate some more to prevent future allocations
	const size_t cap = DDP_MAX(size, DDP_SMALL_ALLOCATION_SIZE);
	char *string = DDP_ALLOCATE(char, cap); // the char array of the string (plus null terminator)
	// copy the passed char array
	memcpy(string, str, size);

	// set the string fields
	ret->large.str = string;
	ret->large.cap = cap;
}

// free a ddpstring
void ddp_free_string(const ddpstring *str) {
	DDP_DBGLOG("free_string: %p", str);
	if (DDP_IS_LARGE_STRING(str)) {
		DDP_DBGLOG("freeing large string: %p", str->large.str);
		DDP_FREE_ARRAY(char, str->large.str, str->large.cap);
	}
	DDP_DBGLOG("freed small string: %p", str);
}

// allocate a new ddpstring as copy of str
void ddp_deep_copy_string(ddpstring *ret, const ddpstring *str) {
	DDP_DBGLOG("_ddp_deep_copy_string: %p, ret: %p", str, ret);
	if (ret == str) {
		return;
	}

	if (DDP_IS_SMALL_STRING(str)) {
		*ret = *str;
		return;
	}

	// allocate some more to prevent future allocations
	const size_t cap = DDP_MAX(str->len + 1, DDP_SMALL_ALLOCATION_SIZE);
	char *cpy = DDP_ALLOCATE(char, cap);	   // allocate the char array for the copy
	memcpy(cpy, str->large.str, str->len + 1); // copy the chars + null-terminator
	// set the fields of the copy
	ret->len = str->len;
	ret->large.str = cpy;
	ret->large.cap = cap;
}

// returns wether the character length of str is 0
ddpbool ddp_string_empty(const ddpstring *str) {
	return str->len == 0;
}

// appends data to the given string and takes care of small vs large strings
void ddp_strncat(ddpstring *str, const char *data, size_t n) {
	const ddpint oldCap = DDP_STRING_CAP(str);
	const ddpint newLen = str->len + n;

	char *oldData = DDP_STRING_DATA(str);

	// no need to allocate
	if (newLen < oldCap) {
		memcpy(&oldData[str->len], data, n);
		str->len = newLen;
		oldData[str->len] = '\0';
		return;
	}

	const size_t newCap = DDP_MAX(newLen + 1, DDP_SMALL_ALLOCATION_SIZE);

	// small becomes large
	if (DDP_IS_SMALL_STRING(str)) {
		char *newData = DDP_ALLOCATE(char, newCap); // allocate the char array for the copy
		memcpy(newData, oldData, str->len);
		memcpy(&newData[str->len], data, n);
		str->len = newLen;
		newData[str->len] = '\0';
		str->large.str = newData;
		str->large.cap = newCap;
		return;
	}

	// large stays large
	str->large.str = ddp_reallocate(str->large.str, str->large.cap, newCap);
	str->large.cap = newCap;
	memcpy(&str->large.str[str->len], data, n);
	str->len = newLen;
	str->large.str[str->len] = '\0';
}

// ensures that the large string str has a capacity of at least newCap or more
void ddp_reserve_string_capacity(ddpstring *str, ddpint n) {
	if (n <= DDP_STRING_CAP(str) || DDP_IS_SMALL_STRING(str)) {
		return;
	}

	n = DDP_MAX(n, DDP_SMALL_ALLOCATION_SIZE);

	if (DDP_IS_SMALL_STRING(str)) {
		char *newStr = DDP_ALLOCATE(char, n);
		memcpy(newStr, str->small.str, str->len + 1);
		str->large.str = newStr;
		str->large.cap = n;
		return;
	}

	str->large.str = ddp_reallocate(str->large.str, str->large.cap, n);
	str->large.cap = n;
}

extern ddpvtable ddpint_vtable;
extern ddpvtable ddpfloat_vtable;
extern ddpvtable ddpbool_vtable;
extern ddpvtable ddpchar_vtable;

static bool is_primitive_vtable(ddpvtable *table) {
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

static ddpvtable any_vtable;

/*
 * Branch prediction tuning.
 * The expression is expected to be true (=likely) or false (=unlikely).
 */
#if defined(__GNUC__)
#define likely(x) __builtin_expect(!!(x), 1)
#define unlikely(x) __builtin_expect(!!(x), 0)
#else
#define likely(x) (x)
#define unlikely(x) (x)
#endif

// helper function for generic extern functions which may pass ddpany as a way to pass type information
const ddpvtable *ddp_get_generic_vtable(const ddpany *any) {
	if (unlikely(any_vtable.free_func == NULL)) {
		any_vtable = (ddpvtable){
			.type_size = sizeof(ddpany),
			.free_func = (free_func_ptr)ddp_free_any,
			.deep_copy_func = (deep_copy_func_ptr)ddp_deep_copy_any,
			.equal_func = (equal_func_ptr)ddp_any_equal,
		};
	}

	// the any is empty and therefore T itself is any
	if (any->vtable_ptr == NULL) {
		return &any_vtable;
	}

	return any->vtable_ptr;
}
