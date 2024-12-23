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
	FuncDefVisitor
	StructDeclVisitor
	TypeAliasDeclVisitor
	TypeDefDeclVisitor

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
	TypeCheckVisitor
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
	BreakContinueStmtVisitor
	ReturnStmtVisitor
	TodoStmtVisitor
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
	FuncDefVisitor interface {
		Visitor
		VisitFuncDef(*FuncDef) VisitResult
	}
	StructDeclVisitor interface {
		Visitor
		VisitStructDecl(*StructDecl) VisitResult
	}
	TypeAliasDeclVisitor interface {
		Visitor
		VisitTypeAliasDecl(*TypeAliasDecl) VisitResult
	}
	TypeDefDeclVisitor interface {
		Visitor
		VisitTypeDefDecl(*TypeDefDecl) VisitResult
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
	TypeCheckVisitor interface {
		Visitor
		VisitTypeCheck(*TypeCheck) VisitResult
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
	BreakContinueStmtVisitor interface {
		Visitor
		VisitBreakContinueStmt(*BreakContinueStmt) VisitResult
	}
	ReturnStmtVisitor interface {
		Visitor
		VisitReturnStmt(*ReturnStmt) VisitResult
	}
	TodoStmtVisitor interface {
		Visitor
		VisitTodoStmt(*TodoStmt) VisitResult
	}
)

// helper types to easily create small visitors

type BadDeclVisitorFunc func(*BadDecl) VisitResult

var _ BadDeclVisitor = (BadDeclVisitorFunc)(nil)

func (BadDeclVisitorFunc) Visitor() {}
func (f BadDeclVisitorFunc) VisitBadDecl(stmt *BadDecl) VisitResult {
	return f(stmt)
}

type VarDeclVisitorFunc func(*VarDecl) VisitResult

var _ VarDeclVisitor = (VarDeclVisitorFunc)(nil)

func (VarDeclVisitorFunc) Visitor() {}
func (f VarDeclVisitorFunc) VisitVarDecl(stmt *VarDecl) VisitResult {
	return f(stmt)
}

type FuncDeclVisitorFunc func(*FuncDecl) VisitResult

var _ FuncDeclVisitor = (FuncDeclVisitorFunc)(nil)

func (FuncDeclVisitorFunc) Visitor() {}
func (f FuncDeclVisitorFunc) VisitFuncDecl(stmt *FuncDecl) VisitResult {
	return f(stmt)
}

type FuncDefVisitorFunc func(*FuncDef) VisitResult

var _ FuncDefVisitor = (FuncDefVisitorFunc)(nil)

func (FuncDefVisitorFunc) Visitor() {}
func (f FuncDefVisitorFunc) VisitFuncDef(stmt *FuncDef) VisitResult {
	return f(stmt)
}

type StructDeclVisitorFunc func(*StructDecl) VisitResult

var _ StructDeclVisitor = (StructDeclVisitorFunc)(nil)

func (StructDeclVisitorFunc) Visitor() {}
func (f StructDeclVisitorFunc) VisitStructDecl(stmt *StructDecl) VisitResult {
	return f(stmt)
}

type TypeAliasDeclVisitorFunc func(*TypeAliasDecl) VisitResult

var _ TypeAliasDeclVisitor = (TypeAliasDeclVisitorFunc)(nil)

func (TypeAliasDeclVisitorFunc) Visitor() {}
func (f TypeAliasDeclVisitorFunc) VisitTypeAliasDecl(stmt *TypeAliasDecl) VisitResult {
	return f(stmt)
}

type TypeDefDeclVisitorFunc func(*TypeDefDecl) VisitResult

var _ TypeDefDeclVisitor = (TypeDefDeclVisitorFunc)(nil)

func (TypeDefDeclVisitorFunc) Visitor() {}
func (f TypeDefDeclVisitorFunc) VisitTypeDefDecl(stmt *TypeDefDecl) VisitResult {
	return f(stmt)
}

// Expressions
type BadExprVisitorFunc func(*BadExpr) VisitResult

var _ BadExprVisitor = (BadExprVisitorFunc)(nil)

func (BadExprVisitorFunc) Visitor() {}
func (f BadExprVisitorFunc) VisitBadExpr(expr *BadExpr) VisitResult {
	return f(expr)
}

type IdentVisitorFunc func(*Ident) VisitResult

var _ IdentVisitor = (IdentVisitorFunc)(nil)

func (IdentVisitorFunc) Visitor() {}
func (f IdentVisitorFunc) VisitIdent(expr *Ident) VisitResult {
	return f(expr)
}

type IndexingVisitorFunc func(*Indexing) VisitResult

var _ IndexingVisitor = (IndexingVisitorFunc)(nil)

func (IndexingVisitorFunc) Visitor() {}
func (f IndexingVisitorFunc) VisitIndexing(expr *Indexing) VisitResult {
	return f(expr)
}

type FieldAccessVisitorFunc func(*FieldAccess) VisitResult

var _ FieldAccessVisitor = (FieldAccessVisitorFunc)(nil)

func (FieldAccessVisitorFunc) Visitor() {}
func (f FieldAccessVisitorFunc) VisitFieldAccess(expr *FieldAccess) VisitResult {
	return f(expr)
}

type IntLitVisitorFunc func(*IntLit) VisitResult

var _ IntLitVisitor = (IntLitVisitorFunc)(nil)

func (IntLitVisitorFunc) Visitor() {}
func (f IntLitVisitorFunc) VisitIntLit(expr *IntLit) VisitResult {
	return f(expr)
}

type FloatLitVisitorFunc func(*FloatLit) VisitResult

var _ FloatLitVisitor = (FloatLitVisitorFunc)(nil)

func (FloatLitVisitorFunc) Visitor() {}
func (f FloatLitVisitorFunc) VisitFloatLit(expr *FloatLit) VisitResult {
	return f(expr)
}

type BoolLitVisitorFunc func(*BoolLit) VisitResult

var _ BoolLitVisitor = (BoolLitVisitorFunc)(nil)

func (BoolLitVisitorFunc) Visitor() {}
func (f BoolLitVisitorFunc) VisitBoolLit(expr *BoolLit) VisitResult {
	return f(expr)
}

type CharLitVisitorFunc func(*CharLit) VisitResult

var _ CharLitVisitor = (CharLitVisitorFunc)(nil)

func (CharLitVisitorFunc) Visitor() {}
func (f CharLitVisitorFunc) VisitCharLit(expr *CharLit) VisitResult {
	return f(expr)
}

type StringLitVisitorFunc func(*StringLit) VisitResult

var _ StringLitVisitor = (StringLitVisitorFunc)(nil)

func (StringLitVisitorFunc) Visitor() {}
func (f StringLitVisitorFunc) VisitStringLit(expr *StringLit) VisitResult {
	return f(expr)
}

type ListLitVisitorFunc func(*ListLit) VisitResult

var _ ListLitVisitor = (ListLitVisitorFunc)(nil)

func (ListLitVisitorFunc) Visitor() {}
func (f ListLitVisitorFunc) VisitListLit(expr *ListLit) VisitResult {
	return f(expr)
}

type UnaryExprVisitorFunc func(*UnaryExpr) VisitResult

var _ UnaryExprVisitor = (UnaryExprVisitorFunc)(nil)

func (UnaryExprVisitorFunc) Visitor() {}
func (f UnaryExprVisitorFunc) VisitUnaryExpr(expr *UnaryExpr) VisitResult {
	return f(expr)
}

type BinaryExprVisitorFunc func(*BinaryExpr) VisitResult

var _ BinaryExprVisitor = (BinaryExprVisitorFunc)(nil)

func (BinaryExprVisitorFunc) Visitor() {}
func (f BinaryExprVisitorFunc) VisitBinaryExpr(expr *BinaryExpr) VisitResult {
	return f(expr)
}

type TernaryExprVisitorFunc func(*TernaryExpr) VisitResult

var _ TernaryExprVisitor = (TernaryExprVisitorFunc)(nil)

func (TernaryExprVisitorFunc) Visitor() {}
func (f TernaryExprVisitorFunc) VisitTernaryExpr(expr *TernaryExpr) VisitResult {
	return f(expr)
}

type CastExprVisitorFunc func(*CastExpr) VisitResult

var _ CastExprVisitor = (CastExprVisitorFunc)(nil)

func (CastExprVisitorFunc) Visitor() {}
func (f CastExprVisitorFunc) VisitCastExpr(expr *CastExpr) VisitResult {
	return f(expr)
}

type TypeOpExprVisitorFunc func(*TypeOpExpr) VisitResult

var _ TypeOpExprVisitor = (TypeOpExprVisitorFunc)(nil)

func (TypeOpExprVisitorFunc) Visitor() {}
func (f TypeOpExprVisitorFunc) VisitTypeOpExpr(expr *TypeOpExpr) VisitResult {
	return f(expr)
}

type TypeCheckVisitorFunc func(*TypeCheck) VisitResult

var _ TypeCheckVisitor = (TypeCheckVisitorFunc)(nil)

func (TypeCheckVisitorFunc) Visitor() {}
func (f TypeCheckVisitorFunc) VisitTypeCheck(expr *TypeCheck) VisitResult {
	return f(expr)
}

type GroupingVisitorFunc func(*Grouping) VisitResult

var _ GroupingVisitor = (GroupingVisitorFunc)(nil)

func (GroupingVisitorFunc) Visitor() {}
func (f GroupingVisitorFunc) VisitGrouping(expr *Grouping) VisitResult {
	return f(expr)
}

type FuncCallVisitorFunc func(*FuncCall) VisitResult

var _ FuncCallVisitor = (FuncCallVisitorFunc)(nil)

func (FuncCallVisitorFunc) Visitor() {}
func (f FuncCallVisitorFunc) VisitFuncCall(expr *FuncCall) VisitResult {
	return f(expr)
}

type StructLiteralVisitorFunc func(*StructLiteral) VisitResult

var _ StructLiteralVisitor = (StructLiteralVisitorFunc)(nil)

func (StructLiteralVisitorFunc) Visitor() {}
func (f StructLiteralVisitorFunc) VisitStructLiteral(expr *StructLiteral) VisitResult {
	return f(expr)
}

// Statements
type BadStmtVisitorFunc func(*BadStmt) VisitResult

var _ BadStmtVisitor = (BadStmtVisitorFunc)(nil)

func (BadStmtVisitorFunc) Visitor() {}
func (f BadStmtVisitorFunc) VisitBadStmt(stmt *BadStmt) VisitResult {
	return f(stmt)
}

type DeclStmtVisitorFunc func(*DeclStmt) VisitResult

var _ DeclStmtVisitor = (DeclStmtVisitorFunc)(nil)

func (DeclStmtVisitorFunc) Visitor() {}
func (f DeclStmtVisitorFunc) VisitDeclStmt(stmt *DeclStmt) VisitResult {
	return f(stmt)
}

type ExprStmtVisitorFunc func(*ExprStmt) VisitResult

var _ ExprStmtVisitor = (ExprStmtVisitorFunc)(nil)

func (ExprStmtVisitorFunc) Visitor() {}
func (f ExprStmtVisitorFunc) VisitExprStmt(stmt *ExprStmt) VisitResult {
	return f(stmt)
}

type ImportStmtVisitorFunc func(*ImportStmt) VisitResult

var _ ImportStmtVisitor = (ImportStmtVisitorFunc)(nil)

func (ImportStmtVisitorFunc) Visitor() {}
func (f ImportStmtVisitorFunc) VisitImportStmt(stmt *ImportStmt) VisitResult {
	return f(stmt)
}

type AssignStmtVisitorFunc func(*AssignStmt) VisitResult

var _ AssignStmtVisitor = (AssignStmtVisitorFunc)(nil)

func (AssignStmtVisitorFunc) Visitor() {}
func (f AssignStmtVisitorFunc) VisitAssignStmt(stmt *AssignStmt) VisitResult {
	return f(stmt)
}

type BlockStmtVisitorFunc func(*BlockStmt) VisitResult

var _ BlockStmtVisitor = (BlockStmtVisitorFunc)(nil)

func (BlockStmtVisitorFunc) Visitor() {}
func (f BlockStmtVisitorFunc) VisitBlockStmt(stmt *BlockStmt) VisitResult {
	return f(stmt)
}

type IfStmtVisitorFunc func(*IfStmt) VisitResult

var _ IfStmtVisitor = (IfStmtVisitorFunc)(nil)

func (IfStmtVisitorFunc) Visitor() {}
func (f IfStmtVisitorFunc) VisitIfStmt(stmt *IfStmt) VisitResult {
	return f(stmt)
}

type WhileStmtVisitorFunc func(*WhileStmt) VisitResult

var _ WhileStmtVisitor = (WhileStmtVisitorFunc)(nil)

func (WhileStmtVisitorFunc) Visitor() {}
func (f WhileStmtVisitorFunc) VisitWhileStmt(stmt *WhileStmt) VisitResult {
	return f(stmt)
}

type ForStmtVisitorFunc func(*ForStmt) VisitResult

var _ ForStmtVisitor = (ForStmtVisitorFunc)(nil)

func (ForStmtVisitorFunc) Visitor() {}
func (f ForStmtVisitorFunc) VisitForStmt(stmt *ForStmt) VisitResult {
	return f(stmt)
}

type ForRangeStmtVisitorFunc func(*ForRangeStmt) VisitResult

var _ ForRangeStmtVisitor = (ForRangeStmtVisitorFunc)(nil)

func (ForRangeStmtVisitorFunc) Visitor() {}
func (f ForRangeStmtVisitorFunc) VisitForRangeStmt(stmt *ForRangeStmt) VisitResult {
	return f(stmt)
}

type BreakContinueStmtVisitorFunc func(*BreakContinueStmt) VisitResult

var _ BreakContinueStmtVisitor = (BreakContinueStmtVisitorFunc)(nil)

func (BreakContinueStmtVisitorFunc) Visitor() {}
func (f BreakContinueStmtVisitorFunc) VisitBreakContinueStmt(stmt *BreakContinueStmt) VisitResult {
	return f(stmt)
}

type ReturnStmtVisitorFunc func(*ReturnStmt) VisitResult

var _ ReturnStmtVisitor = (ReturnStmtVisitorFunc)(nil)

func (ReturnStmtVisitorFunc) Visitor() {}
func (f ReturnStmtVisitorFunc) VisitReturnStmt(stmt *ReturnStmt) VisitResult {
	return f(stmt)
}

type TodoStmtVisitorFunc func(*TodoStmt) VisitResult

var _ TodoStmtVisitor = (TodoStmtVisitorFunc)(nil)

func (TodoStmtVisitorFunc) Visitor() {}
func (f TodoStmtVisitorFunc) VisitTodoStmt(stmt *TodoStmt) VisitResult {
	return f(stmt)
}
