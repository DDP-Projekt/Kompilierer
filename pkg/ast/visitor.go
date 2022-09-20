package ast

// interface for visiting DDP expressions, statements and declarations
// see the Visitor pattern
type FullVisitor interface {
	/*
		Declarations
	*/

	VisitBadDecl(*BadDecl) FullVisitor
	VisitVarDecl(*VarDecl) FullVisitor
	VisitFuncDecl(*FuncDecl) FullVisitor

	/*
		Expressions
	*/

	VisitBadExpr(*BadExpr) FullVisitor
	VisitIdent(*Ident) FullVisitor
	VisitIndexing(*Indexing) FullVisitor
	VisitIntLit(*IntLit) FullVisitor
	VisitFloatLit(*FloatLit) FullVisitor
	VisitBoolLit(*BoolLit) FullVisitor
	VisitCharLit(*CharLit) FullVisitor
	VisitStringLit(*StringLit) FullVisitor
	VisitListLit(*ListLit) FullVisitor
	VisitUnaryExpr(*UnaryExpr) FullVisitor
	VisitBinaryExpr(*BinaryExpr) FullVisitor
	VisitTernaryExpr(*TernaryExpr) FullVisitor
	VisitCastExpr(*CastExpr) FullVisitor
	VisitGrouping(*Grouping) FullVisitor
	VisitFuncCall(*FuncCall) FullVisitor

	/*
		Statements
	*/

	VisitBadStmt(*BadStmt) FullVisitor
	VisitDeclStmt(*DeclStmt) FullVisitor
	VisitExprStmt(*ExprStmt) FullVisitor
	VisitAssignStmt(*AssignStmt) FullVisitor
	VisitBlockStmt(*BlockStmt) FullVisitor
	VisitIfStmt(*IfStmt) FullVisitor
	VisitWhileStmt(*WhileStmt) FullVisitor
	VisitForStmt(*ForStmt) FullVisitor
	VisitForRangeStmt(*ForRangeStmt) FullVisitor
	VisitReturnStmt(*ReturnStmt) FullVisitor
}

type (
	BaseVisitor interface {
		VisitNode(Node)
	}
	ScopeVisitor interface {
		UpdateScope(*SymbolTable)
	}
)

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
	ReturnStmtVisitor interface {
		BaseVisitor
		VisitReturnStmt(*ReturnStmt)
	}
)
