/*
The resolver package should not be used independently from the parser.
It is not a complete Visitor itself, but is rather used to resolve single
Nodes while parsing to ensure correct parsing of function-calls etc.

Because of this you will find many r.visit(x) calls to be commented out.
*/
package resolver

import (
	"fmt"

	"github.com/DDP-Projekt/Kompilierer/pkg/ast"
	"github.com/DDP-Projekt/Kompilierer/pkg/ddperror"
	"github.com/DDP-Projekt/Kompilierer/pkg/token"
)

// holds state to resolve the symbols of an AST and its nodes
// and checking if they are valid
// fills the ASTs SymbolTable while doing so
type Resolver struct {
	ErrorHandler ddperror.Handler // function to which errors are passed
	CurrentTable *ast.SymbolTable // needed state, public for the parser
	Errored      bool             // wether the resolver errored
}

// create a new resolver to resolve the passed AST
func New(Ast *ast.Ast, errorHandler ddperror.Handler) *Resolver {
	if errorHandler == nil {
		errorHandler = ddperror.EmptyHandler
	}
	return &Resolver{
		ErrorHandler: errorHandler,
		CurrentTable: Ast.Symbols,
		Errored:      false,
	}
}

// fills out the asts SymbolTables and reports any errors on the way
func ResolveAst(Ast *ast.Ast, errorHandler ddperror.Handler) {
	Ast.Symbols = ast.NewSymbolTable(nil) // reset the ASTs symbols

	resolver := New(Ast, errorHandler)

	// visit all nodes of the AST
	for i, l := 0, len(Ast.Statements); i < l; i++ {
		resolver.visit(Ast.Statements[i])
	}

	// if the resolver errored, the AST is not valid DDP code
	if resolver.Errored {
		Ast.Faulty = true
	}
}

// resolve a single node
func (r *Resolver) ResolveNode(node ast.Node) {
	r.visit(node)
}

// helper to visit a node
func (r *Resolver) visit(node ast.Node) {
	node.Accept(r)
}

func (r *Resolver) setScope(symbols *ast.SymbolTable) {
	r.CurrentTable = symbols
}

func (r *Resolver) exitScope() {
	r.CurrentTable = r.CurrentTable.Enclosing
}

// helper for errors
func (r *Resolver) err(code ddperror.Code, Range token.Range, msg string, file string) {
	r.Errored = true
	r.ErrorHandler(ddperror.New(code, Range, msg, file))
}

func (*Resolver) BaseVisitor() {}

// if a BadDecl exists the AST is faulty
func (r *Resolver) VisitBadDecl(decl *ast.BadDecl) {
	r.Errored = true
}
func (r *Resolver) VisitVarDecl(decl *ast.VarDecl) {
	r.visit(decl.InitVal) // resolve the initial value
	// insert the variable into the current scope (SymbolTable)
	if existed := r.CurrentTable.InsertDecl(decl.Name.Literal, decl); existed {
		r.err(ddperror.SEM_NAME_ALREADY_DEFINED, decl.Name.Range, ddperror.MsgNameAlreadyExists(decl.Name.Literal), decl.Name.File) // variables may only be declared once in the same scope
	}
}
func (r *Resolver) VisitFuncDecl(decl *ast.FuncDecl) {
	// all of the below was already resolved by the parser

	/*
		if existed := r.CurrentTable.InsertFunc(decl.Name.Literal, decl); existed {
			r.err(decl.Name, "Die Funktion '%s' existiert bereits", decl.Name.Literal) // functions may only be declared once
		}
		if !ast.IsExternFunc(decl) {
			decl.Body.Symbols = ast.NewSymbolTable(r.CurrentTable) // create a new scope for the function body
			// add the function parameters to the scope of the function body
			for i, l := 0, len(decl.ParamNames); i < l; i++ {
				decl.Body.Symbols.InsertVar(decl.ParamNames[i].Literal, &ast.VarDecl{Name: decl.ParamNames[i], Type: decl.ParamTypes[i].Type, Range: token.NewRange(decl.ParamNames[i], decl.ParamNames[i])})
			}

			r.visit(decl.Body) // resolve the function body
		}
	*/
}

// if a BadExpr exists the AST is faulty
func (r *Resolver) VisitBadExpr(expr *ast.BadExpr) {
	r.Errored = true
}
func (r *Resolver) VisitIdent(expr *ast.Ident) {
	// check if the variable exists
	if _, exists, isVar := r.CurrentTable.LookupDecl(expr.Literal.Literal); !exists {
		r.err(ddperror.SEM_NAME_UNDEFINED, expr.Token().Range, fmt.Sprintf("Der Name '%s' wurde noch nicht als Variable deklariert", expr.Literal.Literal), expr.Literal.File)
	} else if !isVar {
		r.err(ddperror.SEM_BAD_NAME_CONTEXT, expr.Token().Range, fmt.Sprintf("Der Name '%s' steht für eine Funktion und nicht für eine Variable", expr.Literal.Literal), expr.Literal.File)
	}
}
func (r *Resolver) VisitIndexing(expr *ast.Indexing) {
	r.visit(expr.Lhs)
	r.visit(expr.Index)
}

