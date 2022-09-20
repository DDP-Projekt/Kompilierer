package ast

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

func (b *helperVisitor) VisitBadDecl(decl *BadDecl) FullVisitor {
	if vis, ok := b.actualVisitor.(BadDeclVisitor); ok {
		vis.VisitBadDecl(decl)
	}
	return b
}
func (b *helperVisitor) VisitVarDecl(decl *VarDecl) FullVisitor {
	if vis, ok := b.actualVisitor.(VarDeclVisitor); ok {
		vis.VisitVarDecl(decl)
	}
	decl.InitVal.Accept(b)
	return b
}
func (b *helperVisitor) VisitFuncDecl(decl *FuncDecl) FullVisitor {
	if vis, ok := b.actualVisitor.(FuncDeclVisitor); ok {
		vis.VisitFuncDecl(decl)
	}
	if decl.Body != nil {
		decl.Body.Accept(b)
	}
	return b
}

// if a BadExpr exists the AST is faulty
func (b *helperVisitor) VisitBadExpr(expr *BadExpr) FullVisitor {
	if vis, ok := b.actualVisitor.(BadExprVisitor); ok {
		vis.VisitBadExpr(expr)
	}
	return b
}
func (b *helperVisitor) VisitIdent(expr *Ident) FullVisitor {
	if vis, ok := b.actualVisitor.(IdentVisitor); ok {
		vis.VisitIdent(expr)
	}
	return b
}
func (b *helperVisitor) VisitIndexing(expr *Indexing) FullVisitor {
	if vis, ok := b.actualVisitor.(IndexingVisitor); ok {
		vis.VisitIndexing(expr)
	}
	expr.Lhs.Accept(b)
	expr.Index.Accept(b)
	return b
}

// nothing to do for literals
func (b *helperVisitor) VisitIntLit(expr *IntLit) FullVisitor {
	if vis, ok := b.actualVisitor.(IntLitVisitor); ok {
		vis.VisitIntLit(expr)
	}
	return b
}
func (b *helperVisitor) VisitFloatLit(expr *FloatLit) FullVisitor {
	if vis, ok := b.actualVisitor.(FloatLitVisitor); ok {
		vis.VisitFloatLit(expr)
	}
	return b
}
func (b *helperVisitor) VisitBoolLit(expr *BoolLit) FullVisitor {
	if vis, ok := b.actualVisitor.(BoolLitVisitor); ok {
		vis.VisitBoolLit(expr)
	}
	return b
}
func (b *helperVisitor) VisitCharLit(expr *CharLit) FullVisitor {
	if vis, ok := b.actualVisitor.(CharLitVisitor); ok {
		vis.VisitCharLit(expr)
	}
	return b
}
func (b *helperVisitor) VisitStringLit(expr *StringLit) FullVisitor {
	if vis, ok := b.actualVisitor.(StringLitVisitor); ok {
		vis.VisitStringLit(expr)
	}
	return b
}
func (b *helperVisitor) VisitListLit(expr *ListLit) FullVisitor {
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
	return b
}
func (b *helperVisitor) VisitUnaryExpr(expr *UnaryExpr) FullVisitor {
	if vis, ok := b.actualVisitor.(UnaryExprVisitor); ok {
		vis.VisitUnaryExpr(expr)
	}
	expr.Rhs.Accept(b)
	return b
}
func (b *helperVisitor) VisitBinaryExpr(expr *BinaryExpr) FullVisitor {
	if vis, ok := b.actualVisitor.(BinaryExprVisitor); ok {
		vis.VisitBinaryExpr(expr)
	}
	expr.Lhs.Accept(b)
	expr.Rhs.Accept(b)
	return b
}
func (b *helperVisitor) VisitTernaryExpr(expr *TernaryExpr) FullVisitor {
	if vis, ok := b.actualVisitor.(TernaryExprVisitor); ok {
		vis.VisitTernaryExpr(expr)
	}
	expr.Lhs.Accept(b)
	expr.Mid.Accept(b)
	expr.Rhs.Accept(b)
	return b
}
func (b *helperVisitor) VisitCastExpr(expr *CastExpr) FullVisitor {
	if vis, ok := b.actualVisitor.(CastExprVisitor); ok {
		vis.VisitCastExpr(expr)
	}
	expr.Lhs.Accept(b)
	return b
}
func (b *helperVisitor) VisitGrouping(expr *Grouping) FullVisitor {
	if vis, ok := b.actualVisitor.(GroupingVisitor); ok {
		vis.VisitGrouping(expr)
	}
	expr.Expr.Accept(b)
	return b
}
func (b *helperVisitor) VisitFuncCall(expr *FuncCall) FullVisitor {
	if vis, ok := b.actualVisitor.(FuncCallVisitor); ok {
		vis.VisitFuncCall(expr)
	}
	// visit the passed arguments
	for _, v := range expr.Args {
		v.Accept(b)
	}
	return b
}

