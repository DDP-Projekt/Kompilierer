package ast

import (
	"fmt"

	"github.com/DDP-Projekt/Kompilierer/pkg/token"
)

// represents an Abstract Syntax Tree for a token.DDP program
type Ast struct {
	Statements []Statement // the top level statements
	Symbols    *SymbolTable
	Faulty     bool   // set if the ast has any errors (doesn't matter what from which phase they came)
	File       string // the file from which this ast was produced
}

// invoke the Visitor for each top level statement in the Ast
func WalkAst(ast *Ast, visitor Visitor) {
	for _, stmt := range ast.Statements {
		stmt.Accept(visitor)
	}
}

// basic Node interfaces
type (
	Node interface {
		fmt.Stringer
		Token() token.Token
		GetRange() token.Range
		Accept(Visitor) Visitor
	}

	Expression interface {
		Node
		expressionNode() // dummy function for the interface
	}

	Statement interface {
		Node
		statementNode() // dummy function for the interface
	}

	Declaration interface {
		Node
		declarationNode() // dummy function for the interface
	}

	// *Ident or *Indexing
	// Nodes that fulfill this interface can be
	// on the left side of an assignement (meaning, variables or references)
	Assigneable interface {
		Expression
		assigneable() // dummy function for the interface
	}
)

// Declarations
type (
	// an invalid Declaration
	BadDecl struct {
		Range   token.Range
		Tok     token.Token // first token of the bad declaration
		Message string      // error message
	}

	VarDecl struct {
		Range   token.Range
		Type    token.DDPType
		Name    token.Token // identifier name
		InitVal Expression  // initial value
	}

	FuncDecl struct {
		Range      token.Range
		Func       token.Token     // Funktion
		Name       token.Token     // identifier name
		ParamNames []token.Token   // x, y und z
		ParamTypes []token.DDPType // Zahl, Text und Boolean
		Type       token.DDPType   // Zahl Kommazahl nichts ...
		Body       *BlockStmt      // nil for extern functions
		ExternFile token.Token     // string literal with filepath (only pesent if Body is nil)
	}
)

func (decl *BadDecl) String() string  { return "BadDecl" }
func (decl *VarDecl) String() string  { return "VarDecl" }
func (decl *FuncDecl) String() string { return "FuncDecl" }

func (decl *BadDecl) Token() token.Token  { return decl.Tok }
func (decl *VarDecl) Token() token.Token  { return decl.Name }
func (decl *FuncDecl) Token() token.Token { return decl.Func }

func (decl *BadDecl) GetRange() token.Range  { return decl.Range }
func (decl *VarDecl) GetRange() token.Range  { return decl.Range }
func (decl *FuncDecl) GetRange() token.Range { return decl.Range }

func (decl *BadDecl) Accept(visitor Visitor) Visitor  { return visitor.VisitBadDecl(decl) }
func (decl *VarDecl) Accept(visitor Visitor) Visitor  { return visitor.VisitVarDecl(decl) }
func (decl *FuncDecl) Accept(visitor Visitor) Visitor { return visitor.VisitFuncDecl(decl) }

func (decl *BadDecl) declarationNode()  {}
func (decl *VarDecl) declarationNode()  {}
func (decl *FuncDecl) declarationNode() {}

// Expressions
type (
	BadExpr struct {
		Range   token.Range
		Tok     token.Token // first token of the bad expression
		Message string      // error message
	}

	Ident struct {
		Literal token.Token
	}

	// also exists as Binary expression for Literals
	// this one can count as Reference, and may be used
	// inplace of Ident (may be assigned to etc.)
	Indexing struct {
		Name  *Ident // variable Name
		Index Expression
	}

	IntLit struct {
		Literal token.Token
		Value   int64
	}

	FloatLit struct {
		Literal token.Token
		Value   float64 // the parsed float
	}

	BoolLit struct {
		Literal token.Token
		Value   bool
	}

	CharLit struct {
		Literal token.Token
		Value   rune
	}

	StringLit struct {
		Literal token.Token
		Value   string // the evaluated string
	}

	ListLit struct {
		Tok   token.Token
		Range token.Range // should only be used if Values is nil
		// type of the empty list if Values is nil
		// the typechecker fills this field if Values is nil
		Type   token.DDPType
		Values []Expression // nil if it is an empty list
	}

	UnaryExpr struct {
		Range    token.Range
		Operator token.Token
		Rhs      Expression
	}

	BinaryExpr struct {
		Range    token.Range
		Lhs      Expression
		Operator token.Token
		Rhs      Expression
	}

	// currently only used for von bis
	TernaryExpr struct {
		Range    token.Range
		Lhs      Expression
		Mid      Expression
		Rhs      Expression
		Operator token.Token
	}

	// als Expressions cannot be unary
	// because the type operator might be multiple
	// tokens long
	CastExpr struct {
		Range token.Range
		Type  token.DDPType
		Lhs   Expression
	}

	Grouping struct {
		Range  token.Range
		LParen token.Token // (
		Expr   Expression
	}

	FuncCall struct {
		Range token.Range
		Tok   token.Token // first token of the call
		Name  string      // name of the function
		Args  map[string]Expression
	}
)

