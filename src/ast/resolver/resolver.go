/*
The resolver package should not be used independently from the parser.
It is not a complete Visitor itself, but is rather used to resolve single
Nodes while parsing to ensure correct parsing of function-calls etc.

Because of this you will find many r.visit(x) calls to be commented out.
*/
package resolver

import (
	"fmt"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/token"
)

// holds state to resolve the symbols of an AST and its nodes
// and checking if they are valid
// fills the ASTs SymbolTable while doing so
type Resolver struct {
	ErrorHandler ddperror.Handler // function to which errors are passed
	CurrentTable *ast.SymbolTable // needed state, public for the parser
	Module       *ast.Module      // the module that is being resolved
}

// create a new resolver to resolve the passed AST
func New(Mod *ast.Module, errorHandler ddperror.Handler, file string) *Resolver {
	if errorHandler == nil {
		errorHandler = ddperror.EmptyHandler
	}
	return &Resolver{
		ErrorHandler: errorHandler,
		CurrentTable: Mod.Ast.Symbols,
		Module:       Mod,
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
func (r *Resolver) err(code ddperror.Code, Range token.Range, msg string) {
	r.Module.Ast.Faulty = true
	r.ErrorHandler(ddperror.New(code, Range, msg, r.Module.FileName))
}

func (*Resolver) BaseVisitor() {}

// if a BadDecl exists the AST is faulty
func (r *Resolver) VisitBadDecl(decl *ast.BadDecl) {
	r.Module.Ast.Faulty = true
}
func (r *Resolver) VisitVarDecl(decl *ast.VarDecl) {
	r.visit(decl.InitVal) // resolve the initial value
	// insert the variable into the current scope (SymbolTable)
	if existed := r.CurrentTable.InsertDecl(decl.Name(), decl); existed {
		r.err(ddperror.SEM_NAME_ALREADY_DEFINED, decl.NameTok.Range, ddperror.MsgNameAlreadyExists(decl.Name())) // variables may only be declared once in the same scope
	}

	if decl.Public() && r.CurrentTable.Enclosing != nil {
		r.err(ddperror.SEM_NON_GLOBAL_PUBLIC_DECL, decl.NameTok.Range, "Nur globale Variablen können öffentlich sein")
	} else if _, alreadyExists := r.Module.PublicDecls[decl.Name()]; decl.IsPublic && !alreadyExists {
		r.Module.PublicDecls[decl.Name()] = decl
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
	r.Module.Ast.Faulty = true
}
func (r *Resolver) VisitIdent(expr *ast.Ident) {
	// check if the variable exists
	if decl, exists, isVar := r.CurrentTable.LookupDecl(expr.Literal.Literal); !exists {
		r.err(ddperror.SEM_NAME_UNDEFINED, expr.Token().Range, fmt.Sprintf("Der Name '%s' wurde noch nicht als Variable deklariert", expr.Literal.Literal))
	} else if !isVar {
		r.err(ddperror.SEM_BAD_NAME_CONTEXT, expr.Token().Range, fmt.Sprintf("Der Name '%s' steht für eine Funktion und nicht für eine Variable", expr.Literal.Literal))
	} else { // set the reference to the declaration
		expr.Declaration = decl.(*ast.VarDecl)
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
	r.Module.Ast.Faulty = true
}
func (r *Resolver) VisitDeclStmt(stmt *ast.DeclStmt) {
	r.visit(stmt.Decl)
}
func (r *Resolver) VisitExprStmt(stmt *ast.ExprStmt) {
	r.visit(stmt.Expr)
}
func (r *Resolver) VisitImportStmt(stmt *ast.ImportStmt) {
	if stmt.Module == nil {
		return // TODO: handle this better
	}

	// add imported symbols
	ast.IterateImportedDecls(stmt, func(name string, decl ast.Declaration, tok token.Token) bool {
		if decl == nil {
			r.err(ddperror.SEM_NAME_UNDEFINED, tok.Range, fmt.Sprintf("Der Name '%s' entspricht keiner öffentlichen Deklaration aus dem Modul '%s'", name, ast.TrimStringLit(stmt.FileName)))
		} else {
			if r.CurrentTable.InsertDecl(name, decl) {
				r.err(ddperror.SEM_NAME_ALREADY_DEFINED, tok.Range, fmt.Sprintf("Der Name '%s' aus dem Modul '%s' existiert bereits in diesem Modul", name, ast.TrimStringLit(stmt.FileName)))
			}
		}
		return true
	})
}
func (r *Resolver) VisitAssignStmt(stmt *ast.AssignStmt) {
	switch assign := stmt.Var.(type) {
	case *ast.Ident:
		// check if the variable exists
		if varDecl, exists, isVar := r.CurrentTable.LookupDecl(assign.Literal.Literal); !exists {
			r.err(ddperror.SEM_NAME_UNDEFINED, assign.Literal.Range, fmt.Sprintf("Der Name '%s' wurde in noch nicht als Variable deklariert", assign.Literal.Literal))
		} else if !isVar {
			r.err(ddperror.SEM_BAD_NAME_CONTEXT, assign.Token().Range, fmt.Sprintf("Der Name '%s' steht für eine Funktion und nicht für eine Variable", assign.Literal.Literal))
		} else { // set the reference to the declaration
			assign.Declaration = varDecl.(*ast.VarDecl)
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
	// only visit the InitVal because the variable is already in the scope
	r.visit(stmt.Initializer.InitVal)
	r.visit(stmt.To)
	if stmt.StepSize != nil {
		r.visit(stmt.StepSize)
	}
	// r.visit(stmt.Body) // created by calling checkedDeclration in the parser so already resolved
	r.exitScope()
}
func (r *Resolver) VisitForRangeStmt(stmt *ast.ForRangeStmt) {
	r.setScope(stmt.Body.Symbols)
	// only visit the InitVal because the variable is already in the scope
	r.visit(stmt.Initializer.InitVal)
	r.visit(stmt.In)
	// r.visit(stmt.Body) // created by calling checkedDeclration in the parser so already resolved
	r.exitScope()
}
func (r *Resolver) VisitReturnStmt(stmt *ast.ReturnStmt) {
	if stmt.Value == nil {
		return
	}
	r.visit(stmt.Value)
}
