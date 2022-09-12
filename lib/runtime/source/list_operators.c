#include "ddptypes.h"
#include "memory.h"
#include "debug.h"
#include "gc.h"
#include "utf8/utf8.h"
#include <math.h>

ddpstring* _ddp_ddpintlist_to_string(ddpintlist* list) {
	ddpstring* str = ALLOCATE(ddpstring, 1); // up here to log the adress in debug mode
	str->str = ALLOCATE(char, 1);
	str->str[0] = '\0';
	str->cap = 1;
	DBGLOG("_ddp_ddpintlist_to_string: %p", str);

	if (list->len <= 0) {
		return str;
	}
	char buffer[23];
	for (size_t i = 0; i < list->len-1; i++) {
		int len = sprintf(buffer, "%lld, ", list->arr[i]);
		ddpint new_cap = str->cap + len;
		str->str = reallocate(str->str, str->cap, new_cap);
		memcpy(str->str + str->cap-1, buffer, len);
		str->cap = new_cap;
	}
	int len = sprintf(buffer, "%lld", list->arr[list->len-1]);
	ddpint new_cap = str->cap + len;
	str->str = reallocate(str->str, str->cap, new_cap);
	memcpy(str->str + str->cap-1, buffer, len);
	str->cap = new_cap;

	str->str[str->cap-1] = '\0';
	return str;
}

ddpstring* _ddp_ddpfloatlist_to_string(ddpfloatlist* list) {
	ddpstring* str = ALLOCATE(ddpstring, 1); // up here to log the adress in debug mode
	str->str = ALLOCATE(char, 1);
	str->str[0] = '\0';
	str->cap = 1;
	DBGLOG("_ddp_ddpfloatlist_to_string: %p", str);

	if (list->len <= 0) {
		return str;
	}
	char buffer[53];
	for (size_t i = 0; i < list->len-1; i++) {
		int len = sprintf(buffer, isinf(list->arr[i]) ? "Unendlich, " : (isnan(list->arr[i]) ? "Keine Zahl (NaN), " : "%.16g, "), list->arr[i]);
		ddpint new_cap = str->cap + len;
		str->str = reallocate(str->str, str->cap, new_cap);
		memcpy(str->str + str->cap-1, buffer, len);
		str->cap = new_cap;
	}
	int len = sprintf(buffer,  isinf(list->arr[list->len-1]) ? "Unendlich" : (isnan(list->arr[list->len-1]) ? "Keine Zahl (NaN)" : "%.16g"), list->arr[list->len-1]);
	ddpint new_cap = str->cap + len;
	str->str = reallocate(str->str, str->cap, new_cap);
	memcpy(str->str + str->cap-1, buffer, len);
	str->cap = new_cap;

	str->str[str->cap-1] = '\0';
	return str;
}

ddpstring* _ddp_ddpboollist_to_string(ddpboollist* list) {
	ddpstring* str = ALLOCATE(ddpstring, 1); // up here to log the adress in debug mode
	str->str = ALLOCATE(char, 1);
	str->str[0] = '\0';
	str->cap = 1;
	DBGLOG("_ddp_ddpboollist_to_string: %p", str);

	if (list->len <= 0) {
		return str;
	}
	char buffer[9];
	for (size_t i = 0; i < list->len-1; i++) {
		int len = sprintf(buffer, list->arr[i] ? "wahr, " : "falsch, ");
		ddpint new_cap = str->cap + len;
		str->str = reallocate(str->str, str->cap, new_cap);
		memcpy(str->str + str->cap-1, buffer, len);
		str->cap = new_cap;
	}
	int len = sprintf(buffer, list->arr[list->len-1] ? "wahr" : "falsch");
	ddpint new_cap = str->cap + len;
	str->str = reallocate(str->str, str->cap, new_cap);
	memcpy(str->str + str->cap-1, buffer, len);
	str->cap = new_cap;

	str->str[str->cap-1] = '\0';
	return str;
}

