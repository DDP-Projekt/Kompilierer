// generate_list_code.go generates code for ddplists
// to be copy-pasted into the according files
package main

import (
	"os"
	"text/template"
)

type DDP_List struct {
	T string // Typename
	E string // ElementTypeName
	D bool   // Dynamically allocated (string)
}

var List_Types = []DDP_List{
	{"ddpintlist", "ddpint", false},
	{"ddpfloatlist", "ddpfloat", false},
	{"ddpboollist", "ddpbool", false},
	{"ddpcharlist", "ddpchar", false},
	{"ddpstringlist", "ddpstring*", true},
}

func generate(fileName string) {
	templ := template.Must(template.New("tmpl." + fileName).ParseFiles("./templates/tmpl." + fileName))
	f, err := os.OpenFile("./generated/gen."+fileName, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := templ.Execute(f, List_Types); err != nil {
		panic(err)
	}
}

func main() {
	if len(os.Args) > 1 {
		for i := 1; i < len(os.Args); i++ {
			generate(os.Args[i])
		}
	} else {
		generate("ddptypes.h")
		generate("ddptypes.c")
		generate("compiler.go")
	}
}
