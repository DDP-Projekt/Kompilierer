#include "DDP/ddpmemory.h"
#include "DDP/ddptypes.h"
#include "DDP/utf8/utf8.h"
#include <string.h>

static void grow_if_needed(ddpgenericlistref list, ddpint elem_size) {
	if (list->len == list->cap) {
		ddpint old_cap = list->cap;
		list->cap = DDP_GROW_CAPACITY(list->cap);
		list->arr = ddp_reallocate(list->arr, old_cap * elem_size, list->cap * elem_size);
	}
}

static void claim_non_primitive(const ddpvtable *vtable, ddpgenericref elem, ddpanyref any) {
	if (vtable->deep_copy_func != NULL) {
		vtable->deep_copy_func(elem, any->vtable_ptr != NULL ? DDP_ANY_VALUE_PTR(any) : any);
	}
}

void efficient_list_append(ddpgenericlistref list, ddpgenericref elem, ddpanyref any) {
	const ddpvtable *vtable = ddp_get_generic_vtable(any);

	grow_if_needed(list, vtable->type_size);
	memcpy(&((uint8_t *)list->arr)[list->len * vtable->type_size], elem, vtable->type_size);
	list->len++;

	claim_non_primitive(vtable, elem, any);
}

void efficient_list_prepend(ddpgenericlistref list, ddpgenericref elem, ddpanyref any) {
	const ddpvtable *vtable = ddp_get_generic_vtable(any);

	grow_if_needed(list, vtable->type_size);
	memmove(&((uint8_t *)list->arr)[vtable->type_size], list->arr, list->len * vtable->type_size);
	memcpy(list->arr, elem, vtable->type_size);
	list->len++;

	claim_non_primitive(vtable, elem, any);
}

#define CLAMP(index, len) ((index) < 0 ? 0 : ((index) >= (len) ? (len)-1 : (index)))

// the range is inclusive [start, end]
// the indices are 0-based (like in C, not like in DDP)
void efficient_list_delete_range(ddpgenericlistref list, ddpint start, ddpint end, ddpanyref any) {
	if (list->len <= 0) {
		return;
	}

	if (start > end) {
		ddp_runtime_error(1, "start index ist größer als end index (" DDP_INT_FMT ", " DDP_INT_FMT ")", start, end);
	}

	const ddpvtable *vtable = ddp_get_generic_vtable(any);

	start = CLAMP(start, list->len);
	end = CLAMP(end, list->len);

	if (vtable->free_func != NULL) {
		// free the old non-primitives before shallow-copying them
		for (ddpint i = start; i <= end; i++) {
			vtable->free_func(&(((uint8_t *)list->arr)[i * vtable->type_size]));
		}
	}

	ddpint new_len = list->len - (end - start + 1);
	memmove(&((uint8_t *)list->arr)[start * vtable->type_size], &((uint8_t *)list->arr)[(end + 1) * vtable->type_size], (list->len - end - 1) * vtable->type_size);
	list->len = new_len;
}

// the index is 0-based (like in C, not like in DDP)
void efficient_list_insert(ddpgenericlistref list, ddpint index, ddpgenericref elem, ddpanyref any) {
	if (index < 0 || index > list->len) {
		ddp_runtime_error(1, "Index außerhalb der Listen Länge (Index war " DDP_INT_FMT ", Listen Länge war " DDP_INT_FMT ")", index, list->len);
	}

	const ddpvtable *vtable = ddp_get_generic_vtable(any);

	grow_if_needed(list, vtable->type_size);
	memmove(&((uint8_t *)list->arr)[(index + 1) * vtable->type_size], &((uint8_t *)list->arr)[index * vtable->type_size], (list->len - index) * vtable->type_size);
	memcpy(&((uint8_t *)list->arr)[index * vtable->type_size], elem, vtable->type_size);
	list->len++;

	claim_non_primitive(vtable, elem, any);
}

// the index is 0-based (like in C, not like in DDP)
void efficient_list_insert_range(ddpgenericlistref list, ddpint index, ddpgenericlistref other, ddpanyref any) {
	if (index < 0 || index > list->len) {
		ddp_runtime_error(1, "Index außerhalb der Listen Länge (Index war " DDP_INT_FMT ", Listen Länge war " DDP_INT_FMT ")", index, list->len);
	}

	const ddpvtable *vtable = ddp_get_generic_vtable(any);

	ddpint new_len = list->len + other->len;
	if (new_len > list->cap) {
		ddpint old_cap = list->cap;
		list->cap = DDP_GROW_CAPACITY(new_len);
		list->arr = ddp_reallocate(list->arr, old_cap * vtable->type_size, list->cap * vtable->type_size);
	}

	memmove(&((uint8_t *)list->arr)[(index + other->len) * vtable->type_size], &((uint8_t *)list->arr)[index * vtable->type_size], (list->len - index) * vtable->type_size);
	memcpy(&((uint8_t *)list->arr)[index * vtable->type_size], other->arr, other->len * vtable->type_size);
	list->len = new_len;

	if (vtable->deep_copy_func != NULL) {
		// free the old non-primitives before shallow-copying them
		for (ddpint i = 0; i < other->len; i++) {
			claim_non_primitive(vtable, &(((uint8_t *)other->arr)[i * vtable->type_size]), any);
		}
	}
}

void Aneinandergehaengt_Buchstabe_Ref(ddpstring *ret, ddpcharlistref liste) {
	size_t num_bytes = 0;
	ddpchar *end = &liste->arr[liste->len];
	for (ddpchar *it = liste->arr; it != end; it++) {
		num_bytes += utf8_num_bytes_char(*it);
	}

	ret->str = DDP_ALLOCATE(char, num_bytes + 1);
	ret->cap = num_bytes + 1;

	char *str_it = ret->str;
	for (ddpchar *it = liste->arr; it != end; it++) {
		str_it += utf8_char_to_string(str_it, *it);
	}

	ret->str[num_bytes] = '\0';
}