ddpstring* _ddp_ddpcharlist_to_string(ddpcharlist* list) {
	ddpstring* str = ALLOCATE(ddpstring, 1); // up here to log the adress in debug mode
	str->str = ALLOCATE(char, 1);
	str->str[0] = '\0';
	str->cap = 1;
	DBGLOG("_ddp_ddpcharlist_to_string: %p", str);

	if (list->len <= 0) {
		return str;
	}
	char buffer[7];
	char ch[5];
	for (size_t i = 0; i < list->len-1; i++) {
		utf8_char_to_string(ch, list->arr[i]);
		int len = sprintf(buffer, "%s, ", ch);
		ddpint new_cap = str->cap + len;
		str->str = reallocate(str->str, str->cap, new_cap);
		memcpy(str->str + str->cap-1, buffer, len);
		str->cap = new_cap;
	}
	utf8_char_to_string(ch, list->arr[list->len-1]);
	int len = sprintf(buffer, "%s", ch);
	ddpint new_cap = str->cap + len;
	str->str = reallocate(str->str, str->cap, new_cap);
	memcpy(str->str + str->cap-1, buffer, len);
	str->cap = new_cap;

	str->str[str->cap-1] = '\0';
	return str;
}

ddpstring* _ddp_ddpstringlist_to_string(ddpstringlist* list) {
	ddpstring* str = ALLOCATE(ddpstring, 1); // up here to log the adress in debug mode
	DBGLOG("_ddp_ddpstringlist_to_string: %p", str);

	if (list->len <= 0) {
		str->str = ALLOCATE(char, 1);
		str->str[0] = '\0';
		str->cap = 1;
		return str;
	}

	str->cap = 1; // 1 for the null terminator
	for (size_t i = 0; i < list->len-1; i++) {
		str->cap += strlen(list->arr[i]->str)+2; // + 2 for the ", "
	}
	str->cap += strlen(list->arr[list->len-1]->str);

	str->str = ALLOCATE(char, str->cap);

	size_t j = 0;
	for (size_t i = 0; i < list->len-1; i++) {
		sprintf(str->str + j, "%s, ", list->arr[i]->str);
		j += strlen(list->arr[i]->str)+2;
	}
	sprintf(str->str + j, "%s", list->arr[list->len-1]->str);
	str->str[str->cap-1] = '\0';

	return str;
}

/***** Partially generated code *****/
extern ddpbool _ddp_string_equal(ddpstring*, ddpstring*);
extern ddpstring* _ddp_deep_copy_string(ddpstring*);

static ddpint clamp(ddpint i, ddpint min, ddpint max) {
  const ddpint t = i < min ? min : i;
  return t > max ? max : t;
}


ddpbool _ddp_ddpintlist_equal(ddpintlist* list1, ddpintlist* list2) {
	if (list1 == list2) return true;
	if (list1->len != list2->len) return false; // if the length is different, it's a quick false return
	
	return memcmp(list1->arr, list2->arr, sizeof(ddpint) * list1->len) == 0;
	
}

ddpintlist* _ddp_ddpintlist_slice(ddpintlist* list, ddpint index1, ddpint index2) {
	DBGLOG("_ddp_ddpintlist_slice: %p", list);

	if (list->len <= 0)
		return list;

	index1 = clamp(index1, 1, list->len);
	index2 = clamp(index2, 1, list->len);
	if (index2 < index1) {
		runtime_error(1, "Invalide Indexe (Index 1 war %ld, Index 2 war %ld)\n", index1, index2);
	}

	index1--,index2--; // ddp indices start at 1, c indices at 0

	size_t new_list_len = (index2 - index1) + 1; // + 1 if indices are equal
	size_t new_list_cap = GROW_CAPACITY(new_list_len);
	ddpint* arr = ALLOCATE(ddpint, new_list_cap);

	
	memcpy(arr, &list->arr[index1], sizeof(ddpint) * new_list_len);
	

	FREE_ARRAY(ddpint, list->arr, list->cap);
	list->arr = arr;
	list->len = new_list_len;
	list->cap = new_list_cap;
	return list;
}

