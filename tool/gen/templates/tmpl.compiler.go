// declares some internal list functions
// and completes the ddp<type>list structs
func (c *Compiler) setupListTypes() {
	{{range .}}
	// complete the {{ .T }} definition to interact with the c ddp runtime
	{{ .T }}.Fields = make([]types.Type, 3)
	{{ .T }}.Fields[0] = {{if .D}}ptr(ddpstrptr){{else}}ptr({{ .E }}){{end}}
	{{ .T }}.Fields[1] = ddpint
	{{ .T }}.Fields[2] = ddpint
	c.mod.NewTypeDef("{{ .T }}", {{ .T }})

	// creates a {{ .T }} from the elements and returns a pointer to it
	// the caller is responsible for calling increment_ref_count on this pointer
	c.declareInbuiltFunction("inbuilt_{{ .T }}_from_constants", {{ .T }}ptr, ir.NewParam("count", ddpint))
	c.functions["inbuilt_{{ .T }}_from_constants"].irFunc.Sig.Variadic = true

	// returns a copy of the passed string as a new pointer
	// the caller is responsible for calling increment_ref_count on this pointer
	c.declareInbuiltFunction("inbuilt_deep_copy_{{ .T }}", {{ .T }}ptr, ir.NewParam("list", {{ .T }}ptr))

	// inbuilt operators for lists
	c.declareInbuiltFunction("inbuilt_{{ .T }}_equal", ddpbool, ir.NewParam("list1", {{ .T }}ptr), ir.NewParam("list2", {{ .T }}ptr))
	c.declareInbuiltFunction("inbuilt_{{ .T }}_slice", {{ .T }}ptr, ir.NewParam("list", {{ .T }}ptr), ir.NewParam("index1", ddpint), ir.NewParam("index2", ddpint))
	c.declareInbuiltFunction("inbuilt_{{ .T }}_to_string", ddpstrptr, ir.NewParam("list", {{ .T }}ptr))
	{{end}}
}