package ast

import (
	"sort"

	"github.com/DDP-Projekt/Kompilierer/pkg/token"
)

type helperVisitor struct {
	actualVisitor BaseVisitor
	conditional   bool
}

// invokes visitor on each Node of ast
// while checking if visitor implements
// other *Visitor-Interfaces and invoking
// them accordingly
func VisitAst(ast *Ast, visitor BaseVisitor) {
	h := &helperVisitor{
		actualVisitor: visitor,
		conditional:   false,
	}
	if _, ok := h.actualVisitor.(ConditionalVisitor); ok {
		h.conditional = true
	}
	if scpVis, ok := h.actualVisitor.(ScopeVisitor); ok {
		scpVis.UpdateScope(ast.Symbols)
	}
	for _, stmt := range ast.Statements {
		h.visit(stmt)
	}
}

// invokes visitor on each Node of ast
// while checking if visitor implements
// other *Visitor-Interfaces and invoking
// them accordingly
// a optional SymbolTable may be passed if neccessery
func VisitNode(visitor BaseVisitor, node Node, currentScope *SymbolTable) {
	h := &helperVisitor{
		actualVisitor: visitor,
		conditional:   false,
	}
	if _, ok := h.actualVisitor.(ConditionalVisitor); ok {
		h.conditional = true
	}
	if scpVis, ok := h.actualVisitor.(ScopeVisitor); ok && currentScope != nil {
		scpVis.UpdateScope(currentScope)
	}
	h.visit(node)
}

func (h *helperVisitor) visit(node Node) {
	if h.conditional && h.actualVisitor.(ConditionalVisitor).ShouldVisit(node) {
		node.Accept(h)
	} else if !h.conditional {
		node.Accept(h)
	}
}

func (*helperVisitor) BaseVisitor() {}

func (h *helperVisitor) VisitBadDecl(decl *BadDecl) {
	if vis, ok := h.actualVisitor.(BadDeclVisitor); ok {
		vis.VisitBadDecl(decl)
	}
}
func (h *helperVisitor) VisitVarDecl(decl *VarDecl) {
	if vis, ok := h.actualVisitor.(VarDeclVisitor); ok {
		vis.VisitVarDecl(decl)
	}
	h.visit(decl.InitVal)
}
func (h *helperVisitor) VisitFuncDecl(decl *FuncDecl) {
	if vis, ok := h.actualVisitor.(FuncDeclVisitor); ok {
		vis.VisitFuncDecl(decl)
	}
	if decl.Body != nil {
		h.visit(decl.Body)
	}
}

// if a BadExpr exists the AST is faulty
func (h *helperVisitor) VisitBadExpr(expr *BadExpr) {
	if vis, ok := h.actualVisitor.(BadExprVisitor); ok {
		vis.VisitBadExpr(expr)
	}
}
func (h *helperVisitor) VisitIdent(expr *Ident) {
	if vis, ok := h.actualVisitor.(IdentVisitor); ok {
		vis.VisitIdent(expr)
	}
}
func (h *helperVisitor) VisitIndexing(expr *Indexing) {
	if vis, ok := h.actualVisitor.(IndexingVisitor); ok {
		vis.VisitIndexing(expr)
	}
	h.visit(expr.Lhs)
	h.visit(expr.Index)
}

// nothing to do for literals
func (h *helperVisitor) VisitIntLit(expr *IntLit) {
	if vis, ok := h.actualVisitor.(IntLitVisitor); ok {
		vis.VisitIntLit(expr)
	}
}
func (h *helperVisitor) VisitFloatLit(expr *FloatLit) {
	if vis, ok := h.actualVisitor.(FloatLitVisitor); ok {
		vis.VisitFloatLit(expr)
	}
}
func (h *helperVisitor) VisitBoolLit(expr *BoolLit) {
	if vis, ok := h.actualVisitor.(BoolLitVisitor); ok {
		vis.VisitBoolLit(expr)
	}
}
func (h *helperVisitor) VisitCharLit(expr *CharLit) {
	if vis, ok := h.actualVisitor.(CharLitVisitor); ok {
		vis.VisitCharLit(expr)
	}
}
func (h *helperVisitor) VisitStringLit(expr *StringLit) {
	if vis, ok := h.actualVisitor.(StringLitVisitor); ok {
		vis.VisitStringLit(expr)
	}
}
func (h *helperVisitor) VisitListLit(expr *ListLit) {
	if vis, ok := h.actualVisitor.(ListLitVisitor); ok {
		vis.VisitListLit(expr)
	}
	if expr.Values != nil {
		for _, v := range expr.Values {
			h.visit(v)
		}
	} else if expr.Count != nil && expr.Value != nil {
		h.visit(expr.Count)
		h.visit(expr.Value)
	}
}
func (h *helperVisitor) VisitUnaryExpr(expr *UnaryExpr) {
	if vis, ok := h.actualVisitor.(UnaryExprVisitor); ok {
		vis.VisitUnaryExpr(expr)
	}
	h.visit(expr.Rhs)
}
func (h *helperVisitor) VisitBinaryExpr(expr *BinaryExpr) {
	if vis, ok := h.actualVisitor.(BinaryExprVisitor); ok {
		vis.VisitBinaryExpr(expr)
	}
	h.visit(expr.Lhs)
	h.visit(expr.Rhs)
}
func (h *helperVisitor) VisitTernaryExpr(expr *TernaryExpr) {
	if vis, ok := h.actualVisitor.(TernaryExprVisitor); ok {
		vis.VisitTernaryExpr(expr)
	}
	h.visit(expr.Lhs)
	h.visit(expr.Mid)
	h.visit(expr.Rhs)
}
func (h *helperVisitor) VisitCastExpr(expr *CastExpr) {
	if vis, ok := h.actualVisitor.(CastExprVisitor); ok {
		vis.VisitCastExpr(expr)
	}
	h.visit(expr.Lhs)
}
func (h *helperVisitor) VisitGrouping(expr *Grouping) {
	if vis, ok := h.actualVisitor.(GroupingVisitor); ok {
		vis.VisitGrouping(expr)
	}
	h.visit(expr.Expr)
}
func (h *helperVisitor) VisitFuncCall(expr *FuncCall) {
	if vis, ok := h.actualVisitor.(FuncCallVisitor); ok {
		vis.VisitFuncCall(expr)
	}
	if len(expr.Args) != 0 {
		// sort the arguments to visit them in the order they appear
		args := make([]Expression, 0, len(expr.Args))
		for _, arg := range expr.Args {
			args = append(args, arg)
		}
		sort.Slice(args, func(i, j int) bool {
			iRange, jRange := args[i].GetRange(), args[j].GetRange()
			if iRange.Start.Line < jRange.Start.Line {
				return true
			}
			if iRange.Start.Line == jRange.Start.Line {
				return iRange.Start.Column < jRange.Start.Column
			}
			return false
		})

		for _, arg := range args {
			h.visit(arg)
		}
	}
}

