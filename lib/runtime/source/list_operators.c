#include "ddptypes.h"
#include "memory.h"
#include "debug.h"
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
		str->str = _ddp_reallocate(str->str, str->cap, new_cap);
		memcpy(str->str + str->cap-1, buffer, len);
		str->cap = new_cap;
	}
	int len = sprintf(buffer, "%lld", list->arr[list->len-1]);
	ddpint new_cap = str->cap + len;
	str->str = _ddp_reallocate(str->str, str->cap, new_cap);
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
		str->str = _ddp_reallocate(str->str, str->cap, new_cap);
		memcpy(str->str + str->cap-1, buffer, len);
		str->cap = new_cap;
	}
	int len = sprintf(buffer,  isinf(list->arr[list->len-1]) ? "Unendlich" : (isnan(list->arr[list->len-1]) ? "Keine Zahl (NaN)" : "%.16g"), list->arr[list->len-1]);
	ddpint new_cap = str->cap + len;
	str->str = _ddp_reallocate(str->str, str->cap, new_cap);
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
		str->str = _ddp_reallocate(str->str, str->cap, new_cap);
		memcpy(str->str + str->cap-1, buffer, len);
		str->cap = new_cap;
	}
	int len = sprintf(buffer, list->arr[list->len-1] ? "wahr" : "falsch");
	ddpint new_cap = str->cap + len;
	str->str = _ddp_reallocate(str->str, str->cap, new_cap);
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
		str->str = _ddp_reallocate(str->str, str->cap, new_cap);
		memcpy(str->str + str->cap-1, buffer, len);
		str->cap = new_cap;
	}
	utf8_char_to_string(ch, list->arr[list->len-1]);
	int len = sprintf(buffer, "%s", ch);
	ddpint new_cap = str->cap + len;
	str->str = _ddp_reallocate(str->str, str->cap, new_cap);
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
	ddpintlist* new_list = ALLOCATE(ddpintlist, 1);
	DBGLOG("_ddp_ddpintlist_slice: %p", new_list);
	new_list->len = 0;
	new_list->cap = 0;
	new_list->arr = NULL;

	if (list->len <= 0)
		return new_list;

	index1 = clamp(index1, 1, list->len);
	index2 = clamp(index2, 1, list->len);
	if (index2 < index1) {
		runtime_error(1, "Invalide Indexe (Index 1 war %ld, Index 2 war %ld)\n", index1, index2);
	}

	index1--,index2--; // ddp indices start at 1, c indices at 0

	new_list->len = (index2 - index1) + 1; // + 1 if indices are equal
	new_list->cap = GROW_CAPACITY(new_list->len);
	new_list->arr = ALLOCATE(ddpint, new_list->cap);

	
	memcpy(new_list->arr, &list->arr[index1], sizeof(ddpint) * new_list->len);
	

	return new_list;
}

ddpintlist* _ddp_ddpintlist_ddpintlist_verkettet(ddpintlist* list1, ddpintlist* list2) {
	ddpintlist* new_list = ALLOCATE(ddpintlist, 1);
	DBGLOG("_ddp_ddpintlist_ddpintlist_verkettet: %p", new_list);

	new_list->len = list1->len + list2->len;
	new_list->cap = list1->cap;
	while (new_list->cap < new_list->len) new_list->cap = GROW_CAPACITY(new_list->cap);
	new_list->arr = _ddp_reallocate(list1->arr, sizeof(ddpint) * list1->cap, sizeof(ddpint) * new_list->cap);
	
	memcpy(&new_list->arr[list1->len], list2->arr, sizeof(ddpint) * list2->len);
	

	list1->len = 0;
	list1->cap = 0;
	list1->arr = NULL;
	return new_list;
}
ddpintlist* _ddp_ddpintlist_ddpint_verkettet(ddpintlist* list, ddpint el) {
	ddpintlist* new_list = ALLOCATE(ddpintlist, 1);
	DBGLOG("_ddp_ddpintlist_ddpint_verkettet: %p", new_list);

	new_list->len = list->len + 1;
	new_list->cap = list->cap;
	while (new_list->cap < new_list->len) new_list->cap = GROW_CAPACITY(new_list->cap);
	new_list->arr = _ddp_reallocate(list->arr, sizeof(ddpint) * list->cap, sizeof(ddpint) * new_list->cap);
	
	new_list->arr[list->len] = el;
	

	list->len = 0;
	list->cap = 0;
	list->arr = NULL;
	return new_list;
}