func (expr *BadExpr) String() string     { return "BadExpr" }
func (expr *Ident) String() string       { return "Ident" }
func (expr *Indexing) String() string    { return "Indexing" }
func (expr *IntLit) String() string      { return "IntLit" }
func (expr *FloatLit) String() string    { return "FloatLit" }
func (expr *BoolLit) String() string     { return "BoolLit" }
func (expr *CharLit) String() string     { return "CharLit" }
func (expr *StringLit) String() string   { return "StringLit" }
func (expr *ListLit) String() string     { return "ListLit" }
func (expr *UnaryExpr) String() string   { return "UnaryExpr" }
func (expr *BinaryExpr) String() string  { return "BinaryExpr" }
func (expr *TernaryExpr) String() string { return "BinaryExpr" }
func (expr *CastExpr) String() string    { return "CastExpr" }
func (expr *Grouping) String() string    { return "Grouping" }
func (expr *FuncCall) String() string    { return "FuncCall" }

func (expr *BadExpr) Token() token.Token     { return expr.Tok }
func (expr *Ident) Token() token.Token       { return expr.Literal }
func (expr *Indexing) Token() token.Token    { return expr.Name.Token() }
func (expr *IntLit) Token() token.Token      { return expr.Literal }
func (expr *FloatLit) Token() token.Token    { return expr.Literal }
func (expr *BoolLit) Token() token.Token     { return expr.Literal }
func (expr *CharLit) Token() token.Token     { return expr.Literal }
func (expr *StringLit) Token() token.Token   { return expr.Literal }
func (expr *ListLit) Token() token.Token     { return expr.Tok }
func (expr *UnaryExpr) Token() token.Token   { return expr.Operator }
func (expr *BinaryExpr) Token() token.Token  { return expr.Operator }
func (expr *TernaryExpr) Token() token.Token { return expr.Operator }
func (expr *CastExpr) Token() token.Token    { return expr.Lhs.Token() }
func (expr *Grouping) Token() token.Token    { return expr.LParen }
func (expr *FuncCall) Token() token.Token    { return expr.Tok }

func (expr *BadExpr) GetRange() token.Range { return expr.Range }
func (expr *Ident) GetRange() token.Range   { return token.NewRange(expr.Literal, expr.Literal) }
func (expr *Indexing) GetRange() token.Range {
	return token.Range{Start: expr.Name.GetRange().Start, End: expr.Index.GetRange().End}
}
func (expr *IntLit) GetRange() token.Range    { return token.NewRange(expr.Literal, expr.Literal) }
func (expr *FloatLit) GetRange() token.Range  { return token.NewRange(expr.Literal, expr.Literal) }
func (expr *BoolLit) GetRange() token.Range   { return token.NewRange(expr.Literal, expr.Literal) }
func (expr *CharLit) GetRange() token.Range   { return token.NewRange(expr.Literal, expr.Literal) }
func (expr *StringLit) GetRange() token.Range { return token.NewRange(expr.Literal, expr.Literal) }
func (expr *ListLit) GetRange() token.Range {
	if expr.Values != nil {
		return token.Range{
			Start: expr.Values[0].GetRange().Start,
			End:   expr.Values[len(expr.Values)-1].GetRange().End,
		}
	}
	return expr.Range
}
func (expr *UnaryExpr) GetRange() token.Range   { return expr.Range }
func (expr *BinaryExpr) GetRange() token.Range  { return expr.Range }
func (expr *TernaryExpr) GetRange() token.Range { return expr.Range }
func (expr *CastExpr) GetRange() token.Range    { return expr.Range }
func (expr *Grouping) GetRange() token.Range    { return expr.Range }
func (expr *FuncCall) GetRange() token.Range    { return expr.Range }