func (h *helperVisitor) VisitBadStmt(stmt *BadStmt) {
	if vis, ok := h.actualVisitor.(BadStmtVisitor); ok {
		vis.VisitBadStmt(stmt)
	}
}
func (h *helperVisitor) VisitDeclStmt(stmt *DeclStmt) {
	if vis, ok := h.actualVisitor.(DeclStmtVisitor); ok {
		vis.VisitDeclStmt(stmt)
	}
	h.visit(stmt.Decl)
}
func (h *helperVisitor) VisitExprStmt(stmt *ExprStmt) {
	if vis, ok := h.actualVisitor.(ExprStmtVisitor); ok {
		vis.VisitExprStmt(stmt)
	}
	h.visit(stmt.Expr)
}
func (h *helperVisitor) VisitAssignStmt(stmt *AssignStmt) {
	if vis, ok := h.actualVisitor.(AssignStmtVisitor); ok {
		vis.VisitAssignStmt(stmt)
	}
	if stmt.Token().Type == token.SPEICHERE {
		h.visit(stmt.Rhs)
		h.visit(stmt.Var)
		return
	}
	h.visit(stmt.Var)
	h.visit(stmt.Rhs)
}
func (h *helperVisitor) VisitBlockStmt(stmt *BlockStmt) {
	if scpVis, ok := h.actualVisitor.(ScopeVisitor); ok && stmt.Symbols != nil {
		scpVis.UpdateScope(stmt.Symbols)
	}

	if vis, ok := h.actualVisitor.(BlockStmtVisitor); ok {
		vis.VisitBlockStmt(stmt)
	}
	for _, stmt := range stmt.Statements {
		h.visit(stmt)
	}

	if scpVis, ok := h.actualVisitor.(ScopeVisitor); ok && stmt.Symbols != nil {
		scpVis.UpdateScope(stmt.Symbols.Enclosing)
	}
}
func (h *helperVisitor) VisitIfStmt(stmt *IfStmt) {
	if vis, ok := h.actualVisitor.(IfStmtVisitor); ok {
		vis.VisitIfStmt(stmt)
	}
	h.visit(stmt.Condition)
	h.visit(stmt.Then)
	if stmt.Else != nil {
		h.visit(stmt.Else)
	}
}
func (h *helperVisitor) VisitWhileStmt(stmt *WhileStmt) {
	if vis, ok := h.actualVisitor.(WhileStmtVisitor); ok {
		vis.VisitWhileStmt(stmt)
	}
	switch op := stmt.While.Type; op {
	case token.SOLANGE, token.WIEDERHOLE:
		h.visit(stmt.Condition)
		h.visit(stmt.Body)
	case token.MACHE:
		h.visit(stmt.Body)
		h.visit(stmt.Condition)
	}
}
func (h *helperVisitor) VisitForStmt(stmt *ForStmt) {
	if vis, ok := h.actualVisitor.(ForStmtVisitor); ok {
		vis.VisitForStmt(stmt)
	}

	h.visit(stmt.Initializer)
	h.visit(stmt.To)
	if stmt.StepSize != nil {
		h.visit(stmt.StepSize)
	}
	h.visit(stmt.Body)
}
func (h *helperVisitor) VisitForRangeStmt(stmt *ForRangeStmt) {
	if vis, ok := h.actualVisitor.(ForRangeStmtVisitor); ok {
		vis.VisitForRangeStmt(stmt)
	}

	h.visit(stmt.Initializer)
	h.visit(stmt.In)
	h.visit(stmt.Body)
}
func (h *helperVisitor) VisitReturnStmt(stmt *ReturnStmt) {
	if vis, ok := h.actualVisitor.(ReturnStmtVisitor); ok {
		vis.VisitReturnStmt(stmt)
	}
	if stmt.Value == nil {
		return
	}
	h.visit(stmt.Value)
}