// nothing to do for literals
func (r *Resolver) VisitIntLit(expr *ast.IntLit) {
}
func (r *Resolver) VisitFloatLit(expr *ast.FloatLit) {
}
func (r *Resolver) VisitBoolLit(expr *ast.BoolLit) {
}
func (r *Resolver) VisitCharLit(expr *ast.CharLit) {
}
func (r *Resolver) VisitStringLit(expr *ast.StringLit) {
}
func (r *Resolver) VisitListLit(expr *ast.ListLit) {
	if expr.Values != nil {
		for _, v := range expr.Values {
			r.visit(v)
		}
	} else if expr.Count != nil && expr.Value != nil {
		r.visit(expr.Count)
		r.visit(expr.Value)
	}
}
func (r *Resolver) VisitUnaryExpr(expr *ast.UnaryExpr) {
	r.visit(expr.Rhs)
}
func (r *Resolver) VisitBinaryExpr(expr *ast.BinaryExpr) {
	r.visit(expr.Lhs)
	r.visit(expr.Rhs)
}
func (r *Resolver) VisitTernaryExpr(expr *ast.TernaryExpr) {
	r.visit(expr.Lhs)
	r.visit(expr.Mid)
	r.visit(expr.Rhs) // visit the actual expressions
}
func (r *Resolver) VisitCastExpr(expr *ast.CastExpr) {
	r.visit(expr.Lhs) // visit the actual expressions
}
func (r *Resolver) VisitGrouping(expr *ast.Grouping) {
	r.visit(expr.Expr)
}
func (r *Resolver) VisitFuncCall(expr *ast.FuncCall) {
	// visit the passed arguments
	for _, v := range expr.Args {
		r.visit(v)
	}
}

// if a BadStmt exists the AST is faulty
func (r *Resolver) VisitBadStmt(stmt *ast.BadStmt) {
	r.Errored = true
}
func (r *Resolver) VisitDeclStmt(stmt *ast.DeclStmt) {
	r.visit(stmt.Decl)
}
func (r *Resolver) VisitExprStmt(stmt *ast.ExprStmt) {
	r.visit(stmt.Expr)
}
func (r *Resolver) VisitAssignStmt(stmt *ast.AssignStmt) {
	switch assign := stmt.Var.(type) {
	case *ast.Ident:
		// check if the variable exists
		if _, exists, isVar := r.CurrentTable.LookupDecl(assign.Literal.Literal); !exists {
			r.err(ddperror.SEM_NAME_UNDEFINED, assign.Literal.Range, fmt.Sprintf("Der Name '%s' wurde in noch nicht als Variable deklariert", assign.Literal.Literal), assign.Literal.File)
		} else if !isVar {
			r.err(ddperror.SEM_BAD_NAME_CONTEXT, assign.Token().Range, fmt.Sprintf("Der Name '%s' steht für eine Funktion und nicht für eine Variable", assign.Literal.Literal), assign.Literal.File)
		}
	case *ast.Indexing:
		r.visit(assign.Lhs)
		r.visit(assign.Index)
	}
	r.visit(stmt.Rhs)
}
func (r *Resolver) VisitBlockStmt(stmt *ast.BlockStmt) {
	if stmt.Symbols == nil {
		r.setScope(stmt.Symbols) // set the current scope to the block
		// visit every statement in the block
		for _, stmt := range stmt.Statements {
			r.visit(stmt)
		}
		r.exitScope() // restore the enclosing scope
	}
}
func (r *Resolver) VisitIfStmt(stmt *ast.IfStmt) {
	r.visit(stmt.Condition)
	if _, ok := stmt.Then.(*ast.BlockStmt); !ok {
		r.visit(stmt.Then)
	}
	if stmt.Else != nil {
		if _, ok := stmt.Else.(*ast.BlockStmt); !ok {
			r.visit(stmt.Else)
		}
	}
}
func (r *Resolver) VisitWhileStmt(stmt *ast.WhileStmt) {
	r.visit(stmt.Condition)
	// r.visit(stmt.Body)
}
func (r *Resolver) VisitForStmt(stmt *ast.ForStmt) {
	r.setScope(stmt.Body.Symbols)
	// r.visit(stmt.Initializer)
	r.visit(stmt.To)
	if stmt.StepSize != nil {
		r.visit(stmt.StepSize)
	}
	// r.visit(stmt.Body)
	r.exitScope()
}
func (r *Resolver) VisitForRangeStmt(stmt *ast.ForRangeStmt) {
	r.setScope(stmt.Body.Symbols)
	// r.visit(stmt.Initializer) // also visits stmt.In
	r.visit(stmt.In)
	// r.visit(stmt.Body)
	r.exitScope()
}
func (r *Resolver) VisitReturnStmt(stmt *ast.ReturnStmt) {
	if stmt.Value == nil {
		return
	}
	r.visit(stmt.Value)
}
