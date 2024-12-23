#include "DDP/ddpmemory.h"
#include "DDP/ddptypes.h"
#include "DDP/debug.h"
#include "DDP/utf8/utf8.h"
#include <string.h>

typedef struct {
	ddpint *refc;
	void *arr;
	ddpint len;
	ddpint cap;
} generic_list;

typedef generic_list *generic_list_ref;
typedef void *generic_ref;

static void grow_if_needed(generic_list_ref list, ddpint elem_size) {
	if (list->len == list->cap) {
		ddpint old_cap = list->cap;
		list->cap = DDP_GROW_CAPACITY(list->cap);
		list->arr = ddp_reallocate(list->arr, old_cap * elem_size, list->cap * elem_size);
	}
}

static void efficient_list_append(generic_list_ref list, generic_ref elem, ddpint elem_size) {
	grow_if_needed(list, elem_size);
	memcpy(&((uint8_t *)list->arr)[list->len * elem_size], elem, elem_size);
	list->len++;
}

void efficient_list_append_int(ddpintlistref list, ddpintref elem, ddpint elem_size) {
	ddp_perform_cow_ddpintlist(list);
	efficient_list_append((generic_list_ref)list, (generic_ref)elem, elem_size);
}

void efficient_list_append_float(ddpfloatlistref list, ddpfloatref elem, ddpint elem_size) {
	ddp_perform_cow_ddpfloatlist(list);
	efficient_list_append((generic_list_ref)list, (generic_ref)elem, elem_size);
}

void efficient_list_append_bool(ddpboollistref list, ddpboolref elem, ddpint elem_size) {
	ddp_perform_cow_ddpboollist(list);
	efficient_list_append((generic_list_ref)list, (generic_ref)elem, elem_size);
}

void efficient_list_append_char(ddpcharlistref list, ddpcharref elem, ddpint elem_size) {
	ddp_perform_cow_ddpcharlist(list);
	efficient_list_append((generic_list_ref)list, (generic_ref)elem, elem_size);
}

void efficient_list_append_string(ddpstringlistref list, ddpstringref elem, ddpint elem_size) {
	ddp_perform_cow_ddpstringlist(list);
	efficient_list_append((generic_list_ref)list, (generic_ref)elem, elem_size);
	*elem = DDP_EMPTY_STRING;
}

void efficient_list_append_any(ddpanylistref list, ddpanyref elem, ddpint elem_size) {
	ddp_perform_cow_ddpanylist(list);
	efficient_list_append((generic_list_ref)list, (generic_ref)elem, elem_size);
	*elem = DDP_EMPTY_ANY;
}

static void efficient_list_prepend(generic_list_ref list, generic_ref elem, ddpint elem_size) {
	grow_if_needed(list, elem_size);
	memmove(&((uint8_t *)list->arr)[elem_size], list->arr, list->len * elem_size);
	memcpy(list->arr, elem, elem_size);
	list->len++;
}

void efficient_list_prepend_int(ddpintlistref list, ddpintref elem, ddpint elem_size) {
	ddp_perform_cow_ddpintlist(list);
	efficient_list_prepend((generic_list_ref)list, (generic_ref)elem, elem_size);
}

void efficient_list_prepend_float(ddpfloatlistref list, ddpfloatref elem, ddpint elem_size) {
	ddp_perform_cow_ddpfloatlist(list);
	efficient_list_prepend((generic_list_ref)list, (generic_ref)elem, elem_size);
}

void efficient_list_prepend_bool(ddpboollistref list, ddpboolref elem, ddpint elem_size) {
	ddp_perform_cow_ddpboollist(list);
	efficient_list_prepend((generic_list_ref)list, (generic_ref)elem, elem_size);
}

void efficient_list_prepend_char(ddpcharlistref list, ddpcharref elem, ddpint elem_size) {
	ddp_perform_cow_ddpcharlist(list);
	efficient_list_prepend((generic_list_ref)list, (generic_ref)elem, elem_size);
}

void efficient_list_prepend_string(ddpstringlistref list, ddpstringref elem, ddpint elem_size) {
	ddp_perform_cow_ddpstringlist(list);
	efficient_list_prepend((generic_list_ref)list, (generic_ref)elem, elem_size);
	*elem = DDP_EMPTY_STRING;
}

void efficient_list_prepend_any(ddpanylistref list, ddpanyref elem, ddpint elem_size) {
	ddp_perform_cow_ddpanylist(list);
	efficient_list_prepend((generic_list_ref)list, (generic_ref)elem, elem_size);
	*elem = DDP_EMPTY_ANY;
}

#define CLAMP(index, len) ((index) < 0 ? 0 : ((index) >= (len) ? (len) - 1 : (index)))

