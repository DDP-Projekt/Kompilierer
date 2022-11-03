{{range .}}
typedef struct {
	{{ .E }}* arr; // the element array
	ddpint len; // the length of the array
	ddpint cap; // the capacity of the array
} {{ .T }};

void _ddp_free_{{ .T }}({{ .T }}* list);
{{end}}