ddpintlist* _ddp_ddpintlist_ddpintlist_verkettet(ddpintlist* list1, ddpintlist* list2) {
	DBGLOG("_ddp_ddpintlist_ddpintlist_verkettet: %p", list1);

	size_t new_len = list1->len + list2->len;
	size_t new_cap = list1->cap;
	while (new_cap < new_len) new_cap = GROW_CAPACITY(new_cap);
	list1->arr = reallocate(list1->arr, sizeof(ddpint) * list1->cap, sizeof(ddpint) * new_cap);
	
	memcpy(&list1->arr[list1->len], list2->arr, sizeof(ddpint) * list2->len);
	

	list1->len = new_len;
	list1->cap = new_cap;
	return list1;
}
ddpintlist* _ddp_ddpintlist_ddpint_verkettet(ddpintlist* list, ddpint el) {
	DBGLOG("_ddp_ddpintlist_ddpint_verkettet: %p", list);

	size_t new_len = list->len + 1;
	size_t new_cap = list->cap;
	while (new_cap < new_len) new_cap = GROW_CAPACITY(new_cap);
	list->arr = reallocate(list->arr, sizeof(ddpint) * list->cap, sizeof(ddpint) * new_cap);
	
	list->arr[list->len] = el;
	

	list->len = new_len;
	list->cap = new_cap;
	return list;
}

ddpintlist* _ddp_ddpint_ddpint_verkettet(ddpint el1, ddpint el2) {
	ddpintlist* newList = ALLOCATE(ddpintlist, 1); // up here to log the adress in debug mode
	newList->len = 2;
	newList->cap = GROW_CAPACITY(newList->len);
	newList->arr = ALLOCATE(ddpint, newList->cap);
	DBGLOG("_ddp_ddpint_ddpint_verkettet: %p", newList);

	newList->arr[0] = el1;
	newList->arr[1] = el2;

	return newList;
}
ddpintlist* _ddp_ddpint_ddpintlist_verkettet(ddpint el, ddpintlist* list) {
	DBGLOG("_ddp_ddpint_ddpintlist_verkettet: %p", list);
	
	size_t new_len = list->len + 1;
	size_t new_cap = list->cap;
	while (new_cap < new_len) new_cap = GROW_CAPACITY(new_cap);
	list->arr = reallocate(list->arr, sizeof(ddpint) * list->cap, sizeof(ddpint) * new_cap);
	memmove(&list->arr[1], list->arr, sizeof(ddpint) * list->len);
	list->arr[0] = el;

	list->len = new_len;
	list->cap = new_cap;
	return list;
}


ddpbool _ddp_ddpfloatlist_equal(ddpfloatlist* list1, ddpfloatlist* list2) {
	if (list1 == list2) return true;
	if (list1->len != list2->len) return false; // if the length is different, it's a quick false return
	
	return memcmp(list1->arr, list2->arr, sizeof(ddpfloat) * list1->len) == 0;
	
}

ddpfloatlist* _ddp_ddpfloatlist_slice(ddpfloatlist* list, ddpint index1, ddpint index2) {
	DBGLOG("_ddp_ddpfloatlist_slice: %p", list);

	if (list->len <= 0)
		return list;

	index1 = clamp(index1, 1, list->len);
	index2 = clamp(index2, 1, list->len);
	if (index2 < index1) {
		runtime_error(1, "Invalide Indexe (Index 1 war %ld, Index 2 war %ld)\n", index1, index2);
	}

	index1--,index2--; // ddp indices start at 1, c indices at 0

	size_t new_list_len = (index2 - index1) + 1; // + 1 if indices are equal
	size_t new_list_cap = GROW_CAPACITY(new_list_len);
	ddpfloat* arr = ALLOCATE(ddpfloat, new_list_cap);

	
	memcpy(arr, &list->arr[index1], sizeof(ddpfloat) * new_list_len);
	

	FREE_ARRAY(ddpfloat, list->arr, list->cap);
	list->arr = arr;
	list->len = new_list_len;
	list->cap = new_list_cap;
	return list;
}

