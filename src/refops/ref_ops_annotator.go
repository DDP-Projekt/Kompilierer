package refops

import (
	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
	"github.com/DDP-Projekt/Kompilierer/src/parser/typechecker"
)

// RefOpsAnnotator annotates expressions and assignments with RefOpsMeta and AssignTargetMeta respectively
// must run after ImplicitRefCastAnnotator
type RefOpsAnnotator struct {
	ast.BaseVisitor
	visitedGenericInstantiations map[*ast.FuncDecl]struct{}
	// VarDecls that must not be treated like normal variable declarations
	// (the iteration variables of for-range loops)
	skipVarDecls map[*ast.VarDecl]struct{}
}

var (
	_ ast.Annotator = (*RefOpsAnnotator)(nil)

	_ ast.ConstDeclVisitor  = (*RefOpsAnnotator)(nil)
	_ ast.VarDeclVisitor    = (*RefOpsAnnotator)(nil)
	_ ast.FuncDeclVisitor   = (*RefOpsAnnotator)(nil)
	_ ast.StructDeclVisitor = (*RefOpsAnnotator)(nil)

	_ ast.ExprStmtVisitor     = (*RefOpsAnnotator)(nil)
	_ ast.AssignStmtVisitor   = (*RefOpsAnnotator)(nil)
	_ ast.IfStmtVisitor       = (*RefOpsAnnotator)(nil)
	_ ast.WhileStmtVisitor    = (*RefOpsAnnotator)(nil)
	_ ast.ForStmtVisitor      = (*RefOpsAnnotator)(nil)
	_ ast.ForRangeStmtVisitor = (*RefOpsAnnotator)(nil)
	_ ast.ReturnStmtVisitor   = (*RefOpsAnnotator)(nil)
)

func (a *RefOpsAnnotator) VisitConstDecl(d *ast.ConstDecl) ast.VisitResult {
	// the value is evaluated wherever the constant is used (compiler.VisitIdent)
	a.expr(d.Val, true)
	return ast.VisitSkipChildren
}

func (a *RefOpsAnnotator) VisitVarDecl(d *ast.VarDecl) ast.VisitResult {
	if _, skip := a.skipVarDecls[d]; skip || d.InitVal == nil {
		return ast.VisitSkipChildren
	}

	m := a.expr(d.InitVal, true)
	// implicit cast to any (compiler.VisitVarDecl)
	if _, t, refLevel := ddptypes.CastReferenceLevel(d.Type); ddptypes.DeepEqual(t, ddptypes.VARIABLE) && !isAnyValue(d.InitVal, m) {
		a.box(d.InitVal, m, refLevel)
	}
	return ast.VisitSkipChildren
}

func (a *RefOpsAnnotator) VisitFuncDecl(d *ast.FuncDecl) ast.VisitResult {
	// generic functions are only compiled as instantiations
	if ast.IsGeneric(d) && !ast.IsExternFunc(d) {
		return ast.VisitSkipChildren
	}
	return ast.VisitRecurse
}

func (a *RefOpsAnnotator) VisitStructDecl(d *ast.StructDecl) ast.VisitResult {
	// default values of fields are evaluated at every struct literal (compiler.evaluateStructLiteral)
	for _, field := range d.Fields {
		fieldDecl, ok := field.(*ast.VarDecl)
		if !ok || fieldDecl.InitVal == nil {
			continue
		}

		m := a.expr(fieldDecl.InitVal, true)
		if ddptypes.DeepEqual(fieldDecl.Type, ddptypes.VARIABLE) && !isAnyValue(fieldDecl.InitVal, m) {
			a.box(fieldDecl.InitVal, m, 0)
		}
	}
	return ast.VisitSkipChildren
}

func (a *RefOpsAnnotator) VisitExprStmt(s *ast.ExprStmt) ast.VisitResult {
	// the compiler visits the expression directly, so its implicit casts are not applied
	a.expr(s.Expr, false)
	return ast.VisitSkipChildren
}

func (a *RefOpsAnnotator) VisitAssignStmt(s *ast.AssignStmt) ast.VisitResult {
	rhs := a.expr(s.Rhs, true)
	a.expr(s.Var, true)

	// implicit cast to any (compiler.VisitAssignStmt)
	if _, t, refLevel := ddptypes.CastReferenceLevel(s.Var.Type()); ddptypes.IsAnyDeref(t) && !isAnyValue(s.Rhs, rhs) {
		boxPromotions := uint(0)
		if refLevel > 0 {
			// the target already is a reference, so one level less is needed
			boxPromotions = refLevel - 1
		}
		a.box(s.Rhs, rhs, boxPromotions)
	}

	s.SetMetadataAttachement(assignTarget(s))
	return ast.VisitSkipChildren
}

