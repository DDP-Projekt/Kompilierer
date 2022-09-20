package ast

import "github.com/DDP-Projekt/Kompilierer/pkg/token"

type helperVisitor struct {
	actualVisitor BaseVisitor
}

func VisitAst(ast *Ast, visitor BaseVisitor) {
	vis := &helperVisitor{
		actualVisitor: visitor,
	}
	for _, stmt := range ast.Statements {
		stmt.Accept(vis)
	}
}

func (b *helperVisitor) VisitBadDecl(decl *BadDecl) {
	if vis, ok := b.actualVisitor.(BadDeclVisitor); ok {
		vis.VisitBadDecl(decl)
	}
}
func (b *helperVisitor) VisitVarDecl(decl *VarDecl) {
	if vis, ok := b.actualVisitor.(VarDeclVisitor); ok {
		vis.VisitVarDecl(decl)
	}
	decl.InitVal.Accept(b)
}
func (b *helperVisitor) VisitFuncDecl(decl *FuncDecl) {
	if vis, ok := b.actualVisitor.(FuncDeclVisitor); ok {
		vis.VisitFuncDecl(decl)
	}
	if decl.Body != nil {
		decl.Body.Accept(b)
	}
}

// if a BadExpr exists the AST is faulty
func (b *helperVisitor) VisitBadExpr(expr *BadExpr) {
	if vis, ok := b.actualVisitor.(BadExprVisitor); ok {
		vis.VisitBadExpr(expr)
	}
}
func (b *helperVisitor) VisitIdent(expr *Ident) {
	if vis, ok := b.actualVisitor.(IdentVisitor); ok {
		vis.VisitIdent(expr)
	}
}
func (b *helperVisitor) VisitIndexing(expr *Indexing) {
	if vis, ok := b.actualVisitor.(IndexingVisitor); ok {
		vis.VisitIndexing(expr)
	}
	expr.Lhs.Accept(b)
	expr.Index.Accept(b)
}

// nothing to do for literals
func (b *helperVisitor) VisitIntLit(expr *IntLit) {
	if vis, ok := b.actualVisitor.(IntLitVisitor); ok {
		vis.VisitIntLit(expr)
	}
}
func (b *helperVisitor) VisitFloatLit(expr *FloatLit) {
	if vis, ok := b.actualVisitor.(FloatLitVisitor); ok {
		vis.VisitFloatLit(expr)
	}
}
func (b *helperVisitor) VisitBoolLit(expr *BoolLit) {
	if vis, ok := b.actualVisitor.(BoolLitVisitor); ok {
		vis.VisitBoolLit(expr)
	}

}
func (b *helperVisitor) VisitCharLit(expr *CharLit) {
	if vis, ok := b.actualVisitor.(CharLitVisitor); ok {
		vis.VisitCharLit(expr)
	}
}
func (b *helperVisitor) VisitStringLit(expr *StringLit) {
	if vis, ok := b.actualVisitor.(StringLitVisitor); ok {
		vis.VisitStringLit(expr)
	}
}
func (b *helperVisitor) VisitListLit(expr *ListLit) {
	if vis, ok := b.actualVisitor.(ListLitVisitor); ok {
		vis.VisitListLit(expr)
	}
	if expr.Values != nil {
		for _, v := range expr.Values {
			v.Accept(b)
		}
	} else if expr.Count != nil && expr.Value != nil {
		expr.Count.Accept(b)
		expr.Value.Accept(b)
	}
}
func (b *helperVisitor) VisitUnaryExpr(expr *UnaryExpr) {
	if vis, ok := b.actualVisitor.(UnaryExprVisitor); ok {
		vis.VisitUnaryExpr(expr)
	}
	expr.Rhs.Accept(b)
}
func (b *helperVisitor) VisitBinaryExpr(expr *BinaryExpr) {
	if vis, ok := b.actualVisitor.(BinaryExprVisitor); ok {
		vis.VisitBinaryExpr(expr)
	}
	expr.Lhs.Accept(b)
	expr.Rhs.Accept(b)
}
func (b *helperVisitor) VisitTernaryExpr(expr *TernaryExpr) {
	if vis, ok := b.actualVisitor.(TernaryExprVisitor); ok {
		vis.VisitTernaryExpr(expr)
	}
	expr.Lhs.Accept(b)
	expr.Mid.Accept(b)
	expr.Rhs.Accept(b)
}
func (b *helperVisitor) VisitCastExpr(expr *CastExpr) {
	if vis, ok := b.actualVisitor.(CastExprVisitor); ok {
		vis.VisitCastExpr(expr)
	}
	expr.Lhs.Accept(b)
}
func (b *helperVisitor) VisitGrouping(expr *Grouping) {
	if vis, ok := b.actualVisitor.(GroupingVisitor); ok {
		vis.VisitGrouping(expr)
	}
	expr.Expr.Accept(b)
}
func (b *helperVisitor) VisitFuncCall(expr *FuncCall) {
	if vis, ok := b.actualVisitor.(FuncCallVisitor); ok {
		vis.VisitFuncCall(expr)
	}
	// visit the passed arguments
	for _, v := range expr.Args {
		v.Accept(b)
	}
}

