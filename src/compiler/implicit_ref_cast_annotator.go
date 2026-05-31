package compiler

import (
	"fmt"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
	"github.com/DDP-Projekt/Kompilierer/src/parser/typechecker"
)

type ImplicitRefCastMeta struct {
	// n > 0 -> needs n dereferences
	// n < 0 -> needs abs(n) upcasts
	// n == 0 -> needs no casts
	n int
}

func (i ImplicitRefCastMeta) FromRef() bool {
	return i.n > 0
}

func (i ImplicitRefCastMeta) ToRef() bool {
	return i.n < 0
}

func (i ImplicitRefCastMeta) DerefLevel() uint {
	if i.FromRef() {
		return uint(i.n)
	}
	return 0
}

func (i ImplicitRefCastMeta) UpcastLevel() uint {
	if i.ToRef() {
		return uint(0 - i.n)
	}
	return 0
}

var _ ast.MetadataAttachment = ImplicitRefCastMeta{}

func (m ImplicitRefCastMeta) String() string {
	return fmt.Sprintf("ImplicitRefCastMeta(%d)", m.n)
}

const ImplicitRefCastMetaKind ast.MetadataKind = "DDP_ImplicitRefCastMeta"

func (m ImplicitRefCastMeta) Kind() ast.MetadataKind {
	return ImplicitRefCastMetaKind
}

type ImplicitRefCastAnnotator struct {
	ast.BaseVisitor
	visitedGenericInstantiations map[*ast.FuncDecl]struct{}
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
	_ ast.CastExprVisitor      = (*ImplicitRefCastAnnotator)(nil)
	_ ast.FuncCallVisitor      = (*ImplicitRefCastAnnotator)(nil)
	_ ast.StructLiteralVisitor = (*ImplicitRefCastAnnotator)(nil)

	_ ast.AssignStmtVisitor   = (*ImplicitRefCastAnnotator)(nil)
	_ ast.IfStmtVisitor       = (*ImplicitRefCastAnnotator)(nil)
	_ ast.WhileStmtVisitor    = (*ImplicitRefCastAnnotator)(nil)
	_ ast.ForStmtVisitor      = (*ImplicitRefCastAnnotator)(nil)
	_ ast.ForRangeStmtVisitor = (*ImplicitRefCastAnnotator)(nil)
	_ ast.ReturnStmtVisitor   = (*ImplicitRefCastAnnotator)(nil)
)

func (a *ImplicitRefCastAnnotator) clearAnnotation(node ast.Node) {
	if node == nil {
		return
	}
	node.RemoveMetadataAttachment(ImplicitRefCastMetaKind)
}

func (a *ImplicitRefCastAnnotator) annotateFromRef(node ast.Node, n uint) {
	if node == nil {
		return
	}
	node.SetMetadataAttachement(ImplicitRefCastMeta{n: int(n)})
}

func (a *ImplicitRefCastAnnotator) annotateToRef(node ast.Node, n uint) {
	// commented out because we don't want implicit up-casts
	if node == nil {
		return
	}
	node.SetMetadataAttachement(ImplicitRefCastMeta{n: -int(n)})
}

func (a *ImplicitRefCastAnnotator) annotateDeref(expr ast.Expression) {
	if expr == nil {
		return
	}

	if _, _, level := ddptypes.CastReferenceLevel(expr.Type()); level > 0 {
		a.annotateFromRef(expr, level)
	}
}

func (a *ImplicitRefCastAnnotator) VisitVarDecl(d *ast.VarDecl) ast.VisitResult {
	a.Visit(d.InitVal)
	a.clearAnnotation(d.InitVal)
	initDDPType := ddptypes.Type(nil)
	if d.InitVal != nil {
		initDDPType = d.InitVal.Type()
	}
	if level := ddptypes.IsReferenceToLevel(d.Type, initDDPType); level > 0 && d.InitVal != nil && !d.InitVal.HasMetadata(typechecker.AssigneableMetaKind) {
		a.annotateToRef(d.InitVal, level)
	} else if level := ddptypes.IsReferenceToLevel(initDDPType, d.Type); level > 0 {
		a.annotateFromRef(d.InitVal, level)
	}
	return ast.VisitSkipChildren
}

func (a *ImplicitRefCastAnnotator) VisitIdent(e *ast.Ident) ast.VisitResult {
	if varDecl, ok := e.Declaration.(*ast.VarDecl); ok && !ddptypes.IsReference(varDecl.Type) {
		a.annotateFromRef(e, 1)
	}
	return ast.VisitRecurse
}

// TODO: visit children manually and clear annotations
func (a *ImplicitRefCastAnnotator) VisitListLit(e *ast.ListLit) ast.VisitResult {
	elementType := e.Type().(ddptypes.ListType).ElementType
	if e.Values != nil {
		for _, v := range e.Values {
			t := v.Type()
			if level := ddptypes.IsReferenceToLevel(elementType, t); level > 0 {
				// a.annotateToRef(v, level)
			} else if level := ddptypes.IsReferenceToLevel(t, elementType); level > 0 {
				a.annotateFromRef(v, level)
			}
		}
	} else if e.Count != nil && e.Value != nil {
		a.annotateDeref(e.Count)
	}
	return ast.VisitRecurse
}

func (a *ImplicitRefCastAnnotator) VisitUnaryExpr(e *ast.UnaryExpr) ast.VisitResult {
	if e.OverloadedBy != nil {
		a.Visit(e.OverloadedBy.Call)
		return ast.VisitSkipChildren
	}
	if e.Operator != ast.UN_DEREF {
		a.annotateDeref(e.Rhs)
	}
	return ast.VisitRecurse
}

