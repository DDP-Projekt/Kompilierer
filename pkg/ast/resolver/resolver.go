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

	resolver := New(Ast, errorHandler)

	// visit all nodes of the AST
	for i, l := 0, len(Ast.Statements); i < l; i++ {
		Ast.Statements[i].Accept(resolver)
	}

	// if the resolver errored, the AST is not valid DDP code
	if resolver.Errored {
		Ast.Faulty = true
	}
}

// resolve a single node
func (r *Resolver) ResolveNode(node ast.Node) *Resolver {
	return node.Accept(r).(*Resolver)
}

// helper to visit a node
func (r *Resolver) visit(node ast.Node) {
	node.Accept(r)
}

// helper for errors
func (r *Resolver) err(tok token.Token, msg string) {
	r.Errored = true
	r.ErrorHandler(tok, msg)
}

// if a BadDecl exists the AST is faulty
func (r *Resolver) VisitBadDecl(decl *ast.BadDecl) ast.Visitor {
	r.Errored = true
	return r
}
func (r *Resolver) VisitVarDecl(decl *ast.VarDecl) ast.Visitor {
	decl.InitVal.Accept(r) // resolve the initial value
	// insert the variable into the current scope (SymbolTable)
	if existed := r.CurrentTable.InsertVar(decl.Name.Literal, decl.Type); existed {
		r.err(decl.Name, fmt.Sprintf("Die Variable '%s' existiert bereits", decl.Name.Literal)) // variables may only be declared once in the same scope
	}

	return r
}
func (r *Resolver) VisitFuncDecl(decl *ast.FuncDecl) ast.Visitor {
	if existed := r.CurrentTable.InsertFunc(decl.Name.Literal, decl); existed {
		r.err(decl.Name, fmt.Sprintf("Die Funktion '%s' existiert bereits", decl.Name.Literal)) // functions may only be declared once
	}
	if !ast.IsExternFunc(decl) {
		decl.Body.Symbols = ast.NewSymbolTable(r.CurrentTable) // create a new scope for the function body
		// add the function parameters to the scope of the function body
		for i, l := 0, len(decl.ParamNames); i < l; i++ {
			decl.Body.Symbols.InsertVar(decl.ParamNames[i].Literal, decl.ParamTypes[i])
		}

		return decl.Body.Accept(r) // resolve the function body
	}
	return r
}

// if a BadExpr exists the AST is faulty
func (r *Resolver) VisitBadExpr(expr *ast.BadExpr) ast.Visitor {
	r.Errored = true
	return r
}
func (r *Resolver) VisitIdent(expr *ast.Ident) ast.Visitor {
	// check if the variable exists
	if _, exists := r.CurrentTable.LookupVar(expr.Literal.Literal); !exists {
		r.err(expr.Token(), fmt.Sprintf("Der Name '%s' wurde noch nicht als Variable oder Funktions-Alias deklariert", expr.Literal.Literal))
	}
	return r
}
func (r *Resolver) VisitIndexing(expr *ast.Indexing) ast.Visitor {
	r.visit(expr.Name)
	return expr.Index.Accept(r)
}

// nothing to do for literals
func (r *Resolver) VisitIntLit(expr *ast.IntLit) ast.Visitor {
	return r
}
func (r *Resolver) VisitFLoatLit(expr *ast.FloatLit) ast.Visitor {
	return r
}
func (r *Resolver) VisitBoolLit(expr *ast.BoolLit) ast.Visitor {
	return r
}
func (r *Resolver) VisitCharLit(expr *ast.CharLit) ast.Visitor {
	return r
}
func (r *Resolver) VisitStringLit(expr *ast.StringLit) ast.Visitor {
	return r
}
func (r *Resolver) VisitUnaryExpr(expr *ast.UnaryExpr) ast.Visitor {
	return expr.Rhs.Accept(r) // visit the actual expression
}
func (r *Resolver) VisitBinaryExpr(expr *ast.BinaryExpr) ast.Visitor {
	return expr.Rhs.Accept(expr.Lhs.Accept(r)) // visit the actual expressions
}
func (r *Resolver) VisitTernaryExpr(expr *ast.TernaryExpr) ast.Visitor {
	return expr.Rhs.Accept(expr.Mid.Accept(expr.Lhs.Accept(r))) // visit the actual expressions
}
func (r *Resolver) VisitCastExpr(expr *ast.CastExpr) ast.Visitor {
	return expr.Lhs.Accept(r) // visit the actual expressions
}
func (r *Resolver) VisitGrouping(expr *ast.Grouping) ast.Visitor {
	return expr.Expr.Accept(r)
}
func (r *Resolver) VisitFuncCall(expr *ast.FuncCall) ast.Visitor {
	// visit the passed arguments
	for _, v := range expr.Args {
		r.visit(v)
	}
	return r
}

