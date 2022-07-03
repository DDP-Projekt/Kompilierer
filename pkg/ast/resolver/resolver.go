package resolver

import (
	"fmt"

	"github.com/DDP-Projekt/Kompilierer/pkg/ast"
	"github.com/DDP-Projekt/Kompilierer/pkg/scanner"
	"github.com/DDP-Projekt/Kompilierer/pkg/token"
)

// holds state to resolve the symbols of an AST and its nodes
// and checking if they are valid
// fills the ASTs SymbolTable while doing so
type Resolver struct {
	ErrorHandler scanner.ErrorHandler // function to which errors are passed
	CurrentTable *ast.SymbolTable     // needed state, public for the parser
	Errored      bool                 // wether the resolver errored
}

// create a new resolver to resolve the passed AST
func New(ast *ast.Ast, errorHandler scanner.ErrorHandler) *Resolver {
	return &Resolver{
		ErrorHandler: errorHandler,
		CurrentTable: ast.Symbols,
		Errored:      false,
	}
}

// fills out the asts SymbolTables and reports any errors on the way
func ResolveAst(Ast *ast.Ast, errorHandler scanner.ErrorHandler) {
	Ast.Symbols = ast.NewSymbolTable(nil) // reset the ASTs symbols

	r := New(Ast, errorHandler)

	// visit all nodes of the AST
	for i, l := 0, len(Ast.Statements); i < l; i++ {
		Ast.Statements[i].Accept(r)
	}

	// if the resolver errored, the AST is not valid DDP code
	if r.Errored {
		Ast.Faulty = true
	}
}

// resolve a single node
func (r *Resolver) ResolveNode(node ast.Node) *Resolver {
	return node.Accept(r).(*Resolver)
}

// helper to visit a node
func (r *Resolver) visit(node ast.Node) {
	r = node.Accept(r).(*Resolver)
}

// helper for errors
func (r *Resolver) err(t token.Token, msg string) {
	r.Errored = true
	r.ErrorHandler(t, msg)
}

// if a BadDecl exists the AST is faulty
func (r *Resolver) VisitBadDecl(d *ast.BadDecl) ast.Visitor {
	r.Errored = true
	return r
}
func (r *Resolver) VisitVarDecl(d *ast.VarDecl) ast.Visitor {
	d.InitVal.Accept(r) // resolve the initial value
	// insert the variable into the current scope (SymbolTable)
	if existed := r.CurrentTable.InsertVar(d.Name.Literal, d.Type.Type); existed {
		r.err(d.Token(), fmt.Sprintf("Die Variable '%s' existiert bereits", d.Name.Literal)) // variables may only be declared once in the same scope
	}
	return r
}
func (r *Resolver) VisitFuncDecl(d *ast.FuncDecl) ast.Visitor {
	if existed := r.CurrentTable.InsertFunc(d.Name.Literal, d); existed {
		r.err(d.Token(), fmt.Sprintf("Die Funktion '%s' existiert bereits", d.Name.Literal)) // functions may only be declared once
	}
	d.Body.Symbols = ast.NewSymbolTable(r.CurrentTable) // create a new scope for the function body
	// add the function parameters to the scope of the function body
	for i, l := 0, len(d.ParamNames); i < l; i++ {
		d.Body.Symbols.InsertVar(d.ParamNames[i].Literal, d.ParamTypes[i].Type)
	}
	return d.Body.Accept(r) // resolve the function body
}

// if a BadExpr exists the AST is faulty
func (r *Resolver) VisitBadExpr(e *ast.BadExpr) ast.Visitor {
	r.Errored = true
	return r
}
func (r *Resolver) VisitIdent(e *ast.Ident) ast.Visitor {
	// check if the variable exists
	if _, exists := r.CurrentTable.LookupVar(e.Literal.Literal); !exists {
		r.err(e.Token(), fmt.Sprintf("Der Name '%s' wurde noch nicht als Variable deklariert", e.Literal.Literal))
	}
	return r
}