ddpintlist* _ddp_ddpint_ddpint_verkettet(ddpint el1, ddpint el2) {
	ddpintlist* new_list = ALLOCATE(ddpintlist, 1); // up here to log the adress in debug mode
	new_list->len = 2;
	new_list->cap = GROW_CAPACITY(new_list->len);
	new_list->arr = ALLOCATE(ddpint, new_list->cap);
	DBGLOG("_ddp_ddpint_ddpint_verkettet: %p", new_list);

	new_list->arr[0] = el1;
	new_list->arr[1] = el2;

	return new_list;
}
ddpintlist* _ddp_ddpint_ddpintlist_verkettet(ddpint el, ddpintlist* list) {
	ddpintlist* new_list = ALLOCATE(ddpintlist, 1);
	DBGLOG("_ddp_ddpint_ddpintlist_verkettet: %p", new_list);
	
	new_list->len = list->len + 1;
	new_list->cap = list->cap;
	while (new_list->cap < new_list->len) new_list->cap = GROW_CAPACITY(new_list->cap);
	new_list->arr = _ddp_reallocate(list->arr, sizeof(ddpint) * list->cap, sizeof(ddpint) * new_list->cap);
	memmove(&new_list->arr[1], new_list->arr, sizeof(ddpint) * list->len);
	new_list->arr[0] = el;

	list->len = 0;
	list->cap = 0;
	list->arr = NULL;
	return new_list;
}


ddpbool _ddp_ddpfloatlist_equal(ddpfloatlist* list1, ddpfloatlist* list2) {
	if (list1 == list2) return true;
	if (list1->len != list2->len) return false; // if the length is different, it's a quick false return
	
	return memcmp(list1->arr, list2->arr, sizeof(ddpfloat) * list1->len) == 0;
	
}

ddpfloatlist* _ddp_ddpfloatlist_slice(ddpfloatlist* list, ddpint index1, ddpint index2) {
	ddpfloatlist* new_list = ALLOCATE(ddpfloatlist, 1);
	DBGLOG("_ddp_ddpfloatlist_slice: %p", new_list);
	new_list->len = 0;
	new_list->cap = 0;
	new_list->arr = NULL;

	if (list->len <= 0)
		return new_list;

	index1 = clamp(index1, 1, list->len);
	index2 = clamp(index2, 1, list->len);
	if (index2 < index1) {
		runtime_error(1, "Invalide Indexe (Index 1 war %ld, Index 2 war %ld)\n", index1, index2);
	}

	index1--,index2--; // ddp indices start at 1, c indices at 0

	new_list->len = (index2 - index1) + 1; // + 1 if indices are equal
	new_list->cap = GROW_CAPACITY(new_list->len);
	new_list->arr = ALLOCATE(ddpfloat, new_list->cap);

	
	memcpy(new_list->arr, &list->arr[index1], sizeof(ddpfloat) * new_list->len);
	

	return new_list;
}