// the range is inclusive [start, end]
// the indices are 0-based (like in C, not like in DDP)
static void efficient_list_delete_range(generic_list_ref list, ddpint start, ddpint end, ddpint elem_size) {
	if (list->len <= 0) {
		return;
	}

	if (start > end) {
		ddp_runtime_error(1, "start index ist größer als end index (" DDP_INT_FMT ", " DDP_INT_FMT ")", start, end);
	}
	start = CLAMP(start, list->len);
	end = CLAMP(end, list->len);

	ddpint new_len = list->len - (end - start + 1);
	memmove(&((uint8_t *)list->arr)[start * elem_size], &((uint8_t *)list->arr)[(end + 1) * elem_size], (list->len - end - 1) * elem_size);
	list->len = new_len;
}

void efficient_list_delete_range_int(ddpintlistref list, ddpint start, ddpint end, ddpint elem_size) {
	ddp_perform_cow_ddpintlist(list);
	efficient_list_delete_range((generic_list_ref)list, start, end, elem_size);
}

void efficient_list_delete_range_float(ddpfloatlistref list, ddpint start, ddpint end, ddpint elem_size) {
	ddp_perform_cow_ddpfloatlist(list);
	efficient_list_delete_range((generic_list_ref)list, start, end, elem_size);
}

void efficient_list_delete_range_bool(ddpboollistref list, ddpint start, ddpint end, ddpint elem_size) {
	ddp_perform_cow_ddpboollist(list);
	efficient_list_delete_range((generic_list_ref)list, start, end, elem_size);
}

void efficient_list_delete_range_char(ddpcharlistref list, ddpint start, ddpint end, ddpint elem_size) {
	ddp_perform_cow_ddpcharlist(list);
	efficient_list_delete_range((generic_list_ref)list, start, end, elem_size);
}

void efficient_list_delete_range_string(ddpstringlistref list, ddpint start, ddpint end, ddpint elem_size) {
	ddp_perform_cow_ddpstringlist(list);

	// duplicate logic, but better safe than sorry
	if (list->len <= 0) {
		return;
	}

	if (start > end) {
		ddp_runtime_error(1, "start index ist größer als end index (" DDP_INT_FMT ", " DDP_INT_FMT ")", start, end);
	}

	// free the old strings before shallow-copying them
	ddpint free_end = CLAMP(end, list->len);
	for (ddpint i = CLAMP(start, list->len); i <= free_end; i++) {
		ddp_free_string(&list->arr[i]);
	}

	efficient_list_delete_range((generic_list_ref)list, start, end, elem_size);
}

void efficient_list_delete_range_any(ddpanylistref list, ddpint start, ddpint end, ddpint elem_size) {
	ddp_perform_cow_ddpanylist(list);

	// duplicate logic, but better safe than sorry
	if (list->len <= 0) {
		return;
	}

	if (start > end) {
		ddp_runtime_error(1, "start index ist größer als end index (" DDP_INT_FMT ", " DDP_INT_FMT ")", start, end);
	}

	// free the old strings before shallow-copying them
	ddpint free_end = CLAMP(end, list->len);
	for (ddpint i = CLAMP(start, list->len); i <= free_end; i++) {
		ddp_free_any(&list->arr[i]);
	}

	efficient_list_delete_range((generic_list_ref)list, start, end, elem_size);
}

// the index is 0-based (like in C, not like in DDP)
void efficient_list_insert(generic_list_ref list, ddpint index, generic_ref elem, ddpint elem_size) {
	if (index < 0 || index > list->len) {
		ddp_runtime_error(1, "Index außerhalb der Listen Länge (Index war " DDP_INT_FMT ", Listen Länge war " DDP_INT_FMT ")", index, list->len);
	}

	grow_if_needed(list, elem_size);
	memmove(&((uint8_t *)list->arr)[(index + 1) * elem_size], &((uint8_t *)list->arr)[index * elem_size], (list->len - index) * elem_size);
	memcpy(&((uint8_t *)list->arr)[index * elem_size], elem, elem_size);
	list->len++;
}

void efficient_list_insert_int(ddpintlistref list, ddpint index, ddpintref elem, ddpint elem_size) {
	ddp_perform_cow_ddpintlist(list);
	efficient_list_insert((generic_list_ref)list, index, (generic_ref)elem, elem_size);
}

void efficient_list_insert_float(ddpfloatlistref list, ddpint index, ddpfloatref elem, ddpint elem_size) {
	ddp_perform_cow_ddpfloatlist(list);
	efficient_list_insert((generic_list_ref)list, index, (generic_ref)elem, elem_size);
}

void efficient_list_insert_bool(ddpboollistref list, ddpint index, ddpboolref elem, ddpint elem_size) {
	ddp_perform_cow_ddpboollist(list);
	efficient_list_insert((generic_list_ref)list, index, (generic_ref)elem, elem_size);
}

void efficient_list_insert_char(ddpcharlistref list, ddpint index, ddpcharref elem, ddpint elem_size) {
	ddp_perform_cow_ddpcharlist(list);
	efficient_list_insert((generic_list_ref)list, index, (generic_ref)elem, elem_size);
}