func (a *RefOpsAnnotator) VisitIfStmt(s *ast.IfStmt) ast.VisitResult {
	a.expr(s.Condition, true)
	return ast.VisitRecurse
}

func (a *RefOpsAnnotator) VisitWhileStmt(s *ast.WhileStmt) ast.VisitResult {
	a.expr(s.Condition, true)
	return ast.VisitRecurse
}

func (a *RefOpsAnnotator) VisitForStmt(s *ast.ForStmt) ast.VisitResult {
	// the Initializer is visited as normal VarDecl
	a.expr(s.To, true)
	if s.StepSize != nil {
		a.expr(s.StepSize, true)
	}
	return ast.VisitRecurse
}

func (a *RefOpsAnnotator) VisitForRangeStmt(s *ast.ForRangeStmt) ast.VisitResult {
	// Initializer.InitVal is the same expression as In,
	// and the iteration variables are not initialized like normal VarDecls
	if a.skipVarDecls == nil {
		a.skipVarDecls = make(map[*ast.VarDecl]struct{})
	}
	a.skipVarDecls[s.Initializer] = struct{}{}
	if s.Index != nil {
		a.skipVarDecls[s.Index] = struct{}{}
	}

	a.expr(s.In, true)
	return ast.VisitRecurse
}

func (a *RefOpsAnnotator) VisitReturnStmt(s *ast.ReturnStmt) ast.VisitResult {
	if s.Value == nil {
		return ast.VisitSkipChildren
	}

	m := a.expr(s.Value, true)
	// implicit cast to any (compiler.VisitReturnStmt)
	if s.Func != nil && ddptypes.DeepEqual(s.Func.ReturnType, ddptypes.VARIABLE) && !isAnyValue(s.Value, m) {
		a.box(s.Value, m, 0)
	}
	return ast.VisitSkipChildren
}

// annotates expr and all its sub-expressions
// if applyImplicit is false, the ImplicitRefCastMeta of expr is ignored
// (because the compiler does not apply it in that context)
func (a *RefOpsAnnotator) expr(e ast.Expression, applyImplicit bool) RefOpsMeta {
	if e == nil {
		return RefOpsMeta{}
	}

	m := a.node(e)
	if applyImplicit {
		// mirrors compiler.evaluateNumeric
		if att, ok := e.GetMetadataByKind(ImplicitRefCastMetaKind); ok {
			implicit := att.(ImplicitRefCastMeta)
			if n := implicit.DerefLevel(); n > 0 {
				// the compiler only dereferences as long as the value is a reference
				m.Derefs = min(n, m.BaseLevel)
			} else if n := implicit.UpcastLevel(); n > 0 {
				m.Promotions = n
			}
		}
	}
	m.updateLevel()
	e.SetMetadataAttachement(m)
	return m
}

// boxes the value of e (already annotated with m) into an any and promotes it boxPromotions times
func (a *RefOpsAnnotator) box(e ast.Expression, m RefOpsMeta, boxPromotions uint) {
	m.BoxAny = true
	m.BoxPromotions = boxPromotions
	m.updateLevel()
	e.SetMetadataAttachement(m)
}

