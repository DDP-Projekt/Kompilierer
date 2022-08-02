extern ddpbool inbuilt_string_equal(ddpstring*, ddpstring*);
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
{{end}}