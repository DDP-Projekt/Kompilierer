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

static void efficient_list_append(generic_list_ref list, generic_ref elem, ddpint elem_size) {
	if (list->len == list->cap) {
		ddpint old_cap = list->cap;
		list->cap = GROW_CAPACITY(list->cap);
		list->arr = ddp_reallocate(list->arr, old_cap, list->cap * elem_size);
	}
	memcpy(list->arr + list->len * elem_size, elem, elem_size);
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