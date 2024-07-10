package ast

import (
	"reflect"
	"sort"

	"github.com/DDP-Projekt/Kompilierer/src/token"
)

// helper class for visiting the AST
type helperVisitor struct {
	actualVisitor Visitor
}

// visits the AST of the given module
// does nothing if module == nil
//
// visitor can implement any of the Visit<Node> interfaces, and
// the corresponding method is called for each node
//
// if the given Visitor implements the ConditionalVisitor interface,
// the ShouldVisit method is used to determine if a node should be visited
//
// if the given Visitor implements the ScopeSetter interface,
// the SetScope method is called when the scope changes
//
// if the given Visitor implements the ModuleSetter interface,
// the SetModule method is called before visiting the module
func VisitModule(module *Module, visitor Visitor) {
	if module == nil {
		return
	}

	helper := &helperVisitor{
		actualVisitor: visitor,
	}
	visitSingleModule(module, helper)
}

// helper method to visit all nodes in a module
func visitSingleModule(module *Module, v *helperVisitor) {
	if modVis, ok := v.actualVisitor.(ModuleSetter); ok {
		modVis.SetModule(module)
	}
	if scpVis, ok := v.actualVisitor.(ScopeSetter); ok {
		scpVis.SetScope(module.Ast.Symbols)
	}

	for _, stmt := range module.Ast.Statements {
		if v.visit(stmt) == VisitBreak {
			return
		}
	}
}

// visits the given module and all the modules it imports recursively
// does nothing if module == nil
//
// visitor can implement any of the Visit<Node> interfaces, and
// the corresponding method is called for each node
//
// imported modules are visited in the order they are included
//
// if the given Visitor implements the ConditionalVisitor interface,
// the ShouldVisit method is used to determine if a node should be visited
//
// if the given Visitor implements the ScopeSetter interface,
// the SetScope method is called when the scope changes
//
// if the given Visitor implements the ModuleSetter interface,
// the SetModule method is called before visiting each module
func VisitModuleRec(module *Module, visitor Visitor) {
	if module == nil {
		return
	}

	helper := &helperVisitor{
		actualVisitor: visitor,
	}
	visitModuleRec(module, helper, make(map[*Module]struct{}, len(module.Imports)+1))
}

// implementation of VisitModuleRec
func visitModuleRec(module *Module, visitor *helperVisitor, visited map[*Module]struct{}) {
	// return if already visited
	if _, ok := visited[module]; ok {
		return
	}

	// mark the module as visited
	visited[module] = struct{}{}

	// sort imports by include order
	imports := make([]*ImportStmt, len(module.Imports))
	copy(imports, module.Imports)
	sort.Slice(imports, func(i, j int) bool {
		return imports[i].Range.Start.IsBefore(imports[j].Range.Start)
	})

	// visit imports
	for _, imprt := range imports {
		if imprt.Module != nil {
			visitModuleRec(imprt.Module, visitor, visited)
		}
	}

	visitSingleModule(module, visitor)
}

// visits the given node and all its children recursively
// does nothing if node == nil
//
// visitor can implement any of the Visit<Node> interfaces, and
// the corresponding method is called for each node
//
// if the given Visitor implements the ConditionalVisitor interface,
// the ShouldVisit method is used to determine if a node should be visited
//
// if the given Visitor implements the ScopeSetter interface,
// the SetScope method is called when the scope changes
func VisitNode(visitor Visitor, node Node, currentScope *SymbolTable) {
	if node == nil {
		return
	}

	helper := &helperVisitor{
		actualVisitor: visitor,
	}

	if scpVis, ok := helper.actualVisitor.(ScopeSetter); ok && currentScope != nil {
		scpVis.SetScope(currentScope)
	}
	helper.visit(node)
}

// helper method to check for nil nodes and conditional visitors
func (h *helperVisitor) visit(node Node) VisitResult {
	if isNil(node) {
		return VisitRecurse
	}

	condVis, isConditional := h.actualVisitor.(ConditionalVisitor)
	if isConditional && condVis.ShouldVisit(node) {
		return node.Accept(h)
	} else if !isConditional {
		return node.Accept(h)
	}
	return VisitRecurse
}

// helper method to handle return values of child visits
func (h *helperVisitor) visitChildren(result VisitResult, children ...Node) VisitResult {
	switch result {
	case VisitBreak:
		return VisitBreak
	case VisitRecurse:
		for _, child := range children {
			if h.visit(child) == VisitBreak {
				return VisitBreak
			}
		}
	}
	return VisitRecurse
}

