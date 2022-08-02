#include "ddptypes.h"
#include "memory.h"
#include "debug.h"
#include "hashtable.h"

extern ddpbool inbuilt_string_equal(ddpstring*, ddpstring*);
extern void inbuilt_increment_ref_count(void*, uint8_t);
extern ddpstring* inbuilt_deep_copy_string(ddpstring*);

static ddpint clamp(ddpint i, ddpint min, ddpint max) {
  const ddpint t = i < min ? min : i;
  return t > max ? max : t;
}


ddpbool inbuilt_ddpintlist_equal(ddpintlist* list1, ddpintlist* list2) {
	if (list1 == list2) return true;
	if (list1->len != list2->len) return false; // if the length is different, it's a quick false return
	
	return memcmp(list1->arr, list2->arr, sizeof(ddpint) * list1->len) == 0;
	
}

ddpintlist* inbuilt_ddpintlist_slice(ddpintlist* list, ddpint index1, ddpint index2) {
	ddpintlist* cpyList = ALLOCATE(ddpintlist, 1); // up here to log the adress in debug mode
	DBGLOG("inbuilt_ddpintlist_slice: %p", cpyList);

	if (list->len <= 0) {
		cpyList->arr = NULL;
		cpyList->len = 0;
		cpyList->cap = 0;
		return cpyList;
	}

	index1 = clamp(index1, 1, list->len);
	index2 = clamp(index2, 1, list->len);
	if (index2 < index1) {
		runtime_error(1, "Invalide Indexe (Index 1 war %ld, Index 2 war %ld)\n", index1, index2);
	}

	index1--,index2--; // ddp indices start at 1, c indices at 0

	size_t new_list_cap = (index2 - index1) + 1; // + 1 if indices are equal
	ddpint* arr = ALLOCATE(ddpint, new_list_cap);

	
	memcpy(arr, &list->arr[index1], sizeof(ddpint) * new_list_cap);
	

	cpyList->arr = arr;
	cpyList->len = new_list_cap;
	cpyList->cap = new_list_cap;
	return cpyList;
}

ddpbool inbuilt_ddpfloatlist_equal(ddpfloatlist* list1, ddpfloatlist* list2) {
	if (list1 == list2) return true;
	if (list1->len != list2->len) return false; // if the length is different, it's a quick false return
	
	return memcmp(list1->arr, list2->arr, sizeof(ddpfloat) * list1->len) == 0;
	
}

ddpfloatlist* inbuilt_ddpfloatlist_slice(ddpfloatlist* list, ddpint index1, ddpint index2) {
	ddpfloatlist* cpyList = ALLOCATE(ddpfloatlist, 1); // up here to log the adress in debug mode
	DBGLOG("inbuilt_ddpfloatlist_slice: %p", cpyList);

	if (list->len <= 0) {
		cpyList->arr = NULL;
		cpyList->len = 0;
		cpyList->cap = 0;
		return cpyList;
	}

	index1 = clamp(index1, 1, list->len);
	index2 = clamp(index2, 1, list->len);
	if (index2 < index1) {
		runtime_error(1, "Invalide Indexe (Index 1 war %ld, Index 2 war %ld)\n", index1, index2);
	}

	index1--,index2--; // ddp indices start at 1, c indices at 0

	size_t new_list_cap = (index2 - index1) + 1; // + 1 if indices are equal
	ddpfloat* arr = ALLOCATE(ddpfloat, new_list_cap);

	
	memcpy(arr, &list->arr[index1], sizeof(ddpfloat) * new_list_cap);
	

	cpyList->arr = arr;
	cpyList->len = new_list_cap;
	cpyList->cap = new_list_cap;
	return cpyList;
}

ddpbool inbuilt_ddpboollist_equal(ddpboollist* list1, ddpboollist* list2) {
	if (list1 == list2) return true;
	if (list1->len != list2->len) return false; // if the length is different, it's a quick false return
	
	return memcmp(list1->arr, list2->arr, sizeof(ddpbool) * list1->len) == 0;
	
}

