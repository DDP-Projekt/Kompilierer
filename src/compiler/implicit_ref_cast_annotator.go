package compiler

import (
	"fmt"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
	"github.com/DDP-Projekt/Kompilierer/src/parser/typechecker"
)

type ImplicitRefCastMeta struct {
	FromRef bool // if false, the cast is to ref
}

var _ ast.MetadataAttachment = ImplicitRefCastMeta{}

func (m ImplicitRefCastMeta) String() string {
	return fmt.Sprintf("ImplicitRefCastMeta(Deref: %v, Promote: %v)", m.FromRef, !m.FromRef)
}

const ImplicitRefCastMetaKind ast.MetadataKind = "DDP_ImplicitRefCastMeta"

func (m ImplicitRefCastMeta) Kind() ast.MetadataKind {
	return ImplicitRefCastMetaKind
}

type ImplicitRefCastAnnotator struct {
	ast.BaseVisitor
}

var (
	_ ast.Annotator = (*ImplicitRefCastAnnotator)(nil)

	// _ ast.ConstDeclVisitor = (*ImplicitRefCastAnnotator)(nil)
	_ ast.VarDeclVisitor = (*ImplicitRefCastAnnotator)(nil)

	_ ast.IdentVisitor         = (*ImplicitRefCastAnnotator)(nil)
	_ ast.ListLitVisitor       = (*ImplicitRefCastAnnotator)(nil)
	_ ast.UnaryExprVisitor     = (*ImplicitRefCastAnnotator)(nil)
	_ ast.BinaryExprVisitor    = (*ImplicitRefCastAnnotator)(nil)
	_ ast.TernaryExprVisitor   = (*ImplicitRefCastAnnotator)(nil)
	_ ast.FuncCallVisitor      = (*ImplicitRefCastAnnotator)(nil)
	_ ast.StructLiteralVisitor = (*ImplicitRefCastAnnotator)(nil)

	_ ast.AssignStmtVisitor   = (*ImplicitRefCastAnnotator)(nil)
	_ ast.IfStmtVisitor       = (*ImplicitRefCastAnnotator)(nil)
	_ ast.WhileStmtVisitor    = (*ImplicitRefCastAnnotator)(nil)
	_ ast.ForStmtVisitor      = (*ImplicitRefCastAnnotator)(nil)
	_ ast.ForRangeStmtVisitor = (*ImplicitRefCastAnnotator)(nil)
	_ ast.ReturnStmtVisitor   = (*ImplicitRefCastAnnotator)(nil)
)

func (a *ImplicitRefCastAnnotator) typeOf(expr ast.Expression) ddptypes.Type {
	return typechecker.TypeOfTypecheckedExpression(a.CurrentModule.Ast, expr)
}

func (a *ImplicitRefCastAnnotator) clearAnnotation(node ast.Node) {
	a.CurrentModule.Ast.RemoveAttachment(node, ImplicitRefCastMetaKind)
}

func (a *ImplicitRefCastAnnotator) annotateFromRef(node ast.Node) {
	if node == nil {
		return
	}
	a.CurrentModule.Ast.AddAttachement(node, ImplicitRefCastMeta{FromRef: true})
}

func (a *ImplicitRefCastAnnotator) annotateToRef(node ast.Node) {
	if node == nil {
		return
	}
	a.CurrentModule.Ast.AddAttachement(node, ImplicitRefCastMeta{FromRef: false})
}

func (a *ImplicitRefCastAnnotator) annotateDeref(expr ast.Expression) {
	if expr == nil {
		return
	}

	if ddptypes.IsReference(a.typeOf(expr)) {
		a.annotateFromRef(expr)
	}
}

func (a *ImplicitRefCastAnnotator) VisitVarDecl(d *ast.VarDecl) ast.VisitResult {
	a.Visit(d.InitVal)
	a.clearAnnotation(d.InitVal)
	if ddptypes.IsReferenceTo(d.Type, d.InitType) {
		a.annotateToRef(d.InitVal)
	} else if ddptypes.IsReferenceTo(d.InitType, d.Type) {
		a.annotateFromRef(d.InitVal)
	}
	return ast.VisitSkipChildren
}

func (a *ImplicitRefCastAnnotator) VisitIdent(e *ast.Ident) ast.VisitResult {
	if varDecl, ok := e.Declaration.(*ast.VarDecl); ok && !ddptypes.IsReference(varDecl.Type) {
		a.annotateFromRef(e)
	}
	return ast.VisitRecurse
}