ddpfloatlist* _ddp_ddpfloatlist_ddpfloatlist_verkettet(ddpfloatlist* list1, ddpfloatlist* list2) {
	DBGLOG("_ddp_ddpfloatlist_ddpfloatlist_verkettet: %p", list1);

	size_t new_len = list1->len + list2->len;
	size_t new_cap = list1->cap;
	while (new_cap < new_len) new_cap = GROW_CAPACITY(new_cap);
	list1->arr = reallocate(list1->arr, sizeof(ddpfloat) * list1->cap, sizeof(ddpfloat) * new_cap);
	
	memcpy(&list1->arr[list1->len], list2->arr, sizeof(ddpfloat) * list2->len);
	

	list1->len = new_len;
	list1->cap = new_cap;
	return list1;
}
ddpfloatlist* _ddp_ddpfloatlist_ddpfloat_verkettet(ddpfloatlist* list, ddpfloat el) {
	DBGLOG("_ddp_ddpfloatlist_ddpfloat_verkettet: %p", list);

	size_t new_len = list->len + 1;
	size_t new_cap = list->cap;
	while (new_cap < new_len) new_cap = GROW_CAPACITY(new_cap);
	list->arr = reallocate(list->arr, sizeof(ddpfloat) * list->cap, sizeof(ddpfloat) * new_cap);
	
	list->arr[list->len] = el;
	

	list->len = new_len;
	list->cap = new_cap;
	return list;
}

ddpfloatlist* _ddp_ddpfloat_ddpfloat_verkettet(ddpfloat el1, ddpfloat el2) {
	ddpfloatlist* newList = ALLOCATE(ddpfloatlist, 1); // up here to log the adress in debug mode
	newList->len = 2;
	newList->cap = GROW_CAPACITY(newList->len);
	newList->arr = ALLOCATE(ddpfloat, newList->cap);
	DBGLOG("_ddp_ddpfloat_ddpfloat_verkettet: %p", newList);

	newList->arr[0] = el1;
	newList->arr[1] = el2;

	return newList;
}
ddpfloatlist* _ddp_ddpfloat_ddpfloatlist_verkettet(ddpfloat el, ddpfloatlist* list) {
	DBGLOG("_ddp_ddpfloat_ddpfloatlist_verkettet: %p", list);
	
	size_t new_len = list->len + 1;
	size_t new_cap = list->cap;
	while (new_cap < new_len) new_cap = GROW_CAPACITY(new_cap);
	list->arr = reallocate(list->arr, sizeof(ddpfloat) * list->cap, sizeof(ddpfloat) * new_cap);
	memmove(&list->arr[1], list->arr, sizeof(ddpfloat) * list->len);
	list->arr[0] = el;

	list->len = new_len;
	list->cap = new_cap;
	return list;
}


ddpbool _ddp_ddpboollist_equal(ddpboollist* list1, ddpboollist* list2) {
	if (list1 == list2) return true;
	if (list1->len != list2->len) return false; // if the length is different, it's a quick false return
	
	return memcmp(list1->arr, list2->arr, sizeof(ddpbool) * list1->len) == 0;
	
}