// if a BadStmt exists the AST is faulty
func (r *Resolver) VisitBadStmt(stmt *ast.BadStmt) ast.Visitor {
	r.Errored = true
	return r
}
func (r *Resolver) VisitDeclStmt(stmt *ast.DeclStmt) ast.Visitor {
	return stmt.Decl.Accept(r)
}
func (r *Resolver) VisitExprStmt(stmt *ast.ExprStmt) ast.Visitor {
	return stmt.Expr.Accept(r)
}
func (r *Resolver) VisitAssignStmt(stmt *ast.AssignStmt) ast.Visitor {
	switch assign := stmt.Var.(type) {
	case *ast.Ident:
		// check if the variable exists
		if _, exists := r.CurrentTable.LookupVar(assign.Literal.Literal); !exists {
			r.err(stmt.Token(), fmt.Sprintf("Der Name '%s' wurde in noch nicht als Variable deklariert", assign.Literal.Literal))
		}
	case *ast.Indexing:
		if _, exists := r.CurrentTable.LookupVar(assign.Name.Literal.Literal); !exists {
			r.err(stmt.Token(), fmt.Sprintf("Der Name '%s' wurde in noch nicht als Variable deklariert", assign.Name.Literal.Literal))
		}
	}

	return stmt.Rhs.Accept(r)
}
func (r *Resolver) VisitBlockStmt(stmt *ast.BlockStmt) ast.Visitor {
	// a block needs a new scope
	if stmt.Symbols == nil {
		stmt.Symbols = ast.NewSymbolTable(r.CurrentTable)
	}
	r.CurrentTable = stmt.Symbols // set the current scope to the block

	// visit every statement in the block
	for _, stmt := range stmt.Statements {
		r.visit(stmt)
	}

	r.CurrentTable = stmt.Symbols.Enclosing
	return r
}
func (r *Resolver) VisitIfStmt(stmt *ast.IfStmt) ast.Visitor {
	r.visit(stmt.Condition)
	r.visit(stmt.Then)
	if stmt.Else != nil {
		r.visit(stmt.Else)
	}

	return r
}
func (r *Resolver) VisitWhileStmt(stmt *ast.WhileStmt) ast.Visitor {
	r.visit(stmt.Condition)
	return stmt.Body.Accept(r)
}
func (r *Resolver) VisitForStmt(stmt *ast.ForStmt) ast.Visitor {
	var env *ast.SymbolTable // scope of the for loop
	// if it contains a block statement, the counter variable needs to go in there
	if body, ok := stmt.Body.(*ast.BlockStmt); ok {
		body.Symbols = ast.NewSymbolTable(r.CurrentTable)
		env = body.Symbols
	} else { // otherwise, just a new scope
		env = ast.NewSymbolTable(r.CurrentTable)
	}

	r.CurrentTable = env
	r.visit(stmt.Initializer)
	r.visit(stmt.To)
	if stmt.StepSize != nil {
		r.visit(stmt.StepSize)
	}
	r.visit(stmt.Body)
	r.CurrentTable = env.Enclosing

	return r
}
func (r *Resolver) VisitForRangeStmt(stmt *ast.ForRangeStmt) ast.Visitor {
	var env *ast.SymbolTable // scope of the for loop
	// if it contains a block statement, the counter variable needs to go in there
	if body, ok := stmt.Body.(*ast.BlockStmt); ok {
		body.Symbols = ast.NewSymbolTable(r.CurrentTable)
		env = body.Symbols
	} else { // otherwise, just a new scope
		env = ast.NewSymbolTable(r.CurrentTable)
	}

	r.CurrentTable = env
	r.visit(stmt.Initializer) // also visits stmt.In
	r.visit(stmt.Body)
	r.CurrentTable = env.Enclosing

	return r
}
func (r *Resolver) VisitFuncCallStmt(stmt *ast.FuncCallStmt) ast.Visitor {
	return stmt.Call.Accept(r)
}
func (r *Resolver) VisitReturnStmt(stmt *ast.ReturnStmt) ast.Visitor {
	if _, exists := r.CurrentTable.LookupFunc(stmt.Func); !exists {
		r.err(stmt.Token(), "Man kann nur aus Funktionen einen Wert zurückgeben")
	}
	return stmt.Value.Accept(r)
}
