extern void inbuilt_decrement_ref_count(void*);
extern void inbuilt_increment_ref_count(void*, uint8_t);
{{range .}}
{{ .T }}* inbuilt_{{ .T }}_from_constants(ddpint count, ...) {
	{{ .T }}* list = ALLOCATE({{ .T }}, 1); // up here to log the adress in debug mode
	DBGLOG("{{ .T }}_from_constants: %p", list);
	{{ .E }}* arr = ALLOCATE({{ .E }}, count); // the element array of the list
	
	va_list elements;
	va_start(elements, count);

	for (size_t i = 0; i < count; i++) {
		arr[i] = va_arg(elements, {{ .E }});
		{{if .D}}
		inbuilt_increment_ref_count(arr[i], VK_STRING);
		{{end}}
	}

	va_end(elements);

	// set the list fields
	list->arr = arr;
	list->len = count;
	list->cap = count;
	return list;
}

void free_{{ .T }}({{ .T }}* list) {
	DBGLOG("free_{{ .T }}: %p", list);
	{{if .D}}
	for (size_t i = 0; i < list->len; i++) {
		inbuilt_decrement_ref_count(list->arr[i]);
	}
	{{end}}
	FREE_ARRAY({{ .E }}, list->arr, list->cap); // free the element array
	FREE({{ .T }}, list); // free the list pointer
}

{{ .T }}* inbuilt_deep_copy_{{ .T }}({{ .T }}* list) {
	DBGLOG("inbuilt_deep_copy_{{ .T }}: %p", list);
	{{ .E }}* cpy = ALLOCATE({{ .E }}, list->cap); // allocate the element array for the copy
	memcpy(cpy, list->arr, list->cap); // copy the chars
	{{if .D}}
	for (size_t i = 0; i < list->len; i++) {
		cpy[i] = inbuilt_deep_copy_string(cpy[i]);
		inbuilt_increment_ref_count(cpy[i], VK_STRING);
	}
	{{end}}
	{{ .T }}* cpylist = ALLOCATE({{ .T }}, 1); // alocate the copy list
	// set the fields of the copy
	cpylist->arr = cpy;
	cpylist->len = list->len;
	cpylist->cap = list->cap;
	return cpylist;
}
{{end}}