ddpboollist* _ddp_ddpboollist_slice(ddpboollist* list, ddpint index1, ddpint index2) {
	DBGLOG("_ddp_ddpboollist_slice: %p", list);

	if (list->len <= 0)
		return list;

	index1 = clamp(index1, 1, list->len);
	index2 = clamp(index2, 1, list->len);
	if (index2 < index1) {
		runtime_error(1, "Invalide Indexe (Index 1 war %ld, Index 2 war %ld)\n", index1, index2);
	}

	index1--,index2--; // ddp indices start at 1, c indices at 0

	size_t new_list_len = (index2 - index1) + 1; // + 1 if indices are equal
	size_t new_list_cap = GROW_CAPACITY(new_list_len);
	ddpbool* arr = ALLOCATE(ddpbool, new_list_cap);

	
	memcpy(arr, &list->arr[index1], sizeof(ddpbool) * new_list_len);
	

	FREE_ARRAY(ddpbool, list->arr, list->cap);
	list->arr = arr;
	list->len = new_list_len;
	list->cap = new_list_cap;
	return list;
}

ddpboollist* _ddp_ddpboollist_ddpboollist_verkettet(ddpboollist* list1, ddpboollist* list2) {
	DBGLOG("_ddp_ddpboollist_ddpboollist_verkettet: %p", list1);

	size_t new_len = list1->len + list2->len;
	size_t new_cap = list1->cap;
	while (new_cap < new_len) new_cap = GROW_CAPACITY(new_cap);
	list1->arr = reallocate(list1->arr, sizeof(ddpbool) * list1->cap, sizeof(ddpbool) * new_cap);
	
	memcpy(&list1->arr[list1->len], list2->arr, sizeof(ddpbool) * list2->len);
	

	list1->len = new_len;
	list1->cap = new_cap;
	return list1;
}
ddpboollist* _ddp_ddpboollist_ddpbool_verkettet(ddpboollist* list, ddpbool el) {
	DBGLOG("_ddp_ddpboollist_ddpbool_verkettet: %p", list);

	size_t new_len = list->len + 1;
	size_t new_cap = list->cap;
	while (new_cap < new_len) new_cap = GROW_CAPACITY(new_cap);
	list->arr = reallocate(list->arr, sizeof(ddpbool) * list->cap, sizeof(ddpbool) * new_cap);
	
	list->arr[list->len] = el;
	

	list->len = new_len;
	list->cap = new_cap;
	return list;
}

ddpboollist* _ddp_ddpbool_ddpbool_verkettet(ddpbool el1, ddpbool el2) {
	ddpboollist* newList = ALLOCATE(ddpboollist, 1); // up here to log the adress in debug mode
	newList->len = 2;
	newList->cap = GROW_CAPACITY(newList->len);
	newList->arr = ALLOCATE(ddpbool, newList->cap);
	DBGLOG("_ddp_ddpbool_ddpbool_verkettet: %p", newList);

	newList->arr[0] = el1;
	newList->arr[1] = el2;

	return newList;
}
ddpboollist* _ddp_ddpbool_ddpboollist_verkettet(ddpbool el, ddpboollist* list) {
	DBGLOG("_ddp_ddpbool_ddpboollist_verkettet: %p", list);
	
	size_t new_len = list->len + 1;
	size_t new_cap = list->cap;
	while (new_cap < new_len) new_cap = GROW_CAPACITY(new_cap);
	list->arr = reallocate(list->arr, sizeof(ddpbool) * list->cap, sizeof(ddpbool) * new_cap);
	memmove(&list->arr[1], list->arr, sizeof(ddpbool) * list->len);
	list->arr[0] = el;

	list->len = new_len;
	list->cap = new_cap;
	return list;
}


ddpbool _ddp_ddpcharlist_equal(ddpcharlist* list1, ddpcharlist* list2) {
	if (list1 == list2) return true;
	if (list1->len != list2->len) return false; // if the length is different, it's a quick false return
	
	return memcmp(list1->arr, list2->arr, sizeof(ddpchar) * list1->len) == 0;
	
}