func (expr *BadExpr) Accept(v Visitor) Visitor     { return v.VisitBadExpr(expr) }
func (expr *Ident) Accept(v Visitor) Visitor       { return v.VisitIdent(expr) }
func (expr *Indexing) Accept(v Visitor) Visitor    { return v.VisitIndexing(expr) }
func (expr *IntLit) Accept(v Visitor) Visitor      { return v.VisitIntLit(expr) }
func (expr *FloatLit) Accept(v Visitor) Visitor    { return v.VisitFLoatLit(expr) }
func (expr *BoolLit) Accept(v Visitor) Visitor     { return v.VisitBoolLit(expr) }
func (expr *CharLit) Accept(v Visitor) Visitor     { return v.VisitCharLit(expr) }
func (expr *StringLit) Accept(v Visitor) Visitor   { return v.VisitStringLit(expr) }
func (expr *ListLit) Accept(v Visitor) Visitor     { return v.VisitListLit(expr) }
func (expr *UnaryExpr) Accept(v Visitor) Visitor   { return v.VisitUnaryExpr(expr) }
func (expr *BinaryExpr) Accept(v Visitor) Visitor  { return v.VisitBinaryExpr(expr) }
func (expr *TernaryExpr) Accept(v Visitor) Visitor { return v.VisitTernaryExpr(expr) }
func (expr *CastExpr) Accept(v Visitor) Visitor    { return v.VisitCastExpr(expr) }
func (expr *Grouping) Accept(v Visitor) Visitor    { return v.VisitGrouping(expr) }
func (expr *FuncCall) Accept(v Visitor) Visitor    { return v.VisitFuncCall(expr) }

func (expr *BadExpr) expressionNode()     {}
func (expr *Ident) expressionNode()       {}
func (expr *Indexing) expressionNode()    {}
func (expr *IntLit) expressionNode()      {}
func (expr *FloatLit) expressionNode()    {}
func (expr *BoolLit) expressionNode()     {}
func (expr *CharLit) expressionNode()     {}
func (expr *StringLit) expressionNode()   {}
func (expr *ListLit) expressionNode()     {}
func (expr *UnaryExpr) expressionNode()   {}
func (expr *BinaryExpr) expressionNode()  {}
func (expr *TernaryExpr) expressionNode() {}
func (expr *CastExpr) expressionNode()    {}
func (expr *Grouping) expressionNode()    {}
func (expr *FuncCall) expressionNode()    {}

func (expr *Ident) assigneable()    {}
func (expr *Indexing) assigneable() {}

// Statements
type (
	BadStmt struct {
		Range   token.Range
		Tok     token.Token
		Message string // error message
	}

	DeclStmt struct {
		Decl Declaration
	}

	ExprStmt struct {
		Expr Expression
	}

	AssignStmt struct {
		Range token.Range
		Tok   token.Token
		Var   Assigneable // the variable to assign to
		Rhs   Expression  // literal assign value
	}

	BlockStmt struct {
		Range      token.Range
		Colon      token.Token
		Statements []Statement
		Symbols    *SymbolTable
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
		Body        Statement
	}

	ForRangeStmt struct {
		Range       token.Range
		For         token.Token // Für
		Initializer *VarDecl    // InitVal is the same pointer as In
		In          Expression  // the string/list to range over
		Body        Statement
	}

	FuncCallStmt struct {
		Call *FuncCall
	}

	ReturnStmt struct {
		Range  token.Range
		Return token.Token // Gib
		Func   string
		Value  Expression
	}
)

