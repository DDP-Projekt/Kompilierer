#include "ddptypes.h"
#include "memory.h"
#include "debug.h"
#include "gc.h"
#include "utf8/utf8.h"
#include <stdarg.h>

// allocate and create a ddpstring from a constant char array
// str must be null-terminated
ddpstring* inbuilt_string_from_constant(char* str) {
	ddpstring* dstr = ALLOCATE(ddpstring, 1); // up here to log the adress in debug mode
	DBGLOG("inbuilt_string_from_constant: %p", dstr);
	size_t size = strlen(str) + 1;
	char* string = ALLOCATE(char, size); // the char array of the string (plus null terminator)
	// copy the passed char array
	memcpy(string, str, size);

	// set the string fields
	dstr->str = string;
	dstr->cap = size;
	return dstr;
}

// free a ddpstring
void free_string(ddpstring* str) {
	DBGLOG("free_string: %p", str);
	FREE_ARRAY(char, str->str, str->cap); // free the character array
	FREE(ddpstring, str); // free the string pointer
}

// allocate a new ddpstring as copy of str
ddpstring* inbuilt_deep_copy_string(ddpstring* str) {
	DBGLOG("inbuilt_deep_copy_string: %p", str);
	char* cpy = ALLOCATE(char, str->cap); // allocate the char array for the copy
	memcpy(cpy, str->str, str->cap); // copy the chars
	ddpstring* cpystr = ALLOCATE(ddpstring, 1); // alocate the copy string
	// set the fields of the copy
	cpystr->str = cpy;
	cpystr->cap = str->cap;
	return cpystr;
}

/***** Partially generated code *****/

ddpintlist* inbuilt_ddpintlist_from_constants(ddpint count, ...) {
	ddpintlist* list = ALLOCATE(ddpintlist, 1); // up here to log the adress in debug mode
	DBGLOG("ddpintlist_from_constants: %p", list);
	if (count > 0) {
		ddpint* arr = ALLOCATE(ddpint, count); // the element array of the list
		
		va_list elements;
		va_start(elements, count);

		for (size_t i = 0; i < count; i++) {
			arr[i] = va_arg(elements, ddpint);
			
		}

		va_end(elements);

		list->arr = arr;
	} else {
		list->arr = NULL;
	}
	list->len = count;
	list->cap = count;
	return list;
}

void free_ddpintlist(ddpintlist* list) {
	DBGLOG("free_ddpintlist: %p", list);
	
	FREE_ARRAY(ddpint, list->arr, list->cap); // free the element array
	FREE(ddpintlist, list); // free the list pointer
}

ddpintlist* inbuilt_deep_copy_ddpintlist(ddpintlist* list) {
	DBGLOG("inbuilt_deep_copy_ddpintlist: %p", list);
	ddpint* cpy = ALLOCATE(ddpint, list->cap); // allocate the element array for the copy
	memcpy(cpy, list->arr, sizeof(ddpint) * list->cap); // copy the chars
	
	ddpintlist* cpylist = ALLOCATE(ddpintlist, 1); // alocate the copy list
	// set the fields of the copy
	cpylist->arr = cpy;
	cpylist->len = list->len;
	cpylist->cap = list->cap;
	return cpylist;
}

ddpfloatlist* inbuilt_ddpfloatlist_from_constants(ddpint count, ...) {
	ddpfloatlist* list = ALLOCATE(ddpfloatlist, 1); // up here to log the adress in debug mode
	DBGLOG("ddpfloatlist_from_constants: %p", list);
	if (count > 0) {
		ddpfloat* arr = ALLOCATE(ddpfloat, count); // the element array of the list
		
		va_list elements;
		va_start(elements, count);

		for (size_t i = 0; i < count; i++) {
			arr[i] = va_arg(elements, ddpfloat);
			
		}

		va_end(elements);

		list->arr = arr;
	} else {
		list->arr = NULL;
	}
	list->len = count;
	list->cap = count;
	return list;
}

void free_ddpfloatlist(ddpfloatlist* list) {
	DBGLOG("free_ddpfloatlist: %p", list);
	
	FREE_ARRAY(ddpfloat, list->arr, list->cap); // free the element array
	FREE(ddpfloatlist, list); // free the list pointer
}

ddpfloatlist* inbuilt_deep_copy_ddpfloatlist(ddpfloatlist* list) {
	DBGLOG("inbuilt_deep_copy_ddpfloatlist: %p", list);
	ddpfloat* cpy = ALLOCATE(ddpfloat, list->cap); // allocate the element array for the copy
	memcpy(cpy, list->arr, sizeof(ddpfloat) * list->cap); // copy the chars
	
	ddpfloatlist* cpylist = ALLOCATE(ddpfloatlist, 1); // alocate the copy list
	// set the fields of the copy
	cpylist->arr = cpy;
	cpylist->len = list->len;
	cpylist->cap = list->cap;
	return cpylist;
}

