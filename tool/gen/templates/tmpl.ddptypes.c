{{range .}}
{{ .T }}* _ddp_{{ .T }}_from_constants(ddpint count) {
	{{ .T }}* list = ALLOCATE({{ .T }}, 1); // up here to log the adress in debug mode
	DBGLOG("{{ .T }}_from_constants: %p", list);
	list->arr = count > 0 ? ALLOCATE({{ .E }}, count) : NULL; // the element array of the list
	list->len = count;
	list->cap = count;
	return list;
}

void _ddp_free_{{ .T }}({{ .T }}* list) {
	DBGLOG("free_{{ .T }}: %p", list);
	{{if .D}}
	for (size_t i = 0; i < list->len; i++) {
		_ddp_free_string(list->arr[i]);
	}
	{{end}}
	FREE_ARRAY({{ .E }}, list->arr, list->cap); // free the element array
	FREE({{ .T }}, list); // free the list pointer
}

{{ .T }}* _ddp_deep_copy_{{ .T }}({{ .T }}* list) {
	DBGLOG("_ddp_deep_copy_{{ .T }}: %p", list);
	{{ .E }}* cpy = ALLOCATE({{ .E }}, {{ if .D}}list->len{{else}}list->cap{{end}}); // allocate the element array for the copy
	{{ if not .D }}
	memcpy(cpy, list->arr, sizeof({{ .E }}) * list->cap); // copy the elements
	{{end}}
	{{if .D}}
	for (size_t i = 0; i < list->len; i++) {
		cpy[i] = _ddp_deep_copy_string(list->arr[i]);
	}
	{{end}}
	{{ .T }}* cpylist = ALLOCATE({{ .T }}, 1); // alocate the copy list
	// set the fields of the copy
	cpylist->arr = cpy;
	cpylist->len = list->len;
	cpylist->cap = {{ if .D}}list->len{{else}}list->cap{{end}};
	return cpylist;
}
{{end}}