void efficient_list_insert_string(ddpstringlistref list, ddpint index, ddpstringref elem, ddpint elem_size) {
	ddp_perform_cow_ddpstringlist(list);
	efficient_list_insert((generic_list_ref)list, index, (generic_ref)elem, elem_size);
	*elem = DDP_EMPTY_STRING;
}

void efficient_list_insert_any(ddpanylistref list, ddpint index, ddpanyref elem, ddpint elem_size) {
	ddp_perform_cow_ddpanylist(list);
	efficient_list_insert((generic_list_ref)list, index, (generic_ref)elem, elem_size);
	*elem = DDP_EMPTY_ANY;
}

// the index is 0-based (like in C, not like in DDP)
void efficient_list_insert_range(generic_list_ref list, ddpint index, generic_list_ref other, ddpint elem_size) {
	if (index < 0 || index > list->len) {
		ddp_runtime_error(1, "Index außerhalb der Listen Länge (Index war " DDP_INT_FMT ", Listen Länge war " DDP_INT_FMT ")", index, list->len);
	}

	ddpint new_len = list->len + other->len;
	if (new_len > list->cap) {
		DDP_DBGLOG("increasing capacity");
		ddpint old_cap = list->cap;
		list->cap = DDP_GROW_CAPACITY(new_len);
		list->arr = ddp_reallocate(list->arr, old_cap * elem_size, list->cap * elem_size);
	}

	memmove(&((uint8_t *)list->arr)[(index + other->len) * elem_size], &((uint8_t *)list->arr)[index * elem_size], (list->len - index) * elem_size);
	memcpy(&((uint8_t *)list->arr)[index * elem_size], other->arr, other->len * elem_size);
	list->len = new_len;
}

void efficient_list_insert_range_int(ddpintlistref list, ddpint index, ddpintlistref other, ddpint elem_size) {
	DDP_DBGLOG("efficient_list_insert_range_int: %p, %p, %p, %p, %lld, %lld", list, other, list->arr, other->arr, list->cap, other->cap);
	ddp_perform_cow_ddpintlist(list);
	DDP_DBGLOG("after cow: %p, %p, %p, %p, %lld, %lld", list, other, list->arr, other->arr, list->cap, other->cap);
	efficient_list_insert_range((generic_list_ref)list, index, (generic_list_ref)other, elem_size);
	DDP_DBGLOG("after insert: %p, %p, %p, %p, %lld, %lld", list, other, list->arr, other->arr, list->cap, other->cap);
}

void efficient_list_insert_range_float(ddpfloatlistref list, ddpint index, ddpfloatlistref other, ddpint elem_size) {
	ddp_perform_cow_ddpfloatlist(list);
	efficient_list_insert_range((generic_list_ref)list, index, (generic_list_ref)other, elem_size);
}

void efficient_list_insert_range_bool(ddpboollistref list, ddpint index, ddpboollistref other, ddpint elem_size) {
	ddp_perform_cow_ddpboollist(list);
	efficient_list_insert_range((generic_list_ref)list, index, (generic_list_ref)other, elem_size);
}

void efficient_list_insert_range_char(ddpcharlistref list, ddpint index, ddpcharlistref other, ddpint elem_size) {
	ddp_perform_cow_ddpcharlist(list);
	efficient_list_insert_range((generic_list_ref)list, index, (generic_list_ref)other, elem_size);
}

void efficient_list_insert_range_string(ddpstringlistref list, ddpint index, ddpstringlistref other, ddpint elem_size) {
	ddp_perform_cow_ddpstringlist(list);
	efficient_list_insert_range((generic_list_ref)list, index, (generic_list_ref)other, elem_size);
	for (ddpint i = 0; i < other->len; i++) {
		other->arr[i] = DDP_EMPTY_STRING;
	}
}

void efficient_list_insert_range_any(ddpanylistref list, ddpint index, ddpanylistref other, ddpint elem_size) {
	ddp_perform_cow_ddpanylist(list);
	efficient_list_insert_range((generic_list_ref)list, index, (generic_list_ref)other, elem_size);
	for (ddpint i = 0; i < other->len; i++) {
		other->arr[i] = DDP_EMPTY_ANY;
	}
}

void Aneinandergehaengt_Buchstabe_Ref(ddpstring *ret, ddpcharlistref liste) {
	*ret = DDP_EMPTY_STRING;
	size_t num_bytes = 0;
	ddpchar *end = &liste->arr[liste->len];
	for (ddpchar *it = liste->arr; it != end; it++) {
		num_bytes += utf8_num_bytes_char(*it);
	}

	ret->cap = num_bytes + 1;
	if (!DDP_IS_SMALL_STRING(ret)) {
		ret->str = DDP_ALLOCATE(char, num_bytes + 1);
		ret->refc = NULL;
	}
	char *ret_ptr = DDP_GET_STRING_PTR(ret);

	char *str_it = ret_ptr;
	for (ddpchar *it = liste->arr; it != end; it++) {
		str_it += utf8_char_to_string(str_it, *it);
	}

	ret_ptr[num_bytes] = '\0';
}