ddpfloatlist* _ddp_ddpfloatlist_ddpfloatlist_verkettet(ddpfloatlist* list1, ddpfloatlist* list2) {
	ddpfloatlist* new_list = ALLOCATE(ddpfloatlist, 1);
	DBGLOG("_ddp_ddpfloatlist_ddpfloatlist_verkettet: %p", new_list);

	new_list->len = list1->len + list2->len;
	new_list->cap = list1->cap;
	while (new_list->cap < new_list->len) new_list->cap = GROW_CAPACITY(new_list->cap);
	new_list->arr = _ddp_reallocate(list1->arr, sizeof(ddpfloat) * list1->cap, sizeof(ddpfloat) * new_list->cap);
	
	memcpy(&new_list->arr[list1->len], list2->arr, sizeof(ddpfloat) * list2->len);
	

	list1->len = 0;
	list1->cap = 0;
	list1->arr = NULL;
	return new_list;
}
ddpfloatlist* _ddp_ddpfloatlist_ddpfloat_verkettet(ddpfloatlist* list, ddpfloat el) {
	ddpfloatlist* new_list = ALLOCATE(ddpfloatlist, 1);
	DBGLOG("_ddp_ddpfloatlist_ddpfloat_verkettet: %p", new_list);

	new_list->len = list->len + 1;
	new_list->cap = list->cap;
	while (new_list->cap < new_list->len) new_list->cap = GROW_CAPACITY(new_list->cap);
	new_list->arr = _ddp_reallocate(list->arr, sizeof(ddpfloat) * list->cap, sizeof(ddpfloat) * new_list->cap);
	
	new_list->arr[list->len] = el;
	

	list->len = 0;
	list->cap = 0;
	list->arr = NULL;
	return new_list;
}

ddpfloatlist* _ddp_ddpfloat_ddpfloat_verkettet(ddpfloat el1, ddpfloat el2) {
	ddpfloatlist* new_list = ALLOCATE(ddpfloatlist, 1); // up here to log the adress in debug mode
	new_list->len = 2;
	new_list->cap = GROW_CAPACITY(new_list->len);
	new_list->arr = ALLOCATE(ddpfloat, new_list->cap);
	DBGLOG("_ddp_ddpfloat_ddpfloat_verkettet: %p", new_list);

	new_list->arr[0] = el1;
	new_list->arr[1] = el2;

	return new_list;
}
ddpfloatlist* _ddp_ddpfloat_ddpfloatlist_verkettet(ddpfloat el, ddpfloatlist* list) {
	ddpfloatlist* new_list = ALLOCATE(ddpfloatlist, 1);
	DBGLOG("_ddp_ddpfloat_ddpfloatlist_verkettet: %p", new_list);
	
	new_list->len = list->len + 1;
	new_list->cap = list->cap;
	while (new_list->cap < new_list->len) new_list->cap = GROW_CAPACITY(new_list->cap);
	new_list->arr = _ddp_reallocate(list->arr, sizeof(ddpfloat) * list->cap, sizeof(ddpfloat) * new_list->cap);
	memmove(&new_list->arr[1], new_list->arr, sizeof(ddpfloat) * list->len);
	new_list->arr[0] = el;

	list->len = 0;
	list->cap = 0;
	list->arr = NULL;
	return new_list;
}


ddpbool _ddp_ddpboollist_equal(ddpboollist* list1, ddpboollist* list2) {
	if (list1 == list2) return true;
	if (list1->len != list2->len) return false; // if the length is different, it's a quick false return
	
	return memcmp(list1->arr, list2->arr, sizeof(ddpbool) * list1->len) == 0;
	
}

ddpboollist* _ddp_ddpboollist_slice(ddpboollist* list, ddpint index1, ddpint index2) {
	ddpboollist* new_list = ALLOCATE(ddpboollist, 1);
	DBGLOG("_ddp_ddpboollist_slice: %p", new_list);
	new_list->len = 0;
	new_list->cap = 0;
	new_list->arr = NULL;

	if (list->len <= 0)
		return new_list;

	index1 = clamp(index1, 1, list->len);
	index2 = clamp(index2, 1, list->len);
	if (index2 < index1) {
		runtime_error(1, "Invalide Indexe (Index 1 war %ld, Index 2 war %ld)\n", index1, index2);
	}

	index1--,index2--; // ddp indices start at 1, c indices at 0

	new_list->len = (index2 - index1) + 1; // + 1 if indices are equal
	new_list->cap = GROW_CAPACITY(new_list->len);
	new_list->arr = ALLOCATE(ddpbool, new_list->cap);

	
	memcpy(new_list->arr, &list->arr[index1], sizeof(ddpbool) * new_list->len);
	

	return new_list;
}

