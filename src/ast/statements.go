package ast

import (
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
	"github.com/DDP-Projekt/Kompilierer/src/token"
)

type (
	BadStmt struct {
		Tok token.Token
		Err ddperror.Error
	}

	DeclStmt struct {
		Decl Declaration
	}

	ExprStmt struct {
		Expr Expression
	}

	// import statement for meta-information in the ast
	// will be already resolved by the parser
	ImportStmt struct {
		Range token.Range
		// wether the statement imported a directory
		IsDirectoryImport bool
		// wether the statement was a recursive import
		// only true if IsDirectoryImport is also true
		IsRecursive bool
		// the string literal which specified the filename
		FileName token.Token
		// all modules that were imported because of this statement
		// nil if it does not exist or a similar error occured while importing
		Modules []*Module
		// slice of identifiers which specify
		// the individual symbols imported
		// if nil, all symbols are imported
		ImportedSymbols []token.Token
	}

	AssignStmt struct {
		Range   token.Range
		Tok     token.Token
		Var     Assigneable   // the variable to assign to
		Rhs     Expression    // literal assign value
		RhsType ddptypes.Type // filled in by the typechecker, to keep information about typedefs
	}

	BlockStmt struct {
		Range      token.Range
		Colon      token.Token
		Statements []Statement
		Symbols    SymbolTable
	}

	IfStmt struct {
		Range     token.Range
		If        token.Token // wenn/aber
		Condition Expression
		Then      Statement
		Else      Statement
	}

	WhileStmt struct {
		Range     token.Range
		While     token.Token // solange, mache, mal
		Condition Expression
		Body      Statement
	}

	ForStmt struct {
		Range       token.Range
		For         token.Token // Für
		Initializer *VarDecl    // Zahl (name) von (Initializer.InitVal)
		To          Expression  // bis (To)
		StepSize    Expression  // Schrittgröße
		Body        *BlockStmt
	}

	ForRangeStmt struct {
		Range       token.Range
		For         token.Token // Für
		Initializer *VarDecl    // InitVal is the same pointer as In
		Index       *VarDecl    // optional index during iteration
		In          Expression  // the string/list to range over
		Body        *BlockStmt
	}

	BreakContinueStmt struct {
		Range token.Range
		Tok   token.Token // VERLASSE for break, otherwise continue
	}

	ReturnStmt struct {
		Range  token.Range
		Return token.Token // Gib
		Func   *FuncDecl
		Value  Expression // nil for void return
	}

	TodoStmt struct {
		Tok token.Token // ...
	}
)

func (stmt *BadStmt) node()           {}
func (stmt *DeclStmt) node()          {}
func (stmt *ExprStmt) node()          {}
func (stmt *ImportStmt) node()        {}
func (stmt *AssignStmt) node()        {}
func (stmt *BlockStmt) node()         {}
func (stmt *IfStmt) node()            {}
func (stmt *WhileStmt) node()         {}
func (stmt *ForStmt) node()           {}
func (stmt *ForRangeStmt) node()      {}
func (stmt *BreakContinueStmt) node() {}
func (stmt *ReturnStmt) node()        {}
func (stmt *TodoStmt) node()          {}

func (stmt *BadStmt) String() string           { return "BadStmt" }
func (stmt *DeclStmt) String() string          { return "DeclStmt" }
func (stmt *ExprStmt) String() string          { return "ExprStmt" }
func (stmt *ImportStmt) String() string        { return "ImportStmt" }
func (stmt *AssignStmt) String() string        { return "AssignStmt" }
func (stmt *BlockStmt) String() string         { return "BlockStmt" }
func (stmt *IfStmt) String() string            { return "IfStmt" }
func (stmt *WhileStmt) String() string         { return "WhileStmt" }
func (stmt *ForStmt) String() string           { return "ForStmt" }
func (stmt *ForRangeStmt) String() string      { return "ForRangeStmt" }
func (stmt *BreakContinueStmt) String() string { return "BreakContinueStmt" }
func (stmt *ReturnStmt) String() string        { return "ReturnStmt" }
func (stmt *TodoStmt) String() string          { return "TodoStmt" }