func (stmt *BadStmt) String() string      { return "BadStmt" }
func (stmt *DeclStmt) String() string     { return "DeclStmt" }
func (stmt *ExprStmt) String() string     { return "ExprStmt" }
func (stmt *AssignStmt) String() string   { return "AssignStmt" }
func (stmt *BlockStmt) String() string    { return "BlockStmt" }
func (stmt *IfStmt) String() string       { return "IfStmt" }
func (stmt *WhileStmt) String() string    { return "WhileStmt" }
func (stmt *ForStmt) String() string      { return "ForStmt" }
func (stmt *ForRangeStmt) String() string { return "ForRangeStmt" }
func (stmt *FuncCallStmt) String() string { return "FuncCallStmt" }
func (stmt *ReturnStmt) String() string   { return "ReturnStmt" }

func (stmt *BadStmt) Token() token.Token      { return stmt.Tok }
func (stmt *DeclStmt) Token() token.Token     { return stmt.Decl.Token() }
func (stmt *ExprStmt) Token() token.Token     { return stmt.Expr.Token() }
func (stmt *AssignStmt) Token() token.Token   { return stmt.Tok }
func (stmt *BlockStmt) Token() token.Token    { return stmt.Colon }
func (stmt *IfStmt) Token() token.Token       { return stmt.If }
func (stmt *WhileStmt) Token() token.Token    { return stmt.While }
func (stmt *ForStmt) Token() token.Token      { return stmt.For }
func (stmt *ForRangeStmt) Token() token.Token { return stmt.For }
func (stmt *FuncCallStmt) Token() token.Token { return stmt.Call.Token() }
func (stmt *ReturnStmt) Token() token.Token   { return stmt.Return }

func (stmt *BadStmt) GetRange() token.Range      { return stmt.Range }
func (stmt *DeclStmt) GetRange() token.Range     { return stmt.Decl.GetRange() }
func (stmt *ExprStmt) GetRange() token.Range     { return stmt.Expr.GetRange() }
func (stmt *AssignStmt) GetRange() token.Range   { return stmt.Range }
func (stmt *BlockStmt) GetRange() token.Range    { return stmt.Range }
func (stmt *IfStmt) GetRange() token.Range       { return stmt.Range }
func (stmt *WhileStmt) GetRange() token.Range    { return stmt.Range }
func (stmt *ForStmt) GetRange() token.Range      { return stmt.Range }
func (stmt *ForRangeStmt) GetRange() token.Range { return stmt.Range }
func (stmt *FuncCallStmt) GetRange() token.Range { return stmt.Call.GetRange() }
func (stmt *ReturnStmt) GetRange() token.Range   { return stmt.Range }

func (stmt *BadStmt) Accept(v Visitor) Visitor      { return v.VisitBadStmt(stmt) }
func (stmt *DeclStmt) Accept(v Visitor) Visitor     { return v.VisitDeclStmt(stmt) }
func (stmt *ExprStmt) Accept(v Visitor) Visitor     { return v.VisitExprStmt(stmt) }
func (stmt *AssignStmt) Accept(v Visitor) Visitor   { return v.VisitAssignStmt(stmt) }
func (stmt *BlockStmt) Accept(v Visitor) Visitor    { return v.VisitBlockStmt(stmt) }
func (stmt *IfStmt) Accept(v Visitor) Visitor       { return v.VisitIfStmt(stmt) }
func (stmt *WhileStmt) Accept(v Visitor) Visitor    { return v.VisitWhileStmt(stmt) }
func (stmt *ForStmt) Accept(v Visitor) Visitor      { return v.VisitForStmt(stmt) }
func (stmt *ForRangeStmt) Accept(v Visitor) Visitor { return v.VisitForRangeStmt(stmt) }
func (stmt *FuncCallStmt) Accept(v Visitor) Visitor { return v.VisitFuncCallStmt(stmt) }
func (stmt *ReturnStmt) Accept(v Visitor) Visitor   { return v.VisitReturnStmt(stmt) }

func (stmt *BadStmt) statementNode()      {}
func (stmt *DeclStmt) statementNode()     {}
func (stmt *ExprStmt) statementNode()     {}
func (stmt *AssignStmt) statementNode()   {}
func (stmt *BlockStmt) statementNode()    {}
func (stmt *IfStmt) statementNode()       {}
func (stmt *WhileStmt) statementNode()    {}
func (stmt *ForStmt) statementNode()      {}
func (stmt *ForRangeStmt) statementNode() {}
func (stmt *FuncCallStmt) statementNode() {}
func (stmt *ReturnStmt) statementNode()   {}