func (*helperVisitor) Visitor() {}

func (h *helperVisitor) VisitBadDecl(decl *BadDecl) VisitResult {
	if vis, ok := h.actualVisitor.(BadDeclVisitor); ok {
		return vis.VisitBadDecl(decl)
	}
	return VisitRecurse
}

func (h *helperVisitor) VisitVarDecl(decl *VarDecl) VisitResult {
	result := VisitRecurse
	if vis, ok := h.actualVisitor.(VarDeclVisitor); ok {
		result = vis.VisitVarDecl(decl)
	}
	return h.visitChildren(result, decl.InitVal)
}

func (h *helperVisitor) VisitFuncDecl(decl *FuncDecl) VisitResult {
	result := VisitRecurse
	if vis, ok := h.actualVisitor.(FuncDeclVisitor); ok {
		result = vis.VisitFuncDecl(decl)
	}
	return h.visitChildren(result, decl.Body)
}

func (h *helperVisitor) VisitStructDecl(decl *StructDecl) VisitResult {
	result := VisitRecurse
	if vis, ok := h.actualVisitor.(StructDeclVisitor); ok {
		result = vis.VisitStructDecl(decl)
	}
	return h.visitChildren(result, sortedByRange(decl.Fields)...)
}

func (h *helperVisitor) VisitTypeAliasDecl(decl *TypeAliasDecl) VisitResult {
	if vis, ok := h.actualVisitor.(TypeAliasDeclVisitor); ok {
		return vis.VisitTypeAliasDecl(decl)
	}
	return VisitRecurse
}

// if a BadExpr exists the AST is faulty
func (h *helperVisitor) VisitBadExpr(expr *BadExpr) VisitResult {
	if vis, ok := h.actualVisitor.(BadExprVisitor); ok {
		return vis.VisitBadExpr(expr)
	}
	return VisitRecurse
}

func (h *helperVisitor) VisitIdent(expr *Ident) VisitResult {
	if vis, ok := h.actualVisitor.(IdentVisitor); ok {
		return vis.VisitIdent(expr)
	}
	return VisitRecurse
}

func (h *helperVisitor) VisitIndexing(expr *Indexing) VisitResult {
	result := VisitRecurse
	if vis, ok := h.actualVisitor.(IndexingVisitor); ok {
		result = vis.VisitIndexing(expr)
	}
	return h.visitChildren(result, expr.Lhs, expr.Index)
}

func (h *helperVisitor) VisitFieldAccess(expr *FieldAccess) VisitResult {
	result := VisitRecurse
	if vis, ok := h.actualVisitor.(FieldAccessVisitor); ok {
		result = vis.VisitFieldAccess(expr)
	}
	return h.visitChildren(result, expr.Field, expr.Rhs)
}

// nothing to do for literals
func (h *helperVisitor) VisitIntLit(expr *IntLit) VisitResult {
	if vis, ok := h.actualVisitor.(IntLitVisitor); ok {
		return vis.VisitIntLit(expr)
	}
	return VisitRecurse
}

func (h *helperVisitor) VisitFloatLit(expr *FloatLit) VisitResult {
	if vis, ok := h.actualVisitor.(FloatLitVisitor); ok {
		return vis.VisitFloatLit(expr)
	}
	return VisitRecurse
}

func (h *helperVisitor) VisitBoolLit(expr *BoolLit) VisitResult {
	if vis, ok := h.actualVisitor.(BoolLitVisitor); ok {
		return vis.VisitBoolLit(expr)
	}
	return VisitRecurse
}

func (h *helperVisitor) VisitCharLit(expr *CharLit) VisitResult {
	if vis, ok := h.actualVisitor.(CharLitVisitor); ok {
		return vis.VisitCharLit(expr)
	}
	return VisitRecurse
}

func (h *helperVisitor) VisitStringLit(expr *StringLit) VisitResult {
	if vis, ok := h.actualVisitor.(StringLitVisitor); ok {
		return vis.VisitStringLit(expr)
	}
	return VisitRecurse
}

func (h *helperVisitor) VisitListLit(expr *ListLit) VisitResult {
	result := VisitRecurse
	if vis, ok := h.actualVisitor.(ListLitVisitor); ok {
		result = vis.VisitListLit(expr)
	}
	switch result {
	case VisitBreak:
		return VisitBreak
	case VisitRecurse:
		if expr.Values != nil {
			return h.visitChildren(result, toInterfaceSlice[Expression, Node](expr.Values)...)
		} else if expr.Count != nil && expr.Value != nil {
			return h.visitChildren(result, expr.Count, expr.Value)
		}
		return VisitRecurse
	default:
		return VisitRecurse
	}
}

