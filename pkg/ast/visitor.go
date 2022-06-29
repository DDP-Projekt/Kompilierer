package ast

// interface for visiting DDP expressions, statements and declarations
// see the Visitor pattern
type Visitor interface {
	VisitBadDecl(*BadDecl) Visitor
	VisitVarDecl(*VarDecl) Visitor
	VisitFuncDecl(*FuncDecl) Visitor

	VisitBadExpr(*BadExpr) Visitor
	VisitIdent(*Ident) Visitor
	VisitIntLit(*IntLit) Visitor
	VisitFLoatLit(*FloatLit) Visitor
	VisitBoolLit(*BoolLit) Visitor
	VisitCharLit(*CharLit) Visitor
	VisitStringLit(*StringLit) Visitor
	VisitUnaryExpr(*UnaryExpr) Visitor
	VisitBinaryExpr(*BinaryExpr) Visitor
	VisitTernaryExpr(*TernaryExpr) Visitor
	VisitGrouping(*Grouping) Visitor
	VisitFuncCall(*FuncCall) Visitor

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
