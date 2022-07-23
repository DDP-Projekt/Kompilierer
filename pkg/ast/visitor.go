package ast

// interface for visiting DDP expressions, statements and declarations
// see the Visitor pattern
type Visitor interface {
	/*
		Declarations
	*/

	VisitBadDecl(*BadDecl) Visitor
	VisitVarDecl(*VarDecl) Visitor
	VisitFuncDecl(*FuncDecl) Visitor

	/*
		Expressions
	*/

	VisitBadExpr(*BadExpr) Visitor
	VisitIdent(*Ident) Visitor
	VisitIndexing(*Indexing) Visitor
	VisitIntLit(*IntLit) Visitor
	VisitFLoatLit(*FloatLit) Visitor
	VisitBoolLit(*BoolLit) Visitor
	VisitCharLit(*CharLit) Visitor
	VisitStringLit(*StringLit) Visitor
	VisitUnaryExpr(*UnaryExpr) Visitor
	VisitBinaryExpr(*BinaryExpr) Visitor
	VisitTernaryExpr(*TernaryExpr) Visitor
	VisitCastExpr(*CastExpr) Visitor
	VisitGrouping(*Grouping) Visitor
	VisitFuncCall(*FuncCall) Visitor

	/*
		Statements
	*/

	VisitBadStmt(*BadStmt) Visitor
	VisitDeclStmt(*DeclStmt) Visitor
	VisitExprStmt(*ExprStmt) Visitor
	VisitAssignStmt(*AssignStmt) Visitor
	VisitBlockStmt(*BlockStmt) Visitor
	VisitIfStmt(*IfStmt) Visitor
	VisitWhileStmt(*WhileStmt) Visitor
	VisitForStmt(*ForStmt) Visitor
	VisitForRangeStmt(*ForRangeStmt) Visitor
	VisitFuncCallStmt(*FuncCallStmt) Visitor
	VisitReturnStmt(*ReturnStmt) Visitor
}