ddpboollist* _ddp_ddpboollist_ddpboollist_verkettet(ddpboollist* list1, ddpboollist* list2) {
	ddpboollist* new_list = ALLOCATE(ddpboollist, 1);
	DBGLOG("_ddp_ddpboollist_ddpboollist_verkettet: %p", new_list);

	new_list->len = list1->len + list2->len;
	new_list->cap = list1->cap;
	while (new_list->cap < new_list->len) new_list->cap = GROW_CAPACITY(new_list->cap);
	new_list->arr = _ddp_reallocate(list1->arr, sizeof(ddpbool) * list1->cap, sizeof(ddpbool) * new_list->cap);
	
	memcpy(&new_list->arr[list1->len], list2->arr, sizeof(ddpbool) * list2->len);
	

	list1->len = 0;
	list1->cap = 0;
	list1->arr = NULL;
	return new_list;
}
ddpboollist* _ddp_ddpboollist_ddpbool_verkettet(ddpboollist* list, ddpbool el) {
	ddpboollist* new_list = ALLOCATE(ddpboollist, 1);
	DBGLOG("_ddp_ddpboollist_ddpbool_verkettet: %p", new_list);

	new_list->len = list->len + 1;
	new_list->cap = list->cap;
	while (new_list->cap < new_list->len) new_list->cap = GROW_CAPACITY(new_list->cap);
	new_list->arr = _ddp_reallocate(list->arr, sizeof(ddpbool) * list->cap, sizeof(ddpbool) * new_list->cap);
	
	new_list->arr[list->len] = el;
	

	list->len = 0;
	list->cap = 0;
	list->arr = NULL;
	return new_list;
}

ddpboollist* _ddp_ddpbool_ddpbool_verkettet(ddpbool el1, ddpbool el2) {
	ddpboollist* new_list = ALLOCATE(ddpboollist, 1); // up here to log the adress in debug mode
	new_list->len = 2;
	new_list->cap = GROW_CAPACITY(new_list->len);
	new_list->arr = ALLOCATE(ddpbool, new_list->cap);
	DBGLOG("_ddp_ddpbool_ddpbool_verkettet: %p", new_list);

	new_list->arr[0] = el1;
	new_list->arr[1] = el2;

	return new_list;
}
ddpboollist* _ddp_ddpbool_ddpboollist_verkettet(ddpbool el, ddpboollist* list) {
	ddpboollist* new_list = ALLOCATE(ddpboollist, 1);
	DBGLOG("_ddp_ddpbool_ddpboollist_verkettet: %p", new_list);
	
	new_list->len = list->len + 1;
	new_list->cap = list->cap;
	while (new_list->cap < new_list->len) new_list->cap = GROW_CAPACITY(new_list->cap);
	new_list->arr = _ddp_reallocate(list->arr, sizeof(ddpbool) * list->cap, sizeof(ddpbool) * new_list->cap);
	memmove(&new_list->arr[1], new_list->arr, sizeof(ddpbool) * list->len);
	new_list->arr[0] = el;

	list->len = 0;
	list->cap = 0;
	list->arr = NULL;
	return new_list;
}


ddpbool _ddp_ddpcharlist_equal(ddpcharlist* list1, ddpcharlist* list2) {
	if (list1 == list2) return true;
	if (list1->len != list2->len) return false; // if the length is different, it's a quick false return
	
	return memcmp(list1->arr, list2->arr, sizeof(ddpchar) * list1->len) == 0;
	
}