// nothing to do for literals
func (r *Resolver) VisitIntLit(e *ast.IntLit) ast.Visitor {
	return r
}
func (r *Resolver) VisitFLoatLit(e *ast.FloatLit) ast.Visitor {
	return r
}
func (r *Resolver) VisitBoolLit(e *ast.BoolLit) ast.Visitor {
	return r
}
func (r *Resolver) VisitCharLit(e *ast.CharLit) ast.Visitor {
	return r
}
func (r *Resolver) VisitStringLit(e *ast.StringLit) ast.Visitor {
	return r
}
func (r *Resolver) VisitUnaryExpr(e *ast.UnaryExpr) ast.Visitor {
	return e.Rhs.Accept(r) // visit the actual expression
}
func (r *Resolver) VisitBinaryExpr(e *ast.BinaryExpr) ast.Visitor {
	return e.Rhs.Accept(e.Lhs.Accept(r)) // visit the actual expressions
}
func (r *Resolver) VisitTernaryExpr(e *ast.TernaryExpr) ast.Visitor {
	return e.Rhs.Accept(e.Mid.Accept(e.Lhs.Accept(r))) // visit the actual expressions
}
func (r *Resolver) VisitGrouping(e *ast.Grouping) ast.Visitor {
	return e.Expr.Accept(r)
}
func (r *Resolver) VisitFuncCall(e *ast.FuncCall) ast.Visitor {
	// visit the passed arguments
	for _, v := range e.Args {
		r.visit(v)
	}
	return r
}

// if a BadStmt exists the AST is faulty
func (r *Resolver) VisitBadStmt(s *ast.BadStmt) ast.Visitor {
	r.Errored = true
	return r
}
func (r *Resolver) VisitDeclStmt(s *ast.DeclStmt) ast.Visitor {
	return s.Decl.Accept(r)
}
func (r *Resolver) VisitExprStmt(s *ast.ExprStmt) ast.Visitor {
	return s.Expr.Accept(r)
}
func (r *Resolver) VisitAssignStmt(s *ast.AssignStmt) ast.Visitor {
	// check if the variable exists
	if _, exists := r.CurrentTable.LookupVar(s.Name.Literal); !exists {
		r.err(s.Token(), fmt.Sprintf("Der Name '%s' wurde in noch nicht als Variable deklariert", s.Name.Literal))
	}
	return s.Rhs.Accept(r)
}
func (r *Resolver) VisitBlockStmt(s *ast.BlockStmt) ast.Visitor {
	// a block needs a new scope
	if s.Symbols == nil {
		s.Symbols = ast.NewSymbolTable(r.CurrentTable)
	}
	r.CurrentTable = s.Symbols // set the current scope to the block
	// visit every statement in the block
	for _, stmt := range s.Statements {
		r.visit(stmt)
	}
	r.CurrentTable = s.Symbols.Enclosing
	return r
}
func (r *Resolver) VisitIfStmt(s *ast.IfStmt) ast.Visitor {
	r.visit(s.Condition)
	r.visit(s.Then)
	if s.Else != nil {
		r.visit(s.Else)
	}
	return r
}
func (r *Resolver) VisitWhileStmt(s *ast.WhileStmt) ast.Visitor {
	r.visit(s.Condition)
	return s.Body.Accept(r)
}
func (r *Resolver) VisitForStmt(s *ast.ForStmt) ast.Visitor {
	var env *ast.SymbolTable // scope of the for loop
	// if it contains a block statement, the counter variable needs to go in there
	if body, ok := s.Body.(*ast.BlockStmt); ok {
		body.Symbols = ast.NewSymbolTable(r.CurrentTable)
		env = body.Symbols
	} else { // otherwise, just a new scope
		env = ast.NewSymbolTable(r.CurrentTable)
	}
	r.CurrentTable = env
	r.visit(s.Initializer)
	r.visit(s.To)
	if s.StepSize != nil {
		r.visit(s.StepSize)
	}
	r.visit(s.Body)
	r.CurrentTable = env.Enclosing
	return r
}
func (r *Resolver) VisitForRangeStmt(s *ast.ForRangeStmt) ast.Visitor {
	var env *ast.SymbolTable // scope of the for loop
	// if it contains a block statement, the counter variable needs to go in there
	if body, ok := s.Body.(*ast.BlockStmt); ok {
		body.Symbols = ast.NewSymbolTable(r.CurrentTable)
		env = body.Symbols
	} else { // otherwise, just a new scope
		env = ast.NewSymbolTable(r.CurrentTable)
	}
	r.CurrentTable = env
	r.visit(s.Initializer) // also visits s.In
	r.visit(s.Body)
	r.CurrentTable = env.Enclosing
	return r
}
func (r *Resolver) VisitFuncCallStmt(s *ast.FuncCallStmt) ast.Visitor {
	return s.Call.Accept(r)
}
func (r *Resolver) VisitReturnStmt(s *ast.ReturnStmt) ast.Visitor {
	if _, exists := r.CurrentTable.LookupFunc(s.Func); !exists {
		r.err(s.Token(), "Man kann nur aus Funktionen einen Wert zurückgeben")
	}
	return s.Value.Accept(r)
}