ddpboollist* inbuilt_ddpboollist_from_constants(ddpint count, ...) {
	ddpboollist* list = ALLOCATE(ddpboollist, 1); // up here to log the adress in debug mode
	DBGLOG("ddpboollist_from_constants: %p", list);
	if (count > 0) {
		ddpbool* arr = ALLOCATE(ddpbool, count); // the element array of the list
		
		va_list elements;
		va_start(elements, count);

		for (size_t i = 0; i < count; i++) {
			arr[i] = (ddpbool)va_arg(elements, int);
			
		}

		va_end(elements);

		list->arr = arr;
	} else {
		list->arr = NULL;
	}
	list->len = count;
	list->cap = count;
	return list;
}

void free_ddpboollist(ddpboollist* list) {
	DBGLOG("free_ddpboollist: %p", list);
	
	FREE_ARRAY(ddpbool, list->arr, list->cap); // free the element array
	FREE(ddpboollist, list); // free the list pointer
}

ddpboollist* inbuilt_deep_copy_ddpboollist(ddpboollist* list) {
	DBGLOG("inbuilt_deep_copy_ddpboollist: %p", list);
	ddpbool* cpy = ALLOCATE(ddpbool, list->cap); // allocate the element array for the copy
	memcpy(cpy, list->arr, sizeof(ddpbool) * list->cap); // copy the chars
	
	ddpboollist* cpylist = ALLOCATE(ddpboollist, 1); // alocate the copy list
	// set the fields of the copy
	cpylist->arr = cpy;
	cpylist->len = list->len;
	cpylist->cap = list->cap;
	return cpylist;
}

ddpcharlist* inbuilt_ddpcharlist_from_constants(ddpint count, ...) {
	ddpcharlist* list = ALLOCATE(ddpcharlist, 1); // up here to log the adress in debug mode
	DBGLOG("ddpcharlist_from_constants: %p", list);
	if (count > 0) {
		ddpchar* arr = ALLOCATE(ddpchar, count); // the element array of the list
		
		va_list elements;
		va_start(elements, count);

		for (size_t i = 0; i < count; i++) {
			arr[i] = va_arg(elements, ddpchar);
			
		}

		va_end(elements);

		list->arr = arr;
	} else {
		list->arr = NULL;
	}
	list->len = count;
	list->cap = count;
	return list;
}

void free_ddpcharlist(ddpcharlist* list) {
	DBGLOG("free_ddpcharlist: %p", list);
	
	FREE_ARRAY(ddpchar, list->arr, list->cap); // free the element array
	FREE(ddpcharlist, list); // free the list pointer
}

ddpcharlist* inbuilt_deep_copy_ddpcharlist(ddpcharlist* list) {
	DBGLOG("inbuilt_deep_copy_ddpcharlist: %p", list);
	ddpchar* cpy = ALLOCATE(ddpchar, list->cap); // allocate the element array for the copy
	memcpy(cpy, list->arr, sizeof(ddpchar) * list->cap); // copy the chars
	
	ddpcharlist* cpylist = ALLOCATE(ddpcharlist, 1); // alocate the copy list
	// set the fields of the copy
	cpylist->arr = cpy;
	cpylist->len = list->len;
	cpylist->cap = list->cap;
	return cpylist;
}

ddpstringlist* inbuilt_ddpstringlist_from_constants(ddpint count, ...) {
	ddpstringlist* list = ALLOCATE(ddpstringlist, 1); // up here to log the adress in debug mode
	DBGLOG("ddpstringlist_from_constants: %p", list);
	if (count > 0) {
		ddpstring** arr = ALLOCATE(ddpstring*, count); // the element array of the list
		
		va_list elements;
		va_start(elements, count);

		for (size_t i = 0; i < count; i++) {
			arr[i] = va_arg(elements, ddpstring*);
			
			inbuilt_increment_ref_count(arr[i], VK_STRING);
			
		}

		va_end(elements);

		list->arr = arr;
	} else {
		list->arr = NULL;
	}
	list->len = count;
	list->cap = count;
	return list;
}

void free_ddpstringlist(ddpstringlist* list) {
	DBGLOG("free_ddpstringlist: %p", list);
	
	for (size_t i = 0; i < list->len; i++) {
		inbuilt_decrement_ref_count(list->arr[i]);
	}
	
	FREE_ARRAY(ddpstring*, list->arr, list->cap); // free the element array
	FREE(ddpstringlist, list); // free the list pointer
}

ddpstringlist* inbuilt_deep_copy_ddpstringlist(ddpstringlist* list) {
	DBGLOG("inbuilt_deep_copy_ddpstringlist: %p", list);
	ddpstring** cpy = ALLOCATE(ddpstring*, list->cap); // allocate the element array for the copy
	memcpy(cpy, list->arr, sizeof(ddpstring*) * list->cap); // copy the chars
	
	for (size_t i = 0; i < list->len; i++) {
		cpy[i] = inbuilt_deep_copy_string(cpy[i]);
		inbuilt_increment_ref_count(cpy[i], VK_STRING);
	}
	
	ddpstringlist* cpylist = ALLOCATE(ddpstringlist, 1); // alocate the copy list
	// set the fields of the copy
	cpylist->arr = cpy;
	cpylist->len = list->len;
	cpylist->cap = list->cap;
	return cpylist;
}