ddpboollist* inbuilt_ddpboollist_slice(ddpboollist* list, ddpint index1, ddpint index2) {
	ddpboollist* cpyList = ALLOCATE(ddpboollist, 1); // up here to log the adress in debug mode
	DBGLOG("inbuilt_ddpboollist_slice: %p", cpyList);

	if (list->len <= 0) {
		cpyList->arr = NULL;
		cpyList->len = 0;
		cpyList->cap = 0;
		return cpyList;
	}

	index1 = clamp(index1, 1, list->len);
	index2 = clamp(index2, 1, list->len);
	if (index2 < index1) {
		runtime_error(1, "Invalide Indexe (Index 1 war %ld, Index 2 war %ld)\n", index1, index2);
	}

	index1--,index2--; // ddp indices start at 1, c indices at 0

	size_t new_list_cap = (index2 - index1) + 1; // + 1 if indices are equal
	ddpbool* arr = ALLOCATE(ddpbool, new_list_cap);

	
	memcpy(arr, &list->arr[index1], sizeof(ddpbool) * new_list_cap);
	

	cpyList->arr = arr;
	cpyList->len = new_list_cap;
	cpyList->cap = new_list_cap;
	return cpyList;
}

ddpbool inbuilt_ddpcharlist_equal(ddpcharlist* list1, ddpcharlist* list2) {
	if (list1 == list2) return true;
	if (list1->len != list2->len) return false; // if the length is different, it's a quick false return
	
	return memcmp(list1->arr, list2->arr, sizeof(ddpchar) * list1->len) == 0;
	
}

ddpcharlist* inbuilt_ddpcharlist_slice(ddpcharlist* list, ddpint index1, ddpint index2) {
	ddpcharlist* cpyList = ALLOCATE(ddpcharlist, 1); // up here to log the adress in debug mode
	DBGLOG("inbuilt_ddpcharlist_slice: %p", cpyList);

	if (list->len <= 0) {
		cpyList->arr = NULL;
		cpyList->len = 0;
		cpyList->cap = 0;
		return cpyList;
	}

	index1 = clamp(index1, 1, list->len);
	index2 = clamp(index2, 1, list->len);
	if (index2 < index1) {
		runtime_error(1, "Invalide Indexe (Index 1 war %ld, Index 2 war %ld)\n", index1, index2);
	}

	index1--,index2--; // ddp indices start at 1, c indices at 0

	size_t new_list_cap = (index2 - index1) + 1; // + 1 if indices are equal
	ddpchar* arr = ALLOCATE(ddpchar, new_list_cap);

	
	memcpy(arr, &list->arr[index1], sizeof(ddpchar) * new_list_cap);
	

	cpyList->arr = arr;
	cpyList->len = new_list_cap;
	cpyList->cap = new_list_cap;
	return cpyList;
}

ddpbool inbuilt_ddpstringlist_equal(ddpstringlist* list1, ddpstringlist* list2) {
	if (list1 == list2) return true;
	if (list1->len != list2->len) return false; // if the length is different, it's a quick false return
	
	for (size_t i = 0; i < list1->len; i++) {
		if (!inbuilt_string_equal(list1->arr[i], list2->arr[i])) return false;
	}
	return true;
	
}

ddpstringlist* inbuilt_ddpstringlist_slice(ddpstringlist* list, ddpint index1, ddpint index2) {
	ddpstringlist* cpyList = ALLOCATE(ddpstringlist, 1); // up here to log the adress in debug mode
	DBGLOG("inbuilt_ddpstringlist_slice: %p", cpyList);

	if (list->len <= 0) {
		cpyList->arr = NULL;
		cpyList->len = 0;
		cpyList->cap = 0;
		return cpyList;
	}

	index1 = clamp(index1, 1, list->len);
	index2 = clamp(index2, 1, list->len);
	if (index2 < index1) {
		runtime_error(1, "Invalide Indexe (Index 1 war %ld, Index 2 war %ld)\n", index1, index2);
	}

	index1--,index2--; // ddp indices start at 1, c indices at 0

	size_t new_list_cap = (index2 - index1) + 1; // + 1 if indices are equal
	ddpstring** arr = ALLOCATE(ddpstring*, new_list_cap);

	
	for (size_t i = index1; i <= index2 && i < list->len; i++) {
		arr[i] = inbuilt_deep_copy_string(list->arr[i]);
		inbuilt_increment_ref_count(arr[i], VK_STRING);
	}
	

	cpyList->arr = arr;
	cpyList->len = new_list_cap;
	cpyList->cap = new_list_cap;
	return cpyList;
}