func (h *helperVisitor) VisitUnaryExpr(expr *UnaryExpr) VisitResult {
	result := VisitRecurse
	if vis, ok := h.actualVisitor.(UnaryExprVisitor); ok {
		result = vis.VisitUnaryExpr(expr)
	}
	return h.visitChildren(result, expr.Rhs)
}

func (h *helperVisitor) VisitBinaryExpr(expr *BinaryExpr) VisitResult {
	result := VisitRecurse
	if vis, ok := h.actualVisitor.(BinaryExprVisitor); ok {
		result = vis.VisitBinaryExpr(expr)
	}
	return h.visitChildren(result, expr.Lhs, expr.Rhs)
}

func (h *helperVisitor) VisitTernaryExpr(expr *TernaryExpr) VisitResult {
	result := VisitRecurse
	if vis, ok := h.actualVisitor.(TernaryExprVisitor); ok {
		result = vis.VisitTernaryExpr(expr)
	}
	return h.visitChildren(result, expr.Lhs, expr.Mid, expr.Rhs)
}

func (h *helperVisitor) VisitCastExpr(expr *CastExpr) VisitResult {
	result := VisitRecurse
	if vis, ok := h.actualVisitor.(CastExprVisitor); ok {
		result = vis.VisitCastExpr(expr)
	}
	return h.visitChildren(result, expr.Lhs)
}

func (h *helperVisitor) VisitTypeOpExpr(expr *TypeOpExpr) VisitResult {
	result := VisitRecurse
	if vis, ok := h.actualVisitor.(TypeOpExprVisitor); ok {
		result = vis.VisitTypeOpExpr(expr)
	}
	return result
}

func (h *helperVisitor) VisitGrouping(expr *Grouping) VisitResult {
	result := VisitRecurse
	if vis, ok := h.actualVisitor.(GroupingVisitor); ok {
		result = vis.VisitGrouping(expr)
	}
	return h.visitChildren(result, expr.Expr)
}

func (h *helperVisitor) VisitFuncCall(expr *FuncCall) VisitResult {
	result := VisitRecurse
	if vis, ok := h.actualVisitor.(FuncCallVisitor); ok {
		result = vis.VisitFuncCall(expr)
	}
	return h.visitChildren(result, h.sortArgs(expr.Args)...)
}

func (h *helperVisitor) VisitStructLiteral(expr *StructLiteral) VisitResult {
	result := VisitRecurse
	if vis, ok := h.actualVisitor.(StructLiteralVisitor); ok {
		result = vis.VisitStructLiteral(expr)
	}
	return h.visitChildren(result, h.sortArgs(expr.Args)...)
}

func (h *helperVisitor) VisitBadStmt(stmt *BadStmt) VisitResult {
	if vis, ok := h.actualVisitor.(BadStmtVisitor); ok {
		return vis.VisitBadStmt(stmt)
	}
	return VisitRecurse
}

func (h *helperVisitor) VisitDeclStmt(stmt *DeclStmt) VisitResult {
	result := VisitRecurse
	if vis, ok := h.actualVisitor.(DeclStmtVisitor); ok {
		result = vis.VisitDeclStmt(stmt)
	}
	return h.visitChildren(result, stmt.Decl)
}

func (h *helperVisitor) VisitExprStmt(stmt *ExprStmt) VisitResult {
	result := VisitRecurse
	if vis, ok := h.actualVisitor.(ExprStmtVisitor); ok {
		result = vis.VisitExprStmt(stmt)
	}
	return h.visitChildren(result, stmt.Expr)
}

func (h *helperVisitor) VisitImportStmt(stmt *ImportStmt) VisitResult {
	if vis, ok := h.actualVisitor.(ImportStmtVisitor); ok {
		return vis.VisitImportStmt(stmt)
	}
	return VisitRecurse
}

func (h *helperVisitor) VisitAssignStmt(stmt *AssignStmt) VisitResult {
	result := VisitRecurse
	if vis, ok := h.actualVisitor.(AssignStmtVisitor); ok {
		result = vis.VisitAssignStmt(stmt)
	}
	if stmt.Token().Type == token.SPEICHERE {
		return h.visitChildren(result, stmt.Rhs, stmt.Var)
	}
	return h.visitChildren(result, stmt.Var, stmt.Rhs)
}