func (a *ImplicitRefCastAnnotator) VisitListLit(e *ast.ListLit) ast.VisitResult {
	if e.Values != nil {
		for _, v := range e.Values {
			t := a.typeOf(v)
			if ddptypes.IsReferenceTo(e.Type.ElementType, t) {
				a.annotateToRef(v)
			} else if ddptypes.IsReferenceTo(t, e.Type.ElementType) {
				a.annotateFromRef(v)
			}
		}
	} else if e.Count != nil && e.Value != nil {
		a.annotateDeref(e.Count)
	}
	return ast.VisitRecurse
}

func (a *ImplicitRefCastAnnotator) VisitUnaryExpr(e *ast.UnaryExpr) ast.VisitResult {
	if e.OverloadedBy != nil {
		// TODO
		return ast.VisitRecurse
	}
	a.annotateDeref(e.Rhs)
	return ast.VisitRecurse
}

func (a *ImplicitRefCastAnnotator) VisitBinaryExpr(e *ast.BinaryExpr) ast.VisitResult {
	if e.OverloadedBy != nil {
		// TODO
		return ast.VisitRecurse
	}
	a.annotateDeref(e.Lhs)
	a.annotateDeref(e.Rhs)
	return ast.VisitRecurse
}

func (a *ImplicitRefCastAnnotator) VisitTernaryExpr(e *ast.TernaryExpr) ast.VisitResult {
	if e.OverloadedBy != nil {
		// TODO
		return ast.VisitRecurse
	}
	a.annotateDeref(e.Lhs)
	a.annotateDeref(e.Mid)
	a.annotateDeref(e.Rhs)
	return ast.VisitRecurse
}

func (a *ImplicitRefCastAnnotator) VisitFuncCall(e *ast.FuncCall) ast.VisitResult {
	for k, expr := range e.Args {
		var paramType ddptypes.Type
		for _, param := range e.Func.Parameters {
			if param.Name.Literal == k {
				paramType = param.Type
				break
			}
		}

		if ddptypes.IsReferenceTo(paramType, a.typeOf(expr)) {
			a.annotateToRef(expr)
		} else if ddptypes.IsReferenceTo(a.typeOf(expr), paramType) {
			a.annotateFromRef(expr)
		}
	}
	return ast.VisitRecurse
}

func (a *ImplicitRefCastAnnotator) VisitStructLiteral(e *ast.StructLiteral) ast.VisitResult {
	for k, expr := range e.Args {
		var paramType ddptypes.Type
		for _, field := range e.Type.Fields {
			if field.Name == k {
				paramType = field.Type
				break
			}
		}

		if ddptypes.IsReferenceTo(paramType, a.typeOf(expr)) {
			a.annotateToRef(expr)
		} else if ddptypes.IsReferenceTo(a.typeOf(expr), paramType) {
			a.annotateFromRef(expr)
		}
	}
	return ast.VisitRecurse
}

func (a *ImplicitRefCastAnnotator) VisitAssignStmt(s *ast.AssignStmt) ast.VisitResult {
	if ddptypes.IsReferenceTo(s.RhsType, s.VarType) {
		a.annotateFromRef(s.Rhs)
	}

	a.Visit(s.Var)
	a.Visit(s.Rhs)
	a.clearAnnotation(s.Rhs)
	// if ddptypes.IsReferenceTo(s.VarType, s.RhsType) {
	// 	a.annotateToRef(s.Rhs)
	// } else
	if ddptypes.IsReferenceTo(s.RhsType, s.VarType) {
		a.annotateFromRef(s.Rhs)
	}
	return ast.VisitSkipChildren
}

func (a *ImplicitRefCastAnnotator) VisitIfStmt(s *ast.IfStmt) ast.VisitResult {
	a.annotateDeref(s.Condition)
	return ast.VisitRecurse
}

func (a *ImplicitRefCastAnnotator) VisitWhileStmt(s *ast.WhileStmt) ast.VisitResult {
	a.annotateDeref(s.Condition)
	return ast.VisitRecurse
}

func (a *ImplicitRefCastAnnotator) VisitForStmt(s *ast.ForStmt) ast.VisitResult {
	a.annotateDeref(s.To)
	a.annotateDeref(s.StepSize)
	return ast.VisitRecurse
}

func (a *ImplicitRefCastAnnotator) VisitForRangeStmt(s *ast.ForRangeStmt) ast.VisitResult {
	a.annotateDeref(s.In)
	return ast.VisitRecurse
}

func (a *ImplicitRefCastAnnotator) VisitReturnStmt(s *ast.ReturnStmt) ast.VisitResult {
	t := a.typeOf(s.Value)
	if ddptypes.IsReferenceTo(s.Func.ReturnType, t) {
		a.annotateToRef(s.Value)
	} else if ddptypes.IsReferenceTo(t, s.Func.ReturnType) {
		a.annotateFromRef(s.Value)
	}
	return ast.VisitRecurse
}
