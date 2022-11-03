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
	{{ .T }}* new_list = ALLOCATE({{ .T }}, 1);
	DBGLOG("_ddp_{{ .T }}_slice: %p", new_list);
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
	new_list->arr = ALLOCATE({{ .E }}, new_list->cap);

	{{if .D}}
	size_t j = 0;
	for (size_t i = index1; i <= index2 && i < list->len; i++, j++) {
		new_list->arr[j] = _ddp_deep_copy_string(list->arr[i]);
	}
	{{else}}
	memcpy(new_list->arr, &list->arr[index1], sizeof({{ .E }}) * new_list->len);
	{{end}}

	return new_list;
}

{{ .T }}* _ddp_{{ .T }}_{{ .T }}_verkettet({{ .T }}* list1, {{ .T }}* list2) {
	{{ .T }}* new_list = ALLOCATE({{ .T }}, 1);
	DBGLOG("_ddp_{{ .T }}_{{ .T }}_verkettet: %p", new_list);

	new_list->len = list1->len + list2->len;
	new_list->cap = list1->cap;
	while (new_list->cap < new_list->len) new_list->cap = GROW_CAPACITY(new_list->cap);
	new_list->arr = reallocate(list1->arr, sizeof({{ .E }}) * list1->cap, sizeof({{ .E }}) * new_list->cap);
	{{if .D}}
	for (size_t i = 0; i < list2->len; i++) {
		new_list->arr[i+list1->len] = _ddp_deep_copy_string(list2->arr[i]);
	}
	{{else}}
	memcpy(&new_list->arr[list1->len], list2->arr, sizeof({{ .E }}) * list2->len);
	{{end}}

	list1->len = 0;
	list1->cap = 0;
	list1->arr = NULL;
	return new_list;
}
{{ .T }}* _ddp_{{ .T }}_{{ .E }}_verkettet({{ .T }}* list, {{ .E }} el) {
	{{ .T }}* new_list = ALLOCATE({{ .T }}, 1);
	DBGLOG("_ddp_{{ .T }}_{{ .E }}_verkettet: %p", new_list);

	new_list->len = list->len + 1;
	new_list->cap = list->cap;
	while (new_list->cap < new_list->len) new_list->cap = GROW_CAPACITY(new_list->cap);
	new_list->arr = reallocate(list->arr, sizeof({{ .E }}) * list->cap, sizeof({{ .E }}) * new_list->cap);
	{{if .D}}
	new_list->arr[list->len] = _ddp_deep_copy_string(el);
	{{else}}
	new_list->arr[list->len] = el;
	{{end}}

	list->len = 0;
	list->cap = 0;
	list->arr = NULL;
	return new_list;
}
{{if .D}}
{{ .T }}* _ddp_ddpstring_ddpstringlist_verkettet({{ .E }} str, {{ .T }}* list) {
	{{ .T }}* new_list = ALLOCATE({{ .T }}, 1);
	DBGLOG("_ddp_ddpstring_ddpstringlist_verkettet: %p", new_list);

	new_list->len = list->len + 1;
	new_list->cap = list->cap;
	while (new_list->cap < new_list->len) new_list->cap = GROW_CAPACITY(new_list->cap);
	new_list->arr = reallocate(list->arr, sizeof({{ .E }}) * list->cap, sizeof({{ .E }}) * new_list->cap);
	memmove(&new_list->arr[1], new_list->arr, sizeof(ddpstring*) * list->len);
	new_list->arr[0] = _ddp_deep_copy_string(str);

	list->len = 0;
	list->cap = 0;
	list->arr = NULL;
	return new_list;
}
{{else}}
{{ .T }}* _ddp_{{ .E }}_{{ .E }}_verkettet({{ .E }} el1, {{ .E }} el2) {
	{{ .T }}* new_list = ALLOCATE({{ .T }}, 1); // up here to log the adress in debug mode
	new_list->len = 2;
	new_list->cap = GROW_CAPACITY(new_list->len);
	new_list->arr = ALLOCATE({{ .E }}, new_list->cap);
	DBGLOG("_ddp_{{ .E }}_{{ .E }}_verkettet: %p", new_list);

	new_list->arr[0] = el1;
	new_list->arr[1] = el2;

	return new_list;
}
{{ .T }}* _ddp_{{ .E }}_{{ .T }}_verkettet({{ .E }} el, {{ .T }}* list) {
	{{ .T }}* new_list = ALLOCATE({{ .T }}, 1);
	DBGLOG("_ddp_{{ .E }}_{{ .T }}_verkettet: %p", new_list);
	
	new_list->len = list->len + 1;
	new_list->cap = list->cap;
	while (new_list->cap < new_list->len) new_list->cap = GROW_CAPACITY(new_list->cap);
	new_list->arr = reallocate(list->arr, sizeof({{ .E }}) * list->cap, sizeof({{ .E }}) * new_list->cap);
	memmove(&new_list->arr[1], new_list->arr, sizeof({{ .E }}) * list->len);
	new_list->arr[0] = el;

	list->len = 0;
	list->cap = 0;
	list->arr = NULL;
	return new_list;
}
{{end}}
{{end}}