ddpcharlist* _ddp_ddpcharlist_slice(ddpcharlist* list, ddpint index1, ddpint index2) {
	DBGLOG("_ddp_ddpcharlist_slice: %p", list);

	if (list->len <= 0)
		return list;

	index1 = clamp(index1, 1, list->len);
	index2 = clamp(index2, 1, list->len);
	if (index2 < index1) {
		runtime_error(1, "Invalide Indexe (Index 1 war %ld, Index 2 war %ld)\n", index1, index2);
	}

	index1--,index2--; // ddp indices start at 1, c indices at 0

	size_t new_list_len = (index2 - index1) + 1; // + 1 if indices are equal
	size_t new_list_cap = GROW_CAPACITY(new_list_len);
	ddpchar* arr = ALLOCATE(ddpchar, new_list_cap);

	
	memcpy(arr, &list->arr[index1], sizeof(ddpchar) * new_list_len);
	

	FREE_ARRAY(ddpchar, list->arr, list->cap);
	list->arr = arr;
	list->len = new_list_len;
	list->cap = new_list_cap;
	return list;
}

ddpcharlist* _ddp_ddpcharlist_ddpcharlist_verkettet(ddpcharlist* list1, ddpcharlist* list2) {
	DBGLOG("_ddp_ddpcharlist_ddpcharlist_verkettet: %p", list1);

	size_t new_len = list1->len + list2->len;
	size_t new_cap = list1->cap;
	while (new_cap < new_len) new_cap = GROW_CAPACITY(new_cap);
	list1->arr = reallocate(list1->arr, sizeof(ddpchar) * list1->cap, sizeof(ddpchar) * new_cap);
	
	memcpy(&list1->arr[list1->len], list2->arr, sizeof(ddpchar) * list2->len);
	

	list1->len = new_len;
	list1->cap = new_cap;
	return list1;
}
ddpcharlist* _ddp_ddpcharlist_ddpchar_verkettet(ddpcharlist* list, ddpchar el) {
	DBGLOG("_ddp_ddpcharlist_ddpchar_verkettet: %p", list);

	size_t new_len = list->len + 1;
	size_t new_cap = list->cap;
	while (new_cap < new_len) new_cap = GROW_CAPACITY(new_cap);
	list->arr = reallocate(list->arr, sizeof(ddpchar) * list->cap, sizeof(ddpchar) * new_cap);
	
	list->arr[list->len] = el;
	

	list->len = new_len;
	list->cap = new_cap;
	return list;
}

ddpcharlist* _ddp_ddpchar_ddpchar_verkettet(ddpchar el1, ddpchar el2) {
	ddpcharlist* newList = ALLOCATE(ddpcharlist, 1); // up here to log the adress in debug mode
	newList->len = 2;
	newList->cap = GROW_CAPACITY(newList->len);
	newList->arr = ALLOCATE(ddpchar, newList->cap);
	DBGLOG("_ddp_ddpchar_ddpchar_verkettet: %p", newList);

	newList->arr[0] = el1;
	newList->arr[1] = el2;

	return newList;
}
ddpcharlist* _ddp_ddpchar_ddpcharlist_verkettet(ddpchar el, ddpcharlist* list) {
	DBGLOG("_ddp_ddpchar_ddpcharlist_verkettet: %p", list);
	
	size_t new_len = list->len + 1;
	size_t new_cap = list->cap;
	while (new_cap < new_len) new_cap = GROW_CAPACITY(new_cap);
	list->arr = reallocate(list->arr, sizeof(ddpchar) * list->cap, sizeof(ddpchar) * new_cap);
	memmove(&list->arr[1], list->arr, sizeof(ddpchar) * list->len);
	list->arr[0] = el;

	list->len = new_len;
	list->cap = new_cap;
	return list;
}


ddpbool _ddp_ddpstringlist_equal(ddpstringlist* list1, ddpstringlist* list2) {
	if (list1 == list2) return true;
	if (list1->len != list2->len) return false; // if the length is different, it's a quick false return
	
	for (size_t i = 0; i < list1->len; i++) {
		if (!_ddp_string_equal(list1->arr[i], list2->arr[i])) return false;
	}
	return true;
	
}

