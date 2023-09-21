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