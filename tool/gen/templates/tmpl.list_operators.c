extern ddpbool _ddp_string_equal(ddpstring*, ddpstring*);
extern ddpstring* _ddp_deep_copy_string(ddpstring*);

static ddpint clamp(ddpint i, ddpint min, ddpint max) {
  const ddpint t = i < min ? min : i;
  return t > max ? max : t;
}

{{range .}}
ddpbool _ddp_{{ .T }}_equal({{ .T }}* list1, {{ .T }}* list2) {
	if (list1 == list2) return true;
	if (list1->len != list2->len) return false; // if the length is different, it's a quick false return
	{{if .D}}
	for (size_t i = 0; i < list1->len; i++) {
		if (!_ddp_string_equal(list1->arr[i], list2->arr[i])) return false;
	}
	return true;
	{{else}}
	return memcmp(list1->arr, list2->arr, sizeof({{ .E }}) * list1->len) == 0;
	{{end}}
}

{{ .T }}* _ddp_{{ .T }}_slice({{ .T }}* list, ddpint index1, ddpint index2) {
	DBGLOG("_ddp_{{ .T }}_slice: %p", list);

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
	{{ .E }}* arr = ALLOCATE({{ .E }}, new_list_cap);

	{{if .D}}
	size_t j = 0;
	for (size_t i = index1; i <= index2 && i < list->len; i++, j++) {
		arr[j] = _ddp_deep_copy_string(list->arr[i]);
		_ddp_increment_ref_count(arr[j], VK_STRING);
	}
	{{else}}
	memcpy(arr, &list->arr[index1], sizeof({{ .E }}) * new_list_len);
	{{end}}

	FREE_ARRAY({{ .E }}, list->arr, list->cap);
	list->arr = arr;
	list->len = new_list_len;
	list->cap = new_list_cap;
	return list;
}

{{ .T }}* _ddp_{{ .T }}_{{ .T }}_verkettet({{ .T }}* list1, {{ .T }}* list2) {
	DBGLOG("_ddp_{{ .T }}_{{ .T }}_verkettet: %p", list1);

	size_t new_len = list1->len + list2->len;
	size_t new_cap = list1->cap;
	while (new_cap < new_len) new_cap = GROW_CAPACITY(new_cap);
	list1->arr = reallocate(list1->arr, sizeof({{ .E }}) * list1->cap, sizeof({{ .E }}) * new_cap);
	{{if .D}}
	for (size_t i = 0; i < list2->len; i++) {
		list1->arr[i+list1->len] = _ddp_deep_copy_string(list2->arr[i]);
		_ddp_increment_ref_count(list1->arr[i+list1->len], VK_STRING);
	}
	{{else}}
	memcpy(&list1->arr[list1->len], list2->arr, sizeof({{ .E }}) * list2->len);
	{{end}}

	list1->len = new_len;
	list1->cap = new_cap;
	return list1;
}
{{ .T }}* _ddp_{{ .T }}_{{ .E }}_verkettet({{ .T }}* list, {{ .E }} el) {
	DBGLOG("_ddp_{{ .T }}_{{ .E }}_verkettet: %p", list);

	size_t new_len = list->len + 1;
	size_t new_cap = list->cap;
	while (new_cap < new_len) new_cap = GROW_CAPACITY(new_cap);
	list->arr = reallocate(list->arr, sizeof({{ .E }}) * list->cap, sizeof({{ .E }}) * new_cap);
	{{if .D}}
	list->arr[list->len] = _ddp_deep_copy_string(el);
	_ddp_increment_ref_count(list->arr[list->len], VK_STRING);
	{{else}}
	list->arr[list->len] = el;
	{{end}}

	list->len = new_len;
	list->cap = new_cap;
	return list;
}
{{if .D}}
{{ .T }}* _ddp_ddpstring_ddpstringlist_verkettet({{ .E }} str, {{ .T }}* list) {
	DBGLOG("_ddp_ddpstring_ddpstringlist_verkettet: %p", list);

	size_t new_len = list->len + 1;
	size_t new_cap = list->cap;
	while (new_cap < new_len) new_cap = GROW_CAPACITY(new_cap);
	list->arr = reallocate(list->arr, sizeof({{ .E }}) * list->cap, sizeof({{ .E }}) * new_cap);
	memmove(&list->arr[1], list->arr, sizeof(ddpstring*) * list->len);
	list->arr[0] = _ddp_deep_copy_string(str);
	_ddp_increment_ref_count(list->arr[0], VK_STRING);

	list->len = new_len;
	list->cap = new_cap;
	return list;
}
{{else}}
{{ .T }}* _ddp_{{ .E }}_{{ .E }}_verkettet({{ .E }} el1, {{ .E }} el2) {
	{{ .T }}* newList = ALLOCATE({{ .T }}, 1); // up here to log the adress in debug mode
	newList->len = 2;
	newList->cap = GROW_CAPACITY(newList->len);
	newList->arr = ALLOCATE({{ .E }}, newList->cap);
	DBGLOG("_ddp_{{ .E }}_{{ .E }}_verkettet: %p", newList);

	newList->arr[0] = el1;
	newList->arr[1] = el2;

	return newList;
}
{{ .T }}* _ddp_{{ .E }}_{{ .T }}_verkettet({{ .E }} el, {{ .T }}* list) {
	DBGLOG("_ddp_{{ .E }}_{{ .T }}_verkettet: %p", list);
	
	size_t new_len = list->len + 1;
	size_t new_cap = list->cap;
	while (new_cap < new_len) new_cap = GROW_CAPACITY(new_cap);
	list->arr = reallocate(list->arr, sizeof({{ .E }}) * list->cap, sizeof({{ .E }}) * new_cap);
	memmove(&list->arr[1], list->arr, sizeof({{ .E }}) * list->len);
	list->arr[0] = el;

	list->len = new_len;
	list->cap = new_cap;
	return list;
}
{{end}}
{{end}}