func (a *ImplicitRefCastAnnotator) VisitBinaryExpr(e *ast.BinaryExpr) ast.VisitResult {
	if e.OverloadedBy != nil {
		a.Visit(e.OverloadedBy.Call)
		return ast.VisitSkipChildren
	}
	switch e.Operator {
	case ast.BIN_INDEX:
		a.Visit(e.Lhs)
		a.Visit(e.Rhs)
		a.annotateDeref(e.Rhs)
		a.clearAnnotation(e.Lhs)
		if !ddptypes.IsReference(e.Lhs.Type()) {
			a.annotateDeref(e)
		}
		return ast.VisitSkipChildren
	case ast.BIN_FIELD_ACCESS:
		a.Visit(e.Lhs)
		a.Visit(e.Rhs)
		a.clearAnnotation(e.Rhs)
		if !ddptypes.IsReference(e.Rhs.Type()) {
			a.annotateDeref(e)
		}
		return ast.VisitSkipChildren
	default:
		a.annotateDeref(e.Lhs)
		a.annotateDeref(e.Rhs)
	}
	return ast.VisitRecurse
}

func (a *ImplicitRefCastAnnotator) VisitTernaryExpr(e *ast.TernaryExpr) ast.VisitResult {
	if e.OverloadedBy != nil {
		a.Visit(e.OverloadedBy.Call)
		return ast.VisitSkipChildren
	}
	a.annotateDeref(e.Lhs)
	a.annotateDeref(e.Mid)
	a.annotateDeref(e.Rhs)
	return ast.VisitRecurse
}

func (a *ImplicitRefCastAnnotator) VisitCastExpr(e *ast.CastExpr) ast.VisitResult {
	if e.OverloadedBy != nil {
		a.Visit(e.OverloadedBy.Call)
		return ast.VisitSkipChildren
	}

	if ddptypes.DeepEqual(e.Lhs.Type(), e.TargetType) {
		a.Visit(e.Lhs)
		a.clearAnnotation(e.Lhs)
		return ast.VisitSkipChildren
	}

	return ast.VisitRecurse
}

func (a *ImplicitRefCastAnnotator) VisitFuncCall(e *ast.FuncCall) ast.VisitResult {
	if _, ok := a.visitedGenericInstantiations[e.Func]; ast.IsGenericInstantiation(e.Func) && !ok {
		if a.visitedGenericInstantiations == nil {
			a.visitedGenericInstantiations = make(map[*ast.FuncDecl]struct{})
		}
		a.visitedGenericInstantiations[e.Func] = struct{}{}
		a.Visit(e.Func)
	}

	if ast.IsGenericInstantiation(e.Func) {
		oldMod := a.CurrentModule
		a.CurrentModule = e.Func.GenericInstantiation.GenericDecl.Module()
		defer func() {
			a.CurrentModule = oldMod
		}()
	}

	for k, expr := range e.Args {
		a.Visit(expr)
		a.clearAnnotation(expr)
		var paramType ddptypes.Type
		for _, param := range e.Func.Parameters {
			if param.Name.Literal == k {
				paramType = param.Type
				break
			}
		}

		if level := ddptypes.IsReferenceToLevel(paramType, expr.Type()); level > 0 {
			// a.annotateToRef(expr, level)
		} else if level := ddptypes.IsReferenceToLevel(expr.Type(), paramType); level > 0 {
			a.annotateFromRef(expr, level)
		}
	}
	return ast.VisitSkipChildren
}

func (a *ImplicitRefCastAnnotator) VisitStructLiteral(e *ast.StructLiteral) ast.VisitResult {
	for k, expr := range e.Args {
		a.Visit(expr)
		a.clearAnnotation(expr)
		var paramType ddptypes.Type
		for _, field := range e.StructType.Fields {
			if field.Name == k {
				paramType = field.Type
				break
			}
		}

		if level := ddptypes.IsReferenceToLevel(paramType, expr.Type()); level > 0 {
			// a.annotateToRef(expr, level)
		} else if level := ddptypes.IsReferenceToLevel(expr.Type(), paramType); level > 0 {
			a.annotateFromRef(expr, level)
		}
	}
	return ast.VisitRecurse
}

func (a *ImplicitRefCastAnnotator) VisitAssignStmt(s *ast.AssignStmt) ast.VisitResult {
	a.Visit(s.Var)
	a.Visit(s.Rhs)
	a.clearAnnotation(s.Var)
	a.clearAnnotation(s.Rhs)
	varType, rhsType := s.Var.Type(), s.Rhs.Type()

	// varType = ddptypes.Deref(varType)
	if level := ddptypes.IsReferenceToLevel(rhsType, varType); level > 0 {
		a.annotateFromRef(s.Rhs, level)
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

// TODO: visit children manually and clear annotations
func (a *ImplicitRefCastAnnotator) VisitReturnStmt(s *ast.ReturnStmt) ast.VisitResult {
	if s.Value == nil {
		return ast.VisitRecurse
	}

	t := s.Value.Type()
	if level := ddptypes.IsReferenceToLevel(s.Func.ReturnType, t); level > 0 {
		a.annotateToRef(s.Value, level)
	} else if level := ddptypes.IsReferenceToLevel(t, s.Func.ReturnType); level > 0 {
		a.annotateFromRef(s.Value, level)
	}
	return ast.VisitRecurse
}