func (b *helperVisitor) VisitBadStmt(stmt *BadStmt) FullVisitor {
	if vis, ok := b.actualVisitor.(BadStmtVisitor); ok {
		vis.VisitBadStmt(stmt)
	}
	return b
}
func (b *helperVisitor) VisitDeclStmt(stmt *DeclStmt) FullVisitor {
	if vis, ok := b.actualVisitor.(DeclStmtVisitor); ok {
		vis.VisitDeclStmt(stmt)
	}
	stmt.Decl.Accept(b)
	return b
}
func (b *helperVisitor) VisitExprStmt(stmt *ExprStmt) FullVisitor {
	if vis, ok := b.actualVisitor.(ExprStmtVisitor); ok {
		vis.VisitExprStmt(stmt)
	}
	stmt.Expr.Accept(b)
	return b
}
func (b *helperVisitor) VisitAssignStmt(stmt *AssignStmt) FullVisitor {
	if vis, ok := b.actualVisitor.(AssignStmtVisitor); ok {
		vis.VisitAssignStmt(stmt)
	}
	stmt.Var.Accept(b)
	stmt.Rhs.Accept(b)
	return b
}
func (b *helperVisitor) VisitBlockStmt(stmt *BlockStmt) FullVisitor {
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
	return b
}
func (b *helperVisitor) VisitIfStmt(stmt *IfStmt) FullVisitor {
	if vis, ok := b.actualVisitor.(IfStmtVisitor); ok {
		vis.VisitIfStmt(stmt)
	}
	stmt.Condition.Accept(b)
	stmt.Then.Accept(b)
	if stmt.Else != nil {
		stmt.Else.Accept(b)
	}
	return b
}
func (b *helperVisitor) VisitWhileStmt(stmt *WhileStmt) FullVisitor {
	if vis, ok := b.actualVisitor.(WhileStmtVisitor); ok {
		vis.VisitWhileStmt(stmt)
	}
	stmt.Condition.Accept(b)
	stmt.Body.Accept(b)
	return b
}
func (b *helperVisitor) VisitForStmt(stmt *ForStmt) FullVisitor {
	if vis, ok := b.actualVisitor.(ForStmtVisitor); ok {
		vis.VisitForStmt(stmt)
	}

	stmt.Initializer.Accept(b)
	stmt.To.Accept(b)
	if stmt.StepSize != nil {
		stmt.StepSize.Accept(b)
	}
	stmt.Body.Accept(b)

	return b
}
func (b *helperVisitor) VisitForRangeStmt(stmt *ForRangeStmt) FullVisitor {
	if vis, ok := b.actualVisitor.(ForRangeStmtVisitor); ok {
		vis.VisitForRangeStmt(stmt)
	}

	stmt.Initializer.Accept(b)
	stmt.Body.Accept(b)

	return b
}
func (b *helperVisitor) VisitReturnStmt(stmt *ReturnStmt) FullVisitor {
	if vis, ok := b.actualVisitor.(ReturnStmtVisitor); ok {
		vis.VisitReturnStmt(stmt)
	}
	if stmt.Value == nil {
		return b
	}
	stmt.Value.Accept(b)
	return b
}
