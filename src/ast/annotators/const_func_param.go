package annotators

import (
	"fmt"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"golang.org/x/exp/maps"
)

const ConstFuncParamMetaKind ast.MetadataKind = "ConstFuncParam"

type ConstFuncParamMeta struct {
	// wether each parameter is const
	IsConst map[string]bool
}

var _ ast.MetadataAttachment = (*ConstFuncParamMeta)(nil)

func (m ConstFuncParamMeta) String() string {
	return fmt.Sprintf("ConstFuncParamMeta[%v]", m.IsConst)
}

func (m ConstFuncParamMeta) Kind() ast.MetadataKind {
	return ConstFuncParamMetaKind
}

type ConstFuncParamAnnotator struct {
	ast.BaseVisitor
	// wether the currently tracked parameters are still considered constant
	currentParams map[*ast.VarDecl]bool
	currentDecl   *ast.FuncDecl
}

var (
	_ ast.Annotator          = (*ConstFuncParamAnnotator)(nil)
	_ ast.FuncDeclVisitor    = (*ConstFuncParamAnnotator)(nil)
	_ ast.FuncCallVisitor    = (*ConstFuncParamAnnotator)(nil)
	_ ast.AssignStmtVisitor  = (*ConstFuncParamAnnotator)(nil)
	_ ast.ConditionalVisitor = (*ConstFuncParamAnnotator)(nil)
)

func (a *ConstFuncParamAnnotator) ShouldVisit(node ast.Node) bool {
	switch node.(type) {
	case *ast.FuncDecl, *ast.DeclStmt:
		return true
	default:
		return a.currentDecl != nil
	}
}

func (a *ConstFuncParamAnnotator) VisitFuncDecl(decl *ast.FuncDecl) ast.VisitResult {
	a.currentDecl = nil
	// if the function is extern, we have to assume that the parameters are not const
	if ast.IsExternFunc(decl) {
		attachement := ConstFuncParamMeta{
			IsConst: make(map[string]bool, len(decl.Parameters)),
		}
		for _, param := range decl.Parameters {
			attachement.IsConst[param.Name.Literal] = false
		}
		a.CurrentModule.Ast.AddAttachement(decl, attachement)
		return ast.VisitSkipChildren
	}

	a.currentParams = make(map[*ast.VarDecl]bool, len(decl.Parameters))
	attachement := ConstFuncParamMeta{
		IsConst: make(map[string]bool, len(decl.Parameters)),
	}
	// track all the function parameters
	// and initially assume they are const
	body := decl.Body
	if ast.IsForwardDecl(decl) {
		body = decl.Def.Body
	}
	for _, funcParam := range decl.Parameters {
		param, exists, isVar := body.Symbols.LookupDecl(funcParam.Name.Literal)
		if exists && isVar {
			a.currentParams[param.(*ast.VarDecl)] = true
			attachement.IsConst[funcParam.Name.Literal] = true
		}
	}
	a.CurrentModule.Ast.AddAttachement(decl, attachement)
	a.currentDecl = decl

	return ast.VisitRecurse
}

func (a *ConstFuncParamAnnotator) VisitFuncCall(call *ast.FuncCall) ast.VisitResult {
	var isConst map[string]bool
	if attachement, ok := a.CurrentModule.Ast.GetMetadataByKind(call.Func, ConstFuncParamMetaKind); ok {
		isConst = attachement.(ConstFuncParamMeta).IsConst
	}

	currentParams := maps.Keys(a.currentParams)
	for _, param := range call.Func.Parameters {
		if isConst[param.Name.Literal] {
			continue
		}

		for _, referencedVar := range doesReferenceVarMutable(call.Args[param.Name.Literal], currentParams) {
			a.currentParams[referencedVar] = false
		}
	}

	a.overwriteAttachement()

	return ast.VisitRecurse
}

func (a *ConstFuncParamAnnotator) VisitAssignStmt(stmt *ast.AssignStmt) ast.VisitResult {
	currentParams := maps.Keys(a.currentParams)
	for _, referencedVar := range doesReferenceVarMutable(stmt.Var, currentParams) {
		a.currentParams[referencedVar] = false
	}

	a.overwriteAttachement()

	return ast.VisitRecurse
}

// checks if the given expr might mutate any of the given variables
// returns the referenced variables
func doesReferenceVarMutable(expr ast.Expression, decls []*ast.VarDecl) []*ast.VarDecl {
	ass, ok := expr.(ast.Assigneable)
	if !ok {
		return nil
	}

	switch ass := ass.(type) {
	case *ast.Ident:
		for _, decl := range decls {
			if decl == ass.Declaration {
				return []*ast.VarDecl{decl}
			}
		}
		return nil
	case *ast.Indexing:
		return doesReferenceVarMutable(ass.Lhs, decls)
	case *ast.FieldAccess:
		return doesReferenceVarMutable(ass.Rhs, decls)
	default:
		return decls // better safe than sorry
	}
}

func (a *ConstFuncParamAnnotator) overwriteAttachement() {
	attachement := ConstFuncParamMeta{
		IsConst: make(map[string]bool, len(a.currentParams)),
	}
	// overwrite the attachement
	if att, ok := a.CurrentModule.Ast.GetMetadataByKind(a.currentDecl, ConstFuncParamMetaKind); ok {
		attachement = att.(ConstFuncParamMeta)
	}

	for param := range a.currentParams {
		attachement.IsConst[param.Name()] = a.currentParams[param]
	}
	a.CurrentModule.Ast.AddAttachement(a.currentDecl, attachement)
}