func (h *helperVisitor) VisitBlockStmt(stmt *BlockStmt) VisitResult {
	if scpVis, ok := h.actualVisitor.(ScopeSetter); ok && stmt.Symbols != nil {
		scpVis.SetScope(stmt.Symbols)
	}

	result := VisitRecurse
	if vis, ok := h.actualVisitor.(BlockStmtVisitor); ok {
		result = vis.VisitBlockStmt(stmt)
	}

	result = h.visitChildren(result, toInterfaceSlice[Statement, Node](stmt.Statements)...)

	if scpVis, ok := h.actualVisitor.(ScopeSetter); ok && stmt.Symbols != nil {
		scpVis.SetScope(stmt.Symbols.Enclosing)
	}
	return result
}

func (h *helperVisitor) VisitIfStmt(stmt *IfStmt) VisitResult {
	result := VisitRecurse
	if vis, ok := h.actualVisitor.(IfStmtVisitor); ok {
		result = vis.VisitIfStmt(stmt)
	}
	return h.visitChildren(result, stmt.Condition, stmt.Then, stmt.Else)
}

func (h *helperVisitor) VisitWhileStmt(stmt *WhileStmt) VisitResult {
	result := VisitRecurse
	if vis, ok := h.actualVisitor.(WhileStmtVisitor); ok {
		result = vis.VisitWhileStmt(stmt)
	}
	switch op := stmt.While.Type; op {
	case token.SOLANGE:
		return h.visitChildren(result, stmt.Condition, stmt.Body)
	case token.MACHE, token.WIEDERHOLE:
		return h.visitChildren(result, stmt.Body, stmt.Condition)
	}
	return VisitRecurse
}

func (h *helperVisitor) VisitForStmt(stmt *ForStmt) VisitResult {
	result := VisitRecurse
	if vis, ok := h.actualVisitor.(ForStmtVisitor); ok {
		result = vis.VisitForStmt(stmt)
	}
	return h.visitChildren(result, stmt.Initializer, stmt.To, stmt.StepSize, stmt.Body)
}

func (h *helperVisitor) VisitForRangeStmt(stmt *ForRangeStmt) VisitResult {
	result := VisitRecurse
	if vis, ok := h.actualVisitor.(ForRangeStmtVisitor); ok {
		result = vis.VisitForRangeStmt(stmt)
	}
	return h.visitChildren(result, stmt.Initializer, stmt.In, stmt.Body)
}

func (h *helperVisitor) VisitBreakContinueStmt(stmt *BreakContinueStmt) VisitResult {
	if vis, ok := h.actualVisitor.(BreakContineStmtVisitor); ok {
		return vis.VisitBreakContinueStmt(stmt)
	}
	return VisitRecurse
}

func (h *helperVisitor) VisitReturnStmt(stmt *ReturnStmt) VisitResult {
	result := VisitRecurse
	if vis, ok := h.actualVisitor.(ReturnStmtVisitor); ok {
		result = vis.VisitReturnStmt(stmt)
	}
	return h.visitChildren(result, stmt.Value)
}

// helper for visitFuncCall and visitStructLiteral
// sorts the arguments by their order in the source code by using their ranges
func (h *helperVisitor) sortArgs(Args map[string]Expression) []Node {
	if len(Args) != 0 {
		// sort the arguments to visit them in the order they appear
		args := make([]Node, 0, len(Args))
		for _, arg := range Args {
			args = append(args, arg)
		}
		return args
	}
	return nil
}

func sortedByRange[T Node](nodes []T) []Node {
	nodesCopy := toInterfaceSlice[T, Node](nodes)
	sort.Slice(nodesCopy, func(i, j int) bool {
		iRange, jRange := nodes[i].GetRange(), nodes[j].GetRange()
		if iRange.Start.Line < jRange.Start.Line {
			return true
		}
		if iRange.Start.Line == jRange.Start.Line {
			return iRange.Start.Column < jRange.Start.Column
		}
		return false
	})
	return nodesCopy
}

// converts a slice of a subtype to it's basetype
// T must be convertible to U
func toInterfaceSlice[T any, U any](slice []T) []U {
	result := make([]U, len(slice))
	for i := range slice {
		result[i] = any(slice[i]).(U)
	}
	return result
}

func isNil(node Node) bool {
	if node == nil {
		return true
	}
	switch reflect.TypeOf(node).Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Pointer,
		reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		return reflect.ValueOf(node).IsNil()
	default:
		return false
	}
}