// annotates the children of e and returns the RefOpsMeta of e
// without its implicit casts applied (only Op and BaseLevel are set)
func (a *RefOpsAnnotator) node(e ast.Expression) RefOpsMeta {
	isAssigneable := e.HasMetadata(typechecker.AssigneableMetaKind)

	switch e := e.(type) {
	case *ast.Ident:
		// constants are evaluated from their literal, so they are never addressed
		if decl, isVar := e.Declaration.(*ast.VarDecl); isVar && isAssigneable {
			return RefOpsMeta{Op: OpAddr, BaseLevel: refDepth(decl.Type) + 1}
		}
		return valueOf(e)
	case *ast.ListLit:
		for _, v := range e.Values {
			a.expr(v, true)
		}
		a.expr(e.Count, true)
		a.expr(e.Value, true)
		return valueOf(e)
	case *ast.UnaryExpr:
		if e.OverloadedBy != nil {
			a.call(e.OverloadedBy.Call)
			return valueOf(e)
		}

		rhs := a.expr(e.Rhs, true)
		if e.Operator != ast.UN_DEREF {
			return valueOf(e)
		}
		// an assignable 'Wert von' keeps the pointer: &*r = r
		if isAssigneable {
			return RefOpsMeta{Op: OpAddr, BaseLevel: rhs.Level}
		}
		return RefOpsMeta{Op: OpDeref, BaseLevel: satSub(rhs.Level, 1)}
	case *ast.BinaryExpr:
		return a.binary(e, isAssigneable)
	case *ast.TernaryExpr:
		if e.OverloadedBy != nil {
			a.call(e.OverloadedBy.Call)
			return valueOf(e)
		}

		lhs := a.expr(e.Lhs, true)
		a.expr(e.Mid, true)
		a.expr(e.Rhs, true)
		if e.Operator == ast.TER_FALLS {
			// the compiler uses the type of the first operand for the result
			return RefOpsMeta{Op: OpValue, BaseLevel: lhs.Level}
		}
		return valueOf(e)
	case *ast.CastExpr:
		return a.cast(e, isAssigneable)
	case *ast.TypeCheck:
		a.expr(e.Lhs, true)
		return valueOf(e)
	case *ast.Grouping:
		// the compiler visits the inner expression directly, so its implicit casts are not applied,
		// only the ones of the grouping itself
		inner := a.expr(e.Expr, false)
		return RefOpsMeta{Op: inner.Op, BaseLevel: inner.Level}
	case *ast.FuncCall:
		a.call(e)
		return valueOf(e)
	case *ast.StructLiteral:
		a.structLiteral(e)
		return valueOf(e)
	}
	// literals, TypeOpExpr, BadExpr
	return valueOf(e)
}

func (a *RefOpsAnnotator) binary(e *ast.BinaryExpr, isAssigneable bool) RefOpsMeta {
	if e.OverloadedBy != nil {
		a.call(e.OverloadedBy.Call)
		return valueOf(e)
	}

	switch e.Operator {
	case ast.BIN_FIELD_ACCESS:
		// e.Lhs is the name of the field and is not evaluated
		rhs := a.expr(e.Rhs, true)
		// the field address can only be taken if the struct is accessed through a pointer
		if isAssigneable && rhs.Level > 0 {
			return RefOpsMeta{Op: OpAddr, BaseLevel: refDepth(e.Type()) + 1}
		}
		return valueOf(e)
	case ast.BIN_INDEX:
		lhs := a.expr(e.Lhs, true)
		a.expr(e.Rhs, true)
		if e.HasMetadata(ast.StringIndexingMetaKind) || ddptypes.EqualDeref(e.Lhs.Type(), ddptypes.TEXT) {
			return valueOf(e)
		}
		// the element address can only be taken if the list is accessed through a pointer
		if isAssigneable && lhs.Level > 0 {
			return RefOpsMeta{Op: OpAddr, BaseLevel: refDepth(e.Type()) + 1}
		}
		return valueOf(e)
	}

	a.expr(e.Lhs, true)
	a.expr(e.Rhs, true)
	return valueOf(e)
}

// mirrors compiler.VisitCastExpr
func (a *RefOpsAnnotator) cast(e *ast.CastExpr, isAssigneable bool) RefOpsMeta {
	if e.OverloadedBy != nil {
		a.call(e.OverloadedBy.Call)
		return valueOf(e)
	}

	lhs := a.expr(e.Lhs, true)
	// an assignable cast only reinterprets the address of its operand
	if isAssigneable {
		return RefOpsMeta{Op: OpAddr, BaseLevel: lhs.Level}
	}

	target := ddptypes.TrueUnderlying(e.TargetType)
	targetLevel := refDepth(target)
	switch {
	case lhs.Level == 0 && targetLevel > 0:
		// casting an any to a reference unboxes it, everything else is copied into a new cell
		if isAnyValue(e.Lhs, lhs) {
			return RefOpsMeta{Op: OpValue, BaseLevel: targetLevel}
		}
		return RefOpsMeta{Op: OpPromote, BaseLevel: targetLevel}
	case lhs.Level > 0 && targetLevel == 0:
		return RefOpsMeta{Op: OpDeref, BaseLevel: 0}
	}
	return RefOpsMeta{Op: OpValue, BaseLevel: targetLevel}
}

