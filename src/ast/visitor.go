package ast

// result type of all Visitor methods
type VisitResult uint8

const (
	VisitRecurse      VisitResult = iota // visiting continues normally
	VisitSkipChildren                    // children of the node are not visited
	VisitBreak                           // visiting is stopped
)

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
	StructDeclVisitor

	/*
		Expressions
	*/

	BadExprVisitor
	IdentVisitor
	IndexingVisitor
	FieldAccessVisitor
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
	StructLiteralVisitor

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
	BreakContineStmtVisitor
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
		VisitBadDecl(*BadDecl) VisitResult
	}
	VarDeclVisitor interface {
		BaseVisitor
		VisitVarDecl(*VarDecl) VisitResult
	}
	FuncDeclVisitor interface {
		BaseVisitor
		VisitFuncDecl(*FuncDecl) VisitResult
	}
	StructDeclVisitor interface {
		BaseVisitor
		VisitStructDecl(*StructDecl) VisitResult
	}

	BadExprVisitor interface {
		BaseVisitor
		VisitBadExpr(*BadExpr) VisitResult
	}
	IdentVisitor interface {
		BaseVisitor
		VisitIdent(*Ident) VisitResult
	}
	IndexingVisitor interface {
		BaseVisitor
		VisitIndexing(*Indexing) VisitResult
	}
	FieldAccessVisitor interface {
		BaseVisitor
		VisitFieldAccess(*FieldAccess) VisitResult
	}
	IntLitVisitor interface {
		BaseVisitor
		VisitIntLit(*IntLit) VisitResult
	}
	FloatLitVisitor interface {
		BaseVisitor
		VisitFloatLit(*FloatLit) VisitResult
	}
	BoolLitVisitor interface {
		BaseVisitor
		VisitBoolLit(*BoolLit) VisitResult
	}
	CharLitVisitor interface {
		BaseVisitor
		VisitCharLit(*CharLit) VisitResult
	}
	StringLitVisitor interface {
		BaseVisitor
		VisitStringLit(*StringLit) VisitResult
	}
	ListLitVisitor interface {
		BaseVisitor
		VisitListLit(*ListLit) VisitResult
	}
	UnaryExprVisitor interface {
		BaseVisitor
		VisitUnaryExpr(*UnaryExpr) VisitResult
	}
	BinaryExprVisitor interface {
		BaseVisitor
		VisitBinaryExpr(*BinaryExpr) VisitResult
	}
	TernaryExprVisitor interface {
		BaseVisitor
		VisitTernaryExpr(*TernaryExpr) VisitResult
	}
	CastExprVisitor interface {
		BaseVisitor
		VisitCastExpr(*CastExpr) VisitResult
	}
	GroupingVisitor interface {
		BaseVisitor
		VisitGrouping(*Grouping) VisitResult
	}
	FuncCallVisitor interface {
		BaseVisitor
		VisitFuncCall(*FuncCall) VisitResult
	}
	StructLiteralVisitor interface {
		BaseVisitor
		VisitStructLiteral(*StructLiteral) VisitResult
	}

	BadStmtVisitor interface {
		BaseVisitor
		VisitBadStmt(*BadStmt) VisitResult
	}
	DeclStmtVisitor interface {
		BaseVisitor
		VisitDeclStmt(*DeclStmt) VisitResult
	}
	ExprStmtVisitor interface {
		BaseVisitor
		VisitExprStmt(*ExprStmt) VisitResult
	}
	ImportStmtVisitor interface {
		BaseVisitor
		VisitImportStmt(*ImportStmt) VisitResult
	}
	AssignStmtVisitor interface {
		BaseVisitor
		VisitAssignStmt(*AssignStmt) VisitResult
	}
	BlockStmtVisitor interface {
		BaseVisitor
		VisitBlockStmt(*BlockStmt) VisitResult
	}
	IfStmtVisitor interface {
		BaseVisitor
		VisitIfStmt(*IfStmt) VisitResult
	}
	WhileStmtVisitor interface {
		BaseVisitor
		VisitWhileStmt(*WhileStmt) VisitResult
	}
	ForStmtVisitor interface {
		BaseVisitor
		VisitForStmt(*ForStmt) VisitResult
	}
	ForRangeStmtVisitor interface {
		BaseVisitor
		VisitForRangeStmt(*ForRangeStmt) VisitResult
	}
	BreakContineStmtVisitor interface {
		BaseVisitor
		VisitBreakContinueStmt(*BreakContinueStmt) VisitResult
	}
	ReturnStmtVisitor interface {
		BaseVisitor
		VisitReturnStmt(*ReturnStmt) VisitResult
	}
)
