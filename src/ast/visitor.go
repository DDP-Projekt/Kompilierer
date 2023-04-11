package ast

type BaseVisitor interface {
	BaseVisitor() // dummy function for the interface
}

// interface for visiting DDP expressions, statements and declarations
// see the Visitor pattern
type FullVisitor interface {
	BaseVisitor
	/*
		Declarations
	*/

	BadDeclVisitor
	VarDeclVisitor
	FuncDeclVisitor

	/*
		Expressions
	*/

	BadExprVisitor
	IdentVisitor
	IndexingVisitor
	IntLitVisitor
	FloatLitVisitor
	BoolLitVisitor
	CharLitVisitor
	StringLitVisitor
	ListLitVisitor
	UnaryExprVisitor
	BinaryExprVisitor
	TernaryExprVisitor
	CastExprVisitor
	GroupingVisitor
	FuncCallVisitor

	/*
		Statements
	*/

	BadStmtVisitor
	DeclStmtVisitor
	ExprStmtVisitor
	ImportStmtVisitor
	AssignStmtVisitor
	BlockStmtVisitor
	IfStmtVisitor
	WhileStmtVisitor
	ForStmtVisitor
	ForRangeStmtVisitor
	ReturnStmtVisitor
}

type (
	ScopeVisitor interface {
		UpdateScope(*SymbolTable)
	}
	ConditionalVisitor interface {
		ShouldVisit(Node) bool
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
	ImportStmtVisitor interface {
		BaseVisitor
		VisitImportStmt(*ImportStmt)
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
