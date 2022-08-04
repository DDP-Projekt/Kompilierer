extern ddpbool inbuilt_string_equal(ddpstring*, ddpstring*);
extern void inbuilt_increment_ref_count(void*, uint8_t);
extern ddpstring* inbuilt_deep_copy_string(ddpstring*);

static ddpint clamp(ddpint i, ddpint min, ddpint max) {
  const ddpint t = i < min ? min : i;
  return t > max ? max : t;
}

{{range .}}
ddpbool inbuilt_{{ .T }}_equal({{ .T }}* list1, {{ .T }}* list2) {
	if (list1 == list2) return true;
	if (list1->len != list2->len) return false; // if the length is different, it's a quick false return
	{{if .D}}
	for (size_t i = 0; i < list1->len; i++) {
		if (!inbuilt_string_equal(list1->arr[i], list2->arr[i])) return false;
	}
	return true;
	{{else}}
	return memcmp(list1->arr, list2->arr, sizeof({{ .E }}) * list1->len) == 0;
	{{end}}
}

{{ .T }}* inbuilt_{{ .T }}_slice({{ .T }}* list, ddpint index1, ddpint index2) {
	{{ .T }}* cpyList = ALLOCATE({{ .T }}, 1); // up here to log the adress in debug mode
	DBGLOG("inbuilt_{{ .T }}_slice: %p", cpyList);

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
	{{ .E }}* arr = ALLOCATE({{ .E }}, new_list_cap);

	{{if .D}}
	for (size_t i = index1; i <= index2 && i < list->len; i++) {
		arr[i] = inbuilt_deep_copy_string(list->arr[i]);
		inbuilt_increment_ref_count(arr[i], VK_STRING);
	}
	{{else}}
	memcpy(arr, &list->arr[index1], sizeof({{ .E }}) * new_list_cap);
	{{end}}

	cpyList->arr = arr;
	cpyList->len = new_list_cap;
	cpyList->cap = new_list_cap;
	return cpyList;
}

{{ .T }}* inbuilt_{{ .T }}_{{ .T }}_verkettet({{ .T }}* list1, {{ .T }}* list2) {
	{{ .T }}* newList = ALLOCATE({{ .T }}, 1); // up here to log the adress in debug mode
	newList->len = list1->len + list2->len;
	newList->cap = newList->len;
	newList->arr = ALLOCATE({{ .E }}, newList->cap);
	DBGLOG("inbuilt_{{ .T }}_{{ .T }}_verkettet: %p", newList);

	{{if .D}}
	for (size_t i = 0; i < list1->len; i++) {
		newList->arr[i] = inbuilt_deep_copy_string(list1->arr[i]);
		inbuilt_increment_ref_count(newList->arr[i], VK_STRING);
	}
	for (size_t i = 0; i < list2->len; i++) {
		newList->arr[i+list1->len] = inbuilt_deep_copy_string(list2->arr[i]);
		inbuilt_increment_ref_count(newList->arr[i+list1->len], VK_STRING);
	}
	{{else}}
	memcpy(newList->arr, list1->arr, sizeof({{ .E }}) * list1->len);
	memcpy(&newList->arr[list1->len], list2->arr, sizeof({{ .E }}) * list2->len);
	{{end}}

	return newList;
}
{{ .T }}* inbuilt_{{ .T }}_{{ .E }}_verkettet({{ .T }}* list, {{ .E }} el) {
	{{ .T }}* newList = ALLOCATE({{ .T }}, 1); // up here to log the adress in debug mode
	newList->len = list->len + 1;
	newList->cap = newList->len;
	newList->arr = ALLOCATE({{ .E }}, newList->cap);
	DBGLOG("inbuilt_{{ .T }}_{{ .E }}_verkettet: %p", newList);

	{{if .D}}
	for (size_t i = 0; i < list->len; i++) {
		newList->arr[i] = inbuilt_deep_copy_string(list->arr[i]);
		inbuilt_increment_ref_count(newList->arr[i], VK_STRING);
	}
	newList->arr[list->len] = inbuilt_deep_copy_string(el);
	inbuilt_increment_ref_count(newList->arr[list->len], VK_STRING);
	{{else}}
	memcpy(newList->arr, list->arr, sizeof({{ .E }}) * list->len);
	newList->arr[list->len] = el;
	{{end}}

	return newList;
}
{{if .D}}
{{ .T }}* inbuilt_ddpstring_ddpstringlist_verkettet({{ .E }} str, {{ .T }}* list) {
	{{ .T }}* newList = ALLOCATE({{ .T }}, 1); // up here to log the adress in debug mode
	newList->len = list->len + 1;
	newList->cap = newList->len;
	newList->arr = ALLOCATE({{ .E }}, newList->cap);
	DBGLOG("inbuilt_ddpstring_ddpstringlist_verkettet: %p", newList);

	newList->arr[0] = inbuilt_deep_copy_string(str);
	inbuilt_increment_ref_count(newList->arr[0], VK_STRING);
	for (size_t i = 0; i < list->len; i++) {
		newList->arr[i+1] = inbuilt_deep_copy_string(list->arr[i]);
		inbuilt_increment_ref_count(newList->arr[i+1], VK_STRING);
	}

	return newList;
}
{{else}}
{{ .T }}* inbuilt_{{ .E }}_{{ .E }}_verkettet({{ .E }} el1, {{ .E }} el2) {
	{{ .T }}* newList = ALLOCATE({{ .T }}, 1); // up here to log the adress in debug mode
	newList->len = 2;
	newList->cap = newList->len;
	newList->arr = ALLOCATE({{ .E }}, newList->cap);
	DBGLOG("inbuilt_{{ .E }}_{{ .E }}_verkettet: %p", newList);

	newList->arr[0] = el1;
	newList->arr[1] = el2;

	return newList;
}
{{ .T }}* inbuilt_{{ .E }}_{{ .T }}_verkettet({{ .E }} el, {{ .T }}* list) {
	{{ .T }}* newList = ALLOCATE({{ .T }}, 1); // up here to log the adress in debug mode
	newList->len = list->len + 1;
	newList->cap = newList->len;
	newList->arr = ALLOCATE({{ .E }}, newList->cap);
	DBGLOG("inbuilt_{{ .E }}_{{ .T }}_verkettet: %p", newList);
	
	newList->arr[0] = el;
	memcpy(&newList->arr[1], list->arr, sizeof({{ .E }}) * list->len);

	return newList;
}
{{end}}
{{end}}