ddpcharlist* _ddp_ddpcharlist_slice(ddpcharlist* list, ddpint index1, ddpint index2) {
	ddpcharlist* new_list = ALLOCATE(ddpcharlist, 1);
	DBGLOG("_ddp_ddpcharlist_slice: %p", new_list);
	new_list->len = 0;
	new_list->cap = 0;
	new_list->arr = NULL;

	if (list->len <= 0)
		return new_list;

	index1 = clamp(index1, 1, list->len);
	index2 = clamp(index2, 1, list->len);
	if (index2 < index1) {
		runtime_error(1, "Invalide Indexe (Index 1 war %ld, Index 2 war %ld)\n", index1, index2);
	}

	index1--,index2--; // ddp indices start at 1, c indices at 0

	new_list->len = (index2 - index1) + 1; // + 1 if indices are equal
	new_list->cap = GROW_CAPACITY(new_list->len);
	new_list->arr = ALLOCATE(ddpchar, new_list->cap);

	
	memcpy(new_list->arr, &list->arr[index1], sizeof(ddpchar) * new_list->len);
	

	return new_list;
}

ddpcharlist* _ddp_ddpcharlist_ddpcharlist_verkettet(ddpcharlist* list1, ddpcharlist* list2) {
	ddpcharlist* new_list = ALLOCATE(ddpcharlist, 1);
	DBGLOG("_ddp_ddpcharlist_ddpcharlist_verkettet: %p", new_list);

	new_list->len = list1->len + list2->len;
	new_list->cap = list1->cap;
	while (new_list->cap < new_list->len) new_list->cap = GROW_CAPACITY(new_list->cap);
	new_list->arr = _ddp_reallocate(list1->arr, sizeof(ddpchar) * list1->cap, sizeof(ddpchar) * new_list->cap);
	
	memcpy(&new_list->arr[list1->len], list2->arr, sizeof(ddpchar) * list2->len);
	

	list1->len = 0;
	list1->cap = 0;
	list1->arr = NULL;
	return new_list;
}
ddpcharlist* _ddp_ddpcharlist_ddpchar_verkettet(ddpcharlist* list, ddpchar el) {
	ddpcharlist* new_list = ALLOCATE(ddpcharlist, 1);
	DBGLOG("_ddp_ddpcharlist_ddpchar_verkettet: %p", new_list);

	new_list->len = list->len + 1;
	new_list->cap = list->cap;
	while (new_list->cap < new_list->len) new_list->cap = GROW_CAPACITY(new_list->cap);
	new_list->arr = _ddp_reallocate(list->arr, sizeof(ddpchar) * list->cap, sizeof(ddpchar) * new_list->cap);
	
	new_list->arr[list->len] = el;
	

	list->len = 0;
	list->cap = 0;
	list->arr = NULL;
	return new_list;
}

ddpcharlist* _ddp_ddpchar_ddpchar_verkettet(ddpchar el1, ddpchar el2) {
	ddpcharlist* new_list = ALLOCATE(ddpcharlist, 1); // up here to log the adress in debug mode
	new_list->len = 2;
	new_list->cap = GROW_CAPACITY(new_list->len);
	new_list->arr = ALLOCATE(ddpchar, new_list->cap);
	DBGLOG("_ddp_ddpchar_ddpchar_verkettet: %p", new_list);

	new_list->arr[0] = el1;
	new_list->arr[1] = el2;

	return new_list;
}
ddpcharlist* _ddp_ddpchar_ddpcharlist_verkettet(ddpchar el, ddpcharlist* list) {
	ddpcharlist* new_list = ALLOCATE(ddpcharlist, 1);
	DBGLOG("_ddp_ddpchar_ddpcharlist_verkettet: %p", new_list);
	
	new_list->len = list->len + 1;
	new_list->cap = list->cap;
	while (new_list->cap < new_list->len) new_list->cap = GROW_CAPACITY(new_list->cap);
	new_list->arr = _ddp_reallocate(list->arr, sizeof(ddpchar) * list->cap, sizeof(ddpchar) * new_list->cap);
	memmove(&new_list->arr[1], new_list->arr, sizeof(ddpchar) * list->len);
	new_list->arr[0] = el;

	list->len = 0;
	list->cap = 0;
	list->arr = NULL;
	return new_list;
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
	ddpstringlist* new_list = ALLOCATE(ddpstringlist, 1);
	DBGLOG("_ddp_ddpstringlist_slice: %p", new_list);
	new_list->len = 0;
	new_list->cap = 0;
	new_list->arr = NULL;

	if (list->len <= 0)
		return new_list;

	index1 = clamp(index1, 1, list->len);
	index2 = clamp(index2, 1, list->len);
	if (index2 < index1) {
		runtime_error(1, "Invalide Indexe (Index 1 war %ld, Index 2 war %ld)\n", index1, index2);
	}

	index1--,index2--; // ddp indices start at 1, c indices at 0

	new_list->len = (index2 - index1) + 1; // + 1 if indices are equal
	new_list->cap = GROW_CAPACITY(new_list->len);
	new_list->arr = ALLOCATE(ddpstring*, new_list->cap);

	
	size_t j = 0;
	for (size_t i = index1; i <= index2 && i < list->len; i++, j++) {
		new_list->arr[j] = _ddp_deep_copy_string(list->arr[i]);
	}
	

	return new_list;
}