func (stmt *BadStmt) Token() token.Token           { return stmt.Tok }
func (stmt *DeclStmt) Token() token.Token          { return stmt.Decl.Token() }
func (stmt *ExprStmt) Token() token.Token          { return stmt.Expr.Token() }
func (stmt *ImportStmt) Token() token.Token        { return stmt.FileName }
func (stmt *AssignStmt) Token() token.Token        { return stmt.Tok }
func (stmt *BlockStmt) Token() token.Token         { return stmt.Colon }
func (stmt *IfStmt) Token() token.Token            { return stmt.If }
func (stmt *WhileStmt) Token() token.Token         { return stmt.While }
func (stmt *ForStmt) Token() token.Token           { return stmt.For }
func (stmt *ForRangeStmt) Token() token.Token      { return stmt.For }
func (stmt *BreakContinueStmt) Token() token.Token { return stmt.Tok }
func (stmt *ReturnStmt) Token() token.Token        { return stmt.Return }
func (stmt *TodoStmt) Token() token.Token          { return stmt.Tok }

func (stmt *BadStmt) GetRange() token.Range           { return stmt.Err.Range }
func (stmt *DeclStmt) GetRange() token.Range          { return stmt.Decl.GetRange() }
func (stmt *ExprStmt) GetRange() token.Range          { return stmt.Expr.GetRange() }
func (stmt *ImportStmt) GetRange() token.Range        { return stmt.Range }
func (stmt *AssignStmt) GetRange() token.Range        { return stmt.Range }
func (stmt *BlockStmt) GetRange() token.Range         { return stmt.Range }
func (stmt *IfStmt) GetRange() token.Range            { return stmt.Range }
func (stmt *WhileStmt) GetRange() token.Range         { return stmt.Range }
func (stmt *ForStmt) GetRange() token.Range           { return stmt.Range }
func (stmt *ForRangeStmt) GetRange() token.Range      { return stmt.Range }
func (stmt *BreakContinueStmt) GetRange() token.Range { return stmt.Range }
func (stmt *ReturnStmt) GetRange() token.Range        { return stmt.Range }
func (stmt *TodoStmt) GetRange() token.Range          { return stmt.Tok.Range }

func (stmt *BadStmt) Accept(v FullVisitor) VisitResult      { return v.VisitBadStmt(stmt) }
func (stmt *DeclStmt) Accept(v FullVisitor) VisitResult     { return v.VisitDeclStmt(stmt) }
func (stmt *ExprStmt) Accept(v FullVisitor) VisitResult     { return v.VisitExprStmt(stmt) }
func (stmt *ImportStmt) Accept(v FullVisitor) VisitResult   { return v.VisitImportStmt(stmt) }
func (stmt *AssignStmt) Accept(v FullVisitor) VisitResult   { return v.VisitAssignStmt(stmt) }
func (stmt *BlockStmt) Accept(v FullVisitor) VisitResult    { return v.VisitBlockStmt(stmt) }
func (stmt *IfStmt) Accept(v FullVisitor) VisitResult       { return v.VisitIfStmt(stmt) }
func (stmt *WhileStmt) Accept(v FullVisitor) VisitResult    { return v.VisitWhileStmt(stmt) }
func (stmt *ForStmt) Accept(v FullVisitor) VisitResult      { return v.VisitForStmt(stmt) }
func (stmt *ForRangeStmt) Accept(v FullVisitor) VisitResult { return v.VisitForRangeStmt(stmt) }
func (stmt *BreakContinueStmt) Accept(v FullVisitor) VisitResult {
	return v.VisitBreakContinueStmt(stmt)
}
func (stmt *ReturnStmt) Accept(v FullVisitor) VisitResult { return v.VisitReturnStmt(stmt) }
func (stmt *TodoStmt) Accept(v FullVisitor) VisitResult   { return v.VisitTodoStmt(stmt) }

func (stmt *BadStmt) statementNode()           {}
func (stmt *DeclStmt) statementNode()          {}
func (stmt *ExprStmt) statementNode()          {}
func (stmt *ImportStmt) statementNode()        {}
func (stmt *AssignStmt) statementNode()        {}
func (stmt *BlockStmt) statementNode()         {}
func (stmt *IfStmt) statementNode()            {}
func (stmt *WhileStmt) statementNode()         {}
func (stmt *ForStmt) statementNode()           {}
func (stmt *ForRangeStmt) statementNode()      {}
func (stmt *BreakContinueStmt) statementNode() {}
func (stmt *ReturnStmt) statementNode()        {}
func (stmt *TodoStmt) statementNode()          {}

// If this statement was not a directory import, returns the one imported module
func (stmt *ImportStmt) SingleModule() *Module {
	if stmt.IsDirectoryImport || len(stmt.Modules) == 0 {
		return nil
	}

	return stmt.Modules[0]
}
