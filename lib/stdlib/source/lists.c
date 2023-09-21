#include "ddpmemory.h"
#include "ddptypes.h"
#include <string.h>
#include <math.h>

typedef struct {
	void* arr;
	ddpint len;
	ddpint cap;
} generic_list;

typedef generic_list* generic_list_ref;
typedef void* generic_ref;

static void grow_if_needed(generic_list_ref list, ddpint elem_size) {
	if (list->len == list->cap) {
		ddpint old_cap = list->cap;
		list->cap = GROW_CAPACITY(list->cap);
		list->arr = ddp_reallocate(list->arr, old_cap * elem_size, list->cap * elem_size);
	}
}

static void efficient_list_append(generic_list_ref list, generic_ref elem, ddpint elem_size) {
	grow_if_needed(list, elem_size);
	memcpy(&((uint8_t*)list->arr)[list->len * elem_size], elem, elem_size);
	list->len++;
}

void efficient_list_append_int(ddpintlistref list, ddpintref elem, ddpint elem_size) {
	efficient_list_append((generic_list_ref)list, (generic_ref)elem, elem_size);
}

void efficient_list_append_float(ddpfloatlistref list, ddpfloatref elem, ddpint elem_size) {
	efficient_list_append((generic_list_ref)list, (generic_ref)elem, elem_size);
}

void efficient_list_append_bool(ddpboollistref list, ddpboolref elem, ddpint elem_size) {
	efficient_list_append((generic_list_ref)list, (generic_ref)elem, elem_size);
}

void efficient_list_append_char(ddpcharlistref list, ddpcharref elem, ddpint elem_size) {
	efficient_list_append((generic_list_ref)list, (generic_ref)elem, elem_size);
}

void efficient_list_append_string(ddpstringlistref list, ddpstringref elem, ddpint elem_size) {
	efficient_list_append((generic_list_ref)list, (generic_ref)elem, elem_size);
	elem->cap = 0;
	elem->str = NULL;
}

static void efficient_list_prepend(generic_list_ref list, generic_ref elem, ddpint elem_size) {
	grow_if_needed(list, elem_size);
	memmove(&((uint8_t*)list->arr)[elem_size], list->arr, list->len * elem_size);
	memcpy(list->arr, elem, elem_size);
	list->len++;
}

void efficient_list_prepend_int(ddpintlistref list, ddpintref elem, ddpint elem_size) {
	efficient_list_prepend((generic_list_ref)list, (generic_ref)elem, elem_size);
}

void efficient_list_prepend_float(ddpfloatlistref list, ddpfloatref elem, ddpint elem_size) {
	efficient_list_prepend((generic_list_ref)list, (generic_ref)elem, elem_size);
}

void efficient_list_prepend_bool(ddpboollistref list, ddpboolref elem, ddpint elem_size) {
	efficient_list_prepend((generic_list_ref)list, (generic_ref)elem, elem_size);
}

void efficient_list_prepend_char(ddpcharlistref list, ddpcharref elem, ddpint elem_size) {
	efficient_list_prepend((generic_list_ref)list, (generic_ref)elem, elem_size);
}

void efficient_list_prepend_string(ddpstringlistref list, ddpstringref elem, ddpint elem_size) {
	efficient_list_prepend((generic_list_ref)list, (generic_ref)elem, elem_size);
	elem->cap = 0;
	elem->str = NULL;
}

#define CLAMP(index, len) ((index) < 0 ? 0 : ((index) >= (len) ? (len) - 1 : (index)))

// the range is inclusive [start, end]
// the indices are 0-based (like in C, not like in DDP)
static void efficient_list_delete_range(generic_list_ref list, ddpint start, ddpint end, ddpint elem_size) {
	if (list->len <= 0) {
		return;
	}

	if (start > end) {
		ddp_runtime_error(1, "start index ist größer als end index (%d, %d)", start, end);
	}
	start = CLAMP(start, list->len);
	end = CLAMP(end, list->len);

	ddpint new_len = list->len - (end - start + 1);
	memmove(&((uint8_t*)list->arr)[start * elem_size], &((uint8_t*)list->arr)[(end + 1) * elem_size], (list->len - end - 1) * elem_size);
	list->len = new_len;
}

void efficient_list_delete_range_int(ddpintlistref list, ddpint start, ddpint end, ddpint elem_size) {
	efficient_list_delete_range((generic_list_ref)list, start, end, elem_size);
}

void efficient_list_delete_range_float(ddpfloatlistref list, ddpint start, ddpint end, ddpint elem_size) {
	efficient_list_delete_range((generic_list_ref)list, start, end, elem_size);
}

void efficient_list_delete_range_bool(ddpboollistref list, ddpint start, ddpint end, ddpint elem_size) {
	efficient_list_delete_range((generic_list_ref)list, start, end, elem_size);
}

void efficient_list_delete_range_char(ddpcharlistref list, ddpint start, ddpint end, ddpint elem_size) {
	efficient_list_delete_range((generic_list_ref)list, start, end, elem_size);
}

void efficient_list_delete_range_string(ddpstringlistref list, ddpint start, ddpint end, ddpint elem_size) {
	// duplicate logic, but better safe than sorry
	if (list->len <= 0) {
		return;
	}

	if (start > end) {
		ddp_runtime_error(1, "start index ist größer als end index (%d, %d)", start, end);
	}

	// free the old strings before shallow-copying them
	ddpint free_end = CLAMP(end, list->len);
	for (ddpint i = CLAMP(start, list->len); i <= free_end; i++) {
		ddp_free_string(&list->arr[i]);
	}

	efficient_list_delete_range((generic_list_ref)list, start, end, elem_size);
}