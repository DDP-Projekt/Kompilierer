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

	c.declareInbuiltFunction("inbuilt_{{ .T }}_{{ .T }}_verkettet", {{ .T }}ptr, ir.NewParam("list1", {{ .T }}ptr), ir.NewParam("list2", {{ .T }}ptr))
	c.declareInbuiltFunction("inbuilt_{{ .T }}_{{if .D}}ddpstring{{else}}{{ .E }}{{end}}_verkettet", {{ .T}}ptr, ir.NewParam("list", {{ .T }}ptr), ir.NewParam("el", {{if .D}}ddpstrptr{{else}}{{ .E }}{{end}}))
	{{if .D}}
	c.declareInbuiltFunction("inbuilt_ddpstring_ddpstringlist_verkettet", ddpstringlistptr, ir.NewParam("str", ddpstrptr), ir.NewParam("list", ddpstringlistptr))
	{{else}}
	c.declareInbuiltFunction("inbuilt_{{ .E }}_{{ .E }}_verkettet", {{ .T }}ptr, ir.NewParam("el1", {{ .E }}), ir.NewParam("el2", {{ .E }}))
	c.declareInbuiltFunction("inbuilt_{{ .E }}_{{ .T }}_verkettet", {{ .T }}ptr, ir.NewParam("el", {{ .E }}), ir.NewParam("list", {{ .T }}ptr))
	{{end}}
	{{end}}
}


// generated verkettet switch
{{range .}}
case {{ .T }}ptr:
	switch rhs.Type() {
	case {{ .T }}ptr:
		c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_{{ .T }}_{{ .T }}_verkettet"].irFunc, lhs, rhs)
	case {{if .D}} ddpstrptr {{else}} {{ .E }} {{end}}:
		c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_{{ .T }}_{{if .D}}ddpstring{{else}}{{ .E }}{{end}}_verkettet"].irFunc, lhs, rhs)
	default:
		err("invalid Parameter Types for VERKETTET (%s, %s)", lhs.Type(), rhs.Type())
	}
{{if .D}}
case ddpstrptr:
	switch rhs.Type() {
	case ddpstrptr:
		c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_string_string_verkettet"].irFunc, lhs, rhs)
	case ddpchar:
		c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_string_char_verkettet"].irFunc, lhs, rhs)
	case ddpstringlistptr:
		c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_ddpstring_ddpstringlist_verkettet"].irFunc, lhs, rhs)
	default:
		err("invalid Parameter Types for VERKETTET (%s, %s)", lhs.Type(), rhs.Type())
	}
{{else}}
case {{ .E }}:
	switch rhs.Type() {
	case {{ .E }}:
		c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_{{ .E }}_{{ .E }}_verkettet"].irFunc, lhs, rhs)
	case {{ .T }}ptr:
		c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_{{ .E }}_{{ .T }}_verkettet"].irFunc, lhs, rhs)
	default:
		err("invalid Parameter Types for VERKETTET (%s, %s)", lhs.Type(), rhs.Type())
	}
{{end}}
{{end}}
default:
	err("invalid Parameter Types for VERKETTET (%s, %s)", lhs.Type(), rhs.Type())