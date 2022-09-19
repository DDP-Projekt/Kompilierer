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
	VisitFloatLit(*FloatLit) Visitor
	VisitBoolLit(*BoolLit) Visitor
	VisitCharLit(*CharLit) Visitor
	VisitStringLit(*StringLit) Visitor
	VisitListLit(*ListLit) Visitor
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

type BaseVisitor interface {
	BaseVisitor()
}

type (
	BadDeclVisitor interface {
		BaseVisitor
		VisitBadDecl(*BadDecl)
	}
	VarDeclVisitor interface {
		BaseVisitor
		VisitVarDecl(*VarDecl)
	}
	FuncDeclVisitor interface {
		BaseVisitor
		VisitFuncDecl(*FuncDecl)
	}

	BadExprVisitor interface {
		BaseVisitor
		VisitBadExpr(*BadExpr)
	}
	IdentVisitor interface {
		BaseVisitor
		VisitIdent(*Ident)
	}
	IndexingVisitor interface {
		BaseVisitor
		VisitIndexing(*Indexing)
	}
	IntLitVisitor interface {
		BaseVisitor
		VisitIntLit(*IntLit)
	}
	FloatLitVisitor interface {
		BaseVisitor
		VisitFloatLit(*FloatLit)
	}
	BoolLitVisitor interface {
		BaseVisitor
		VisitBoolLit(*BoolLit)
	}
	CharLitVisitor interface {
		BaseVisitor
		VisitCharLit(*CharLit)
	}
	StringLitVisitor interface {
		BaseVisitor
		VisitStringLit(*StringLit)
	}
	ListLitVisitor interface {
		BaseVisitor
		VisitListLit(*ListLit)
	}
	UnaryExprVisitor interface {
		BaseVisitor
		VisitUnaryExpr(*UnaryExpr)
	}
	BinaryExprVisitor interface {
		BaseVisitor
		VisitBinaryExpr(*BinaryExpr)
	}
	TernaryExprVisitor interface {
		BaseVisitor
		VisitTernaryExpr(*TernaryExpr)
	}
	CastExprVisitor interface {
		BaseVisitor
		VisitCastExpr(*CastExpr)
	}
	GroupingVisitor interface {
		BaseVisitor
		VisitGrouping(*Grouping)
	}
	FuncCallVisitor interface {
		BaseVisitor
		VisitFuncCall(*FuncCall)
	}

	BadStmtVisitor interface {
		BaseVisitor
		VisitBadStmt(*BadStmt)
	}
	DeclStmtVisitor interface {
		BaseVisitor
		VisitDeclStmt(*DeclStmt)
	}
	ExprStmtVisitor interface {
		BaseVisitor
		VisitExprStmt(*ExprStmt)
	}
	AssignStmtVisitor interface {
		BaseVisitor
		VisitAssignStmt(*AssignStmt)
	}
	BlockStmtVisitor interface {
		BaseVisitor
		VisitBlockStmt(*BlockStmt)
	}
	IfStmtVisitor interface {
		BaseVisitor
		VisitIfStmt(*IfStmt)
	}
	WhileStmtVisitor interface {
		BaseVisitor
		VisitWhileStmt(*WhileStmt)
	}
	ForStmtVisitor interface {
		BaseVisitor
		VisitForStmt(*ForStmt)
	}
	ForRangeStmtVisitor interface {
		BaseVisitor
		VisitForRangeStmt(*ForRangeStmt)
	}
	FuncCallStmtVisitor interface {
		BaseVisitor
		VisitFuncCallStmt(*FuncCallStmt)
	}
	ReturnStmtVisitor interface {
		BaseVisitor
		VisitReturnStmt(*ReturnStmt)
	}
)