ddpstringlist* _ddp_ddpstringlist_slice(ddpstringlist* list, ddpint index1, ddpint index2) {
	DBGLOG("_ddp_ddpstringlist_slice: %p", list);

	if (list->len <= 0)
		return list;

	index1 = clamp(index1, 1, list->len);
	index2 = clamp(index2, 1, list->len);
	if (index2 < index1) {
		runtime_error(1, "Invalide Indexe (Index 1 war %ld, Index 2 war %ld)\n", index1, index2);
	}

	index1--,index2--; // ddp indices start at 1, c indices at 0

	size_t new_list_len = (index2 - index1) + 1; // + 1 if indices are equal
	size_t new_list_cap = GROW_CAPACITY(new_list_len);
	ddpstring** arr = ALLOCATE(ddpstring*, new_list_cap);

	
	size_t j = 0;
	for (size_t i = index1; i <= index2 && i < list->len; i++, j++) {
		arr[j] = _ddp_deep_copy_string(list->arr[i]);
		_ddp_increment_ref_count(arr[j], VK_STRING);
	}
	

	FREE_ARRAY(ddpstring*, list->arr, list->cap);
	list->arr = arr;
	list->len = new_list_len;
	list->cap = new_list_cap;
	return list;
}

ddpstringlist* _ddp_ddpstringlist_ddpstringlist_verkettet(ddpstringlist* list1, ddpstringlist* list2) {
	DBGLOG("_ddp_ddpstringlist_ddpstringlist_verkettet: %p", list1);

	size_t new_len = list1->len + list2->len;
	size_t new_cap = list1->cap;
	while (new_cap < new_len) new_cap = GROW_CAPACITY(new_cap);
	list1->arr = reallocate(list1->arr, sizeof(ddpstring*) * list1->cap, sizeof(ddpstring*) * new_cap);
	
	for (size_t i = 0; i < list2->len; i++) {
		list1->arr[i+list1->len] = _ddp_deep_copy_string(list2->arr[i]);
		_ddp_increment_ref_count(list1->arr[i+list1->len], VK_STRING);
	}
	

	list1->len = new_len;
	list1->cap = new_cap;
	return list1;
}
ddpstringlist* _ddp_ddpstringlist_ddpstring_verkettet(ddpstringlist* list, ddpstring* el) {
	DBGLOG("_ddp_ddpstringlist_ddpstring*_verkettet: %p", list);

	size_t new_len = list->len + 1;
	size_t new_cap = list->cap;
	while (new_cap < new_len) new_cap = GROW_CAPACITY(new_cap);
	list->arr = reallocate(list->arr, sizeof(ddpstring*) * list->cap, sizeof(ddpstring*) * new_cap);
	
	list->arr[list->len] = _ddp_deep_copy_string(el);
	_ddp_increment_ref_count(list->arr[list->len], VK_STRING);
	

	list->len = new_len;
	list->cap = new_cap;
	return list;
}

ddpstringlist* _ddp_ddpstring_ddpstringlist_verkettet(ddpstring* str, ddpstringlist* list) {
	DBGLOG("_ddp_ddpstring_ddpstringlist_verkettet: %p", list);

	size_t new_len = list->len + 1;
	size_t new_cap = list->cap;
	while (new_cap < new_len) new_cap = GROW_CAPACITY(new_cap);
	list->arr = reallocate(list->arr, sizeof(ddpstring*) * list->cap, sizeof(ddpstring*) * new_cap);
	memmove(&list->arr[1], list->arr, sizeof(ddpstring*) * list->len);
	list->arr[0] = _ddp_deep_copy_string(str);
	_ddp_increment_ref_count(list->arr[0], VK_STRING);

	list->len = new_len;
	list->cap = new_cap;
	return list;
}