func (b *helperVisitor) VisitBadStmt(stmt *BadStmt) {
	if vis, ok := b.actualVisitor.(BadStmtVisitor); ok {
		vis.VisitBadStmt(stmt)
	}
}
func (b *helperVisitor) VisitDeclStmt(stmt *DeclStmt) {
	if vis, ok := b.actualVisitor.(DeclStmtVisitor); ok {
		vis.VisitDeclStmt(stmt)
	}
	stmt.Decl.Accept(b)
}
func (b *helperVisitor) VisitExprStmt(stmt *ExprStmt) {
	if vis, ok := b.actualVisitor.(ExprStmtVisitor); ok {
		vis.VisitExprStmt(stmt)
	}
	stmt.Expr.Accept(b)
}
func (b *helperVisitor) VisitAssignStmt(stmt *AssignStmt) {
	if vis, ok := b.actualVisitor.(AssignStmtVisitor); ok {
		vis.VisitAssignStmt(stmt)
	}
	stmt.Var.Accept(b)
	stmt.Rhs.Accept(b)
}
func (b *helperVisitor) VisitBlockStmt(stmt *BlockStmt) {
	if vis, ok := b.actualVisitor.(ScopeVisitor); ok && stmt.Symbols != nil {
		vis.UpdateScope(stmt.Symbols)
	}

	if vis, ok := b.actualVisitor.(BlockStmtVisitor); ok {
		vis.VisitBlockStmt(stmt)
	}
	for _, stmt := range stmt.Statements {
		stmt.Accept(b)
	}

	if vis, ok := b.actualVisitor.(ScopeVisitor); ok && stmt.Symbols != nil {
		vis.UpdateScope(stmt.Symbols.Enclosing)
	}
}
func (b *helperVisitor) VisitIfStmt(stmt *IfStmt) {
	if vis, ok := b.actualVisitor.(IfStmtVisitor); ok {
		vis.VisitIfStmt(stmt)
	}
	stmt.Condition.Accept(b)
	stmt.Then.Accept(b)
	if stmt.Else != nil {
		stmt.Else.Accept(b)
	}
}
func (b *helperVisitor) VisitWhileStmt(stmt *WhileStmt) {
	if vis, ok := b.actualVisitor.(WhileStmtVisitor); ok {
		vis.VisitWhileStmt(stmt)
	}
	switch op := stmt.While.Type; op {
	case token.SOLANGE, token.COUNT_MAL:
		stmt.Condition.Accept(b)
		stmt.Body.Accept(b)
	case token.MACHE:
		stmt.Body.Accept(b)
		stmt.Condition.Accept(b)
	}
}
func (b *helperVisitor) VisitForStmt(stmt *ForStmt) {
	if vis, ok := b.actualVisitor.(ForStmtVisitor); ok {
		vis.VisitForStmt(stmt)
	}

	stmt.Initializer.Accept(b)
	stmt.To.Accept(b)
	if stmt.StepSize != nil {
		stmt.StepSize.Accept(b)
	}
	stmt.Body.Accept(b)
}
func (b *helperVisitor) VisitForRangeStmt(stmt *ForRangeStmt) {
	if vis, ok := b.actualVisitor.(ForRangeStmtVisitor); ok {
		vis.VisitForRangeStmt(stmt)
	}

	stmt.Initializer.Accept(b)
	stmt.In.Accept(b)
	stmt.Body.Accept(b)
}
func (b *helperVisitor) VisitReturnStmt(stmt *ReturnStmt) {
	if vis, ok := b.actualVisitor.(ReturnStmtVisitor); ok {
		vis.VisitReturnStmt(stmt)
	}
	if stmt.Value == nil {
		return
	}
	stmt.Value.Accept(b)
}