ddpstringlist* _ddp_ddpstringlist_ddpstringlist_verkettet(ddpstringlist* list1, ddpstringlist* list2) {
	ddpstringlist* new_list = ALLOCATE(ddpstringlist, 1);
	DBGLOG("_ddp_ddpstringlist_ddpstringlist_verkettet: %p", new_list);

	new_list->len = list1->len + list2->len;
	new_list->cap = list1->cap;
	while (new_list->cap < new_list->len) new_list->cap = GROW_CAPACITY(new_list->cap);
	new_list->arr = _ddp_reallocate(list1->arr, sizeof(ddpstring*) * list1->cap, sizeof(ddpstring*) * new_list->cap);
	
	for (size_t i = 0; i < list2->len; i++) {
		new_list->arr[i+list1->len] = _ddp_deep_copy_string(list2->arr[i]);
	}
	

	list1->len = 0;
	list1->cap = 0;
	list1->arr = NULL;
	return new_list;
}
ddpstringlist* _ddp_ddpstringlist_ddpstring_verkettet(ddpstringlist* list, ddpstring* el) {
	ddpstringlist* new_list = ALLOCATE(ddpstringlist, 1);
	DBGLOG("_ddp_ddpstringlist_ddpstring*_verkettet: %p", new_list);

	new_list->len = list->len + 1;
	new_list->cap = list->cap;
	while (new_list->cap < new_list->len) new_list->cap = GROW_CAPACITY(new_list->cap);
	new_list->arr = _ddp_reallocate(list->arr, sizeof(ddpstring*) * list->cap, sizeof(ddpstring*) * new_list->cap);
	
	new_list->arr[list->len] = _ddp_deep_copy_string(el);
	

	list->len = 0;
	list->cap = 0;
	list->arr = NULL;
	return new_list;
}

ddpstringlist* _ddp_ddpstring_ddpstringlist_verkettet(ddpstring* str, ddpstringlist* list) {
	ddpstringlist* new_list = ALLOCATE(ddpstringlist, 1);
	DBGLOG("_ddp_ddpstring_ddpstringlist_verkettet: %p", new_list);

	new_list->len = list->len + 1;
	new_list->cap = list->cap;
	while (new_list->cap < new_list->len) new_list->cap = GROW_CAPACITY(new_list->cap);
	new_list->arr = _ddp_reallocate(list->arr, sizeof(ddpstring*) * list->cap, sizeof(ddpstring*) * new_list->cap);
	memmove(&new_list->arr[1], new_list->arr, sizeof(ddpstring*) * list->len);
	new_list->arr[0] = _ddp_deep_copy_string(str);

	list->len = 0;
	list->cap = 0;
	list->arr = NULL;
	return new_list;
}

