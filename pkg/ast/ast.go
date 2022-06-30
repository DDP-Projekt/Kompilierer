package ast

import (
	"fmt"

	"github.com/DDP-Projekt/Kompilierer/pkg/token"
)

// represents an Abstract Syntax Tree for a DDP program
type Ast struct {
	Statements []Statement // the top level statements
	Symbols    *SymbolTable
	Faulty     bool   // set if the ast has any errors (doesn't matter what from which phase they came)
	File       string // the file from which this ast was produced
}

// invoke the Visitor for each top level statement in the Ast
func WalkAst(ast *Ast, v Visitor) {
	for _, stmt := range ast.Statements {
		stmt.Accept(v)
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
)

// Declarations
type (
	// an invalid Declaration
	BadDecl struct {
		Range token.Range
		Tok   token.Token // first token of the bad declaration
	}

	VarDecl struct {
		Range   token.Range
		Type    token.Token // Zahl, Kommazahl etc
		Name    token.Token // identifier name
		InitVal Expression  // initial value
	}

	FuncDecl struct {
		Range      token.Range
		Func       token.Token   // Funktion
		Name       token.Token   // identifier name
		ParamNames []token.Token // x, y und z
		ParamTypes []token.Token // Zahl, Text und Boolean
		Type       token.Token   // Zahl Kommazahl nichts ...
		Body       *BlockStmt
	}
)

func (d *BadDecl) String() string  { return "BadDecl" }
func (d *VarDecl) String() string  { return "VarDecl" }
func (d *FuncDecl) String() string { return "FuncDecl" }

func (d *BadDecl) Token() token.Token  { return d.Tok }
func (d *VarDecl) Token() token.Token  { return d.Type }
func (d *FuncDecl) Token() token.Token { return d.Func }

func (d *BadDecl) GetRange() token.Range  { return d.Range }
func (d *VarDecl) GetRange() token.Range  { return d.Range }
func (d *FuncDecl) GetRange() token.Range { return d.Range }

func (d *BadDecl) Accept(v Visitor) Visitor  { return v.VisitBadDecl(d) }
func (d *VarDecl) Accept(v Visitor) Visitor  { return v.VisitVarDecl(d) }
func (d *FuncDecl) Accept(v Visitor) Visitor { return v.VisitFuncDecl(d) }

func (d *BadDecl) declarationNode()  {}
func (d *VarDecl) declarationNode()  {}
func (d *FuncDecl) declarationNode() {}

// Expressions
type (
	BadExpr struct {
		Range token.Range
		Tok   token.Token // first token of the bad expression
	}

	Ident struct {
		Literal token.Token
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

func (e *BadExpr) String() string     { return "BadExpr" }
func (e *Ident) String() string       { return "Ident" }
func (e *IntLit) String() string      { return "IntLit" }
func (e *FloatLit) String() string    { return "FloatLit" }
func (e *BoolLit) String() string     { return "BoolLit" }
func (e *CharLit) String() string     { return "CharLit" }
func (e *StringLit) String() string   { return "StringLit" }
func (e *UnaryExpr) String() string   { return "UnaryExpr" }
func (e *BinaryExpr) String() string  { return "BinaryExpr" }
func (e *TernaryExpr) String() string { return "BinaryExpr" }
func (e *Grouping) String() string    { return "Grouping" }
func (e *FuncCall) String() string    { return "FuncCall" }

func (e *BadExpr) Token() token.Token     { return e.Tok }
func (e *Ident) Token() token.Token       { return e.Literal }
func (e *IntLit) Token() token.Token      { return e.Literal }
func (e *FloatLit) Token() token.Token    { return e.Literal }
func (e *BoolLit) Token() token.Token     { return e.Literal }
func (e *CharLit) Token() token.Token     { return e.Literal }
func (e *StringLit) Token() token.Token   { return e.Literal }
func (e *UnaryExpr) Token() token.Token   { return e.Operator }
func (e *BinaryExpr) Token() token.Token  { return e.Operator }
func (e *TernaryExpr) Token() token.Token { return e.Operator }
func (e *Grouping) Token() token.Token    { return e.LParen }
func (e *FuncCall) Token() token.Token    { return e.Tok }

func (e *BadExpr) GetRange() token.Range     { return e.Range }
func (e *Ident) GetRange() token.Range       { return token.NewRange(e.Literal, e.Literal) }
func (e *IntLit) GetRange() token.Range      { return token.NewRange(e.Literal, e.Literal) }
func (e *FloatLit) GetRange() token.Range    { return token.NewRange(e.Literal, e.Literal) }
func (e *BoolLit) GetRange() token.Range     { return token.NewRange(e.Literal, e.Literal) }
func (e *CharLit) GetRange() token.Range     { return token.NewRange(e.Literal, e.Literal) }
func (e *StringLit) GetRange() token.Range   { return token.NewRange(e.Literal, e.Literal) }
func (e *UnaryExpr) GetRange() token.Range   { return e.Range }
func (e *BinaryExpr) GetRange() token.Range  { return e.Range }
func (e *TernaryExpr) GetRange() token.Range { return e.Range }
func (e *Grouping) GetRange() token.Range    { return e.Range }
func (e *FuncCall) GetRange() token.Range    { return e.Range }

func (e *BadExpr) Accept(v Visitor) Visitor     { return v.VisitBadExpr(e) }
func (e *Ident) Accept(v Visitor) Visitor       { return v.VisitIdent(e) }
func (e *IntLit) Accept(v Visitor) Visitor      { return v.VisitIntLit(e) }
func (e *FloatLit) Accept(v Visitor) Visitor    { return v.VisitFLoatLit(e) }
func (e *BoolLit) Accept(v Visitor) Visitor     { return v.VisitBoolLit(e) }
func (e *CharLit) Accept(v Visitor) Visitor     { return v.VisitCharLit(e) }
func (e *StringLit) Accept(v Visitor) Visitor   { return v.VisitStringLit(e) }
func (e *UnaryExpr) Accept(v Visitor) Visitor   { return v.VisitUnaryExpr(e) }
func (e *BinaryExpr) Accept(v Visitor) Visitor  { return v.VisitBinaryExpr(e) }
func (e *TernaryExpr) Accept(v Visitor) Visitor { return v.VisitTernaryExpr(e) }
func (e *Grouping) Accept(v Visitor) Visitor    { return v.VisitGrouping(e) }
func (e *FuncCall) Accept(v Visitor) Visitor    { return v.VisitFuncCall(e) }

func (e *BadExpr) expressionNode()     {}
func (e *Ident) expressionNode()       {}
func (e *IntLit) expressionNode()      {}
func (e *FloatLit) expressionNode()    {}
func (e *BoolLit) expressionNode()     {}
func (e *CharLit) expressionNode()     {}
func (e *StringLit) expressionNode()   {}
func (e *UnaryExpr) expressionNode()   {}
func (e *BinaryExpr) expressionNode()  {}
func (e *TernaryExpr) expressionNode() {}
func (e *Grouping) expressionNode()    {}
func (e *FuncCall) expressionNode()    {}

// Statements
type (
	BadStmt struct {
		Range token.Range
		Tok   token.Token
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
		Name  token.Token // name of the variable
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

func (s *BadStmt) String() string      { return "BadStmt" }
func (s *DeclStmt) String() string     { return "DeclStmt" }
func (s *ExprStmt) String() string     { return "ExprStmt" }
func (s *AssignStmt) String() string   { return "AssignStmt" }
func (s *BlockStmt) String() string    { return "BlockStmt" }
func (s *IfStmt) String() string       { return "IfStmt" }
func (s *WhileStmt) String() string    { return "WhileStmt" }
func (s *ForStmt) String() string      { return "ForStmt" }
func (s *ForRangeStmt) String() string { return "ForRangeStmt" }
func (s *FuncCallStmt) String() string { return "FuncCallStmt" }
func (s *ReturnStmt) String() string   { return "ReturnStmt" }

func (s *BadStmt) Token() token.Token      { return s.Tok }
func (s *DeclStmt) Token() token.Token     { return s.Decl.Token() }
func (s *ExprStmt) Token() token.Token     { return s.Expr.Token() }
func (s *AssignStmt) Token() token.Token   { return s.Tok }
func (s *BlockStmt) Token() token.Token    { return s.Colon }
func (s *IfStmt) Token() token.Token       { return s.If }
func (s *WhileStmt) Token() token.Token    { return s.While }
func (s *ForStmt) Token() token.Token      { return s.For }
func (s *ForRangeStmt) Token() token.Token { return s.For }
func (s *FuncCallStmt) Token() token.Token { return s.Call.Token() }
func (s *ReturnStmt) Token() token.Token   { return s.Return }

func (s *BadStmt) GetRange() token.Range      { return s.Range }
func (s *DeclStmt) GetRange() token.Range     { return s.Decl.GetRange() }
func (s *ExprStmt) GetRange() token.Range     { return s.Expr.GetRange() }
func (s *AssignStmt) GetRange() token.Range   { return s.Range }
func (s *BlockStmt) GetRange() token.Range    { return s.Range }
func (s *IfStmt) GetRange() token.Range       { return s.Range }
func (s *WhileStmt) GetRange() token.Range    { return s.Range }
func (s *ForStmt) GetRange() token.Range      { return s.Range }
func (s *ForRangeStmt) GetRange() token.Range { return s.Range }
func (s *FuncCallStmt) GetRange() token.Range { return s.Call.GetRange() }
func (s *ReturnStmt) GetRange() token.Range   { return s.Range }

func (s *BadStmt) Accept(v Visitor) Visitor      { return v.VisitBadStmt(s) }
func (s *DeclStmt) Accept(v Visitor) Visitor     { return v.VisitDeclStmt(s) }
func (s *ExprStmt) Accept(v Visitor) Visitor     { return v.VisitExprStmt(s) }
func (s *AssignStmt) Accept(v Visitor) Visitor   { return v.VisitAssignStmt(s) }
func (s *BlockStmt) Accept(v Visitor) Visitor    { return v.VisitBlockStmt(s) }
func (s *IfStmt) Accept(v Visitor) Visitor       { return v.VisitIfStmt(s) }
func (s *WhileStmt) Accept(v Visitor) Visitor    { return v.VisitWhileStmt(s) }
func (s *ForStmt) Accept(v Visitor) Visitor      { return v.VisitForStmt(s) }
func (s *ForRangeStmt) Accept(v Visitor) Visitor { return v.VisitForRangeStmt(s) }
func (s *FuncCallStmt) Accept(v Visitor) Visitor { return v.VisitFuncCallStmt(s) }
func (s *ReturnStmt) Accept(v Visitor) Visitor   { return v.VisitReturnStmt(s) }

func (s *BadStmt) statementNode()      {}
func (s *DeclStmt) statementNode()     {}
func (s *ExprStmt) statementNode()     {}
func (s *AssignStmt) statementNode()   {}
func (s *BlockStmt) statementNode()    {}
func (s *IfStmt) statementNode()       {}
func (s *WhileStmt) statementNode()    {}
func (s *ForStmt) statementNode()      {}
func (s *ForRangeStmt) statementNode() {}
func (s *FuncCallStmt) statementNode() {}
func (s *ReturnStmt) statementNode()   {}
