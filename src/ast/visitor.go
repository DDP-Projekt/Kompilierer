package ast

// result type of all Visitor methods
type VisitResult uint8

const (
	VisitRecurse      VisitResult = iota // visiting continues normally
	VisitSkipChildren                    // children of the node are not visited
	VisitBreak                           // visiting is stopped
)

// base interface for all visitors
// this interface itself is useless,
// implement one of the sub-interfaces for the actual functionality
type Visitor interface {
	Visitor() // dummy function for the interface
}

// interface for visiting DDP expressions, statements and declarations
// see the Visitor pattern
type FullVisitor interface {
	Visitor
	/*
		Declarations
	*/

	BadDeclVisitor
	VarDeclVisitor
	FuncDeclVisitor
	StructDeclVisitor
	TypeAliasDeclVisitor

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
	TypeOpExprVisitor
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
	BadDeclVisitor interface {
		Visitor
		VisitBadDecl(*BadDecl) VisitResult
	}
	VarDeclVisitor interface {
		Visitor
		VisitVarDecl(*VarDecl) VisitResult
	}
	FuncDeclVisitor interface {
		Visitor
		VisitFuncDecl(*FuncDecl) VisitResult
	}
	StructDeclVisitor interface {
		Visitor
		VisitStructDecl(*StructDecl) VisitResult
	}
	TypeAliasDeclVisitor interface {
		Visitor
		VisitTypeAliasDecl(*TypeAliasDecl) VisitResult
	}

	BadExprVisitor interface {
		Visitor
		VisitBadExpr(*BadExpr) VisitResult
	}
	IdentVisitor interface {
		Visitor
		VisitIdent(*Ident) VisitResult
	}
	IndexingVisitor interface {
		Visitor
		VisitIndexing(*Indexing) VisitResult
	}
	FieldAccessVisitor interface {
		Visitor
		VisitFieldAccess(*FieldAccess) VisitResult
	}
	IntLitVisitor interface {
		Visitor
		VisitIntLit(*IntLit) VisitResult
	}
	FloatLitVisitor interface {
		Visitor
		VisitFloatLit(*FloatLit) VisitResult
	}
	BoolLitVisitor interface {
		Visitor
		VisitBoolLit(*BoolLit) VisitResult
	}
	CharLitVisitor interface {
		Visitor
		VisitCharLit(*CharLit) VisitResult
	}
	StringLitVisitor interface {
		Visitor
		VisitStringLit(*StringLit) VisitResult
	}
	ListLitVisitor interface {
		Visitor
		VisitListLit(*ListLit) VisitResult
	}
	UnaryExprVisitor interface {
		Visitor
		VisitUnaryExpr(*UnaryExpr) VisitResult
	}
	BinaryExprVisitor interface {
		Visitor
		VisitBinaryExpr(*BinaryExpr) VisitResult
	}
	TernaryExprVisitor interface {
		Visitor
		VisitTernaryExpr(*TernaryExpr) VisitResult
	}
	CastExprVisitor interface {
		Visitor
		VisitCastExpr(*CastExpr) VisitResult
	}
	TypeOpExprVisitor interface {
		Visitor
		VisitTypeOpExpr(*TypeOpExpr) VisitResult
	}
	GroupingVisitor interface {
		Visitor
		VisitGrouping(*Grouping) VisitResult
	}
	FuncCallVisitor interface {
		Visitor
		VisitFuncCall(*FuncCall) VisitResult
	}
	StructLiteralVisitor interface {
		Visitor
		VisitStructLiteral(*StructLiteral) VisitResult
	}

	BadStmtVisitor interface {
		Visitor
		VisitBadStmt(*BadStmt) VisitResult
	}
	DeclStmtVisitor interface {
		Visitor
		VisitDeclStmt(*DeclStmt) VisitResult
	}
	ExprStmtVisitor interface {
		Visitor
		VisitExprStmt(*ExprStmt) VisitResult
	}
	ImportStmtVisitor interface {
		Visitor
		VisitImportStmt(*ImportStmt) VisitResult
	}
	AssignStmtVisitor interface {
		Visitor
		VisitAssignStmt(*AssignStmt) VisitResult
	}
	BlockStmtVisitor interface {
		Visitor
		VisitBlockStmt(*BlockStmt) VisitResult
	}
	IfStmtVisitor interface {
		Visitor
		VisitIfStmt(*IfStmt) VisitResult
	}
	WhileStmtVisitor interface {
		Visitor
		VisitWhileStmt(*WhileStmt) VisitResult
	}
	ForStmtVisitor interface {
		Visitor
		VisitForStmt(*ForStmt) VisitResult
	}
	ForRangeStmtVisitor interface {
		Visitor
		VisitForRangeStmt(*ForRangeStmt) VisitResult
	}
	BreakContineStmtVisitor interface {
		Visitor
		VisitBreakContinueStmt(*BreakContinueStmt) VisitResult
	}
	ReturnStmtVisitor interface {
		Visitor
		VisitReturnStmt(*ReturnStmt) VisitResult
	}
)