func (a *RefOpsAnnotator) call(e *ast.FuncCall) {
	if e == nil || e.Func == nil {
		return
	}

	// generic instantiations are not part of the module statements,
	// so they are annotated when they are first called
	if ast.IsGenericInstantiation(e.Func) && !ast.IsExternFunc(e.Func) {
		if a.visitedGenericInstantiations == nil {
			a.visitedGenericInstantiations = make(map[*ast.FuncDecl]struct{})
		}
		if _, visited := a.visitedGenericInstantiations[e.Func]; !visited {
			a.visitedGenericInstantiations[e.Func] = struct{}{}
			ast.VisitNode(a, e.Func, nil)
		}
	}

	// same order as compiler.VisitFuncCall
	for _, param := range e.Func.Parameters {
		a.expr(e.Args[param.Name.Literal], true)
	}
}

// mirrors compiler.evaluateStructLiteral
// missing arguments are replaced by the field defaults, which are annotated in VisitStructDecl
func (a *RefOpsAnnotator) structLiteral(e *ast.StructLiteral) {
	if e.StructType == nil {
		return
	}

	for _, field := range e.StructType.Fields {
		arg, hasArg := e.Args[field.Name]
		if !hasArg {
			continue
		}

		m := a.expr(arg, true)
		if ddptypes.DeepEqual(field.Type, ddptypes.VARIABLE) && !isAnyValue(arg, m) {
			a.box(arg, m, 0)
		}
	}
}

// computes the AssignTargetMeta of an already annotated AssignStmt
func assignTarget(s *ast.AssignStmt) AssignTargetMeta {
	if s.Var.HasMetadata(ast.StringIndexingMetaKind) {
		return AssignTargetMeta{Target: AssignStringIndexing}
	}

	// if the target is not an address, it is a reference that is stored through
	if m, _ := GetRefOps(s.Var); !m.AddrOf() {
		return AssignTargetMeta{Target: AssignStoreThrough}
	}

	target := storageOf(s.Var)
	// storing into a reference variable itself rebinds it
	if ident, isIdent := unwrapGrouping(s.Var).(*ast.Ident); isIdent && target.Target == AssignStoreIntoVar {
		if ddptypes.IsReference(ident.Type()) {
			target.Target = AssignRebind
		}
	}
	return target
}

// determines which storage the address yielded by e (an OpAddr expression) belongs to
func storageOf(e ast.Expression) AssignTargetMeta {
	switch e := e.(type) {
	case *ast.Grouping:
		return storageOf(e.Expr)
	case *ast.Ident:
		if decl, isVar := e.Declaration.(*ast.VarDecl); isVar {
			if decl.IsGlobal {
				return AssignTargetMeta{Target: AssignStoreIntoGlobal, Root: decl}
			}
			return AssignTargetMeta{Target: AssignStoreIntoVar, Root: decl}
		}
	case *ast.BinaryExpr:
		switch e.Operator {
		case ast.BIN_FIELD_ACCESS:
			// the field belongs to the storage of the struct, if the struct itself was addressed,
			// otherwise the struct is accessed through a reference
			if m, _ := GetRefOps(e.Rhs); m.AddrOf() {
				return storageOf(e.Rhs)
			}
		case ast.BIN_INDEX:
			return AssignTargetMeta{Target: AssignStoreIntoList}
		}
	case *ast.CastExpr:
		if e.OverloadedBy == nil {
			if m, _ := GetRefOps(e.Lhs); m.AddrOf() {
				return storageOf(e.Lhs)
			}
		}
	}
	// 'Wert von', references returned from functions or operators, ...
	return AssignTargetMeta{Target: AssignStoreThrough}
}

func unwrapGrouping(e ast.Expression) ast.Expression {
	for {
		grouping, ok := e.(*ast.Grouping)
		if !ok {
			return e
		}
		e = grouping.Expr
	}
}

// the RefOpsMeta of a node that yields its value
func valueOf(e ast.Expression) RefOpsMeta {
	return RefOpsMeta{Op: OpValue, BaseLevel: refDepth(e.Type())}
}

// wether the value described by m (the annotated value of e) is a plain any,
// which corresponds to the compiler checks of the form 'val.typ != c.ddpany'
func isAnyValue(e ast.Expression, m RefOpsMeta) bool {
	if m.BoxAny {
		return m.Level == 0
	}
	if m.Level != 0 {
		return false
	}
	_, t, _ := ddptypes.CastReferenceLevel(e.Type())
	return ddptypes.IsAny(t)
}

func refDepth(t ddptypes.Type) uint {
	if t == nil {
		return 0
	}
	return ddptypes.RefDepth(t)
}

func satSub(a, b uint) uint {
	if a < b {
		return 0
	}
	return a - b
}
