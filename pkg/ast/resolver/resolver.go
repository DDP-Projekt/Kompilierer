package resolver

import (
	"DDP/pkg/ast"
	"DDP/pkg/scanner"
	"DDP/pkg/token"
	"fmt"
)

// TODO: more testing

type Resolver struct {
	errorHandler scanner.ErrorHandler
	CurrentTable *ast.SymbolTable
	errored      bool
}

func NewResolver(ast *ast.Ast, errorHandler scanner.ErrorHandler) *Resolver {
	return &Resolver{
		errorHandler: errorHandler,
		CurrentTable: ast.Symbols,
		errored:      false,
	}
}

// fills out the asts SymbolTables and reports any errors on the way
func ResolveAst(Ast *ast.Ast, errorHandler scanner.ErrorHandler) {
	Ast.Symbols = ast.NewSymbolTable(nil)

	r := NewResolver(Ast, errorHandler)

	for i, l := 0, len(Ast.Statements); i < l; i++ {
		Ast.Statements[i].Accept(r)
	}

	if r.errored {
		Ast.Faulty = true
	}
}

func (r *Resolver) ResolveNode(node ast.Node) *Resolver {
	return node.Accept(r).(*Resolver)
}

func (r *Resolver) Errored() bool {
	return r.errored
}

func (r *Resolver) visit(node ast.Node) {
	r = node.Accept(r).(*Resolver)
}

// helper for errors
func (r *Resolver) err(t token.Token, msg string) {
	r.errored = true
	r.errorHandler(fmt.Sprintf("Fehler in %s in Zeile %d, Spalte %d: %s", t.File, t.Line, t.Column, msg))
}

func (r *Resolver) VisitBadDecl(d *ast.BadDecl) ast.Visitor {
	r.errored = true
	return r
}
func (r *Resolver) VisitVarDecl(d *ast.VarDecl) ast.Visitor {
	d.InitVal.Accept(r)
	if existed := r.CurrentTable.InsertVar(d.Name.Literal, d.Type.Type); existed {
		r.err(d.Token(), fmt.Sprintf("Die Variable '%s' existiert bereits", d.Name.Literal))
	}
	return r
}
func (r *Resolver) VisitFuncDecl(d *ast.FuncDecl) ast.Visitor {
	if existed := r.CurrentTable.InsertFunc(d.Name.Literal, d); existed {
		r.err(d.Token(), fmt.Sprintf("Die Funktion '%s' existiert bereits", d.Name.Literal))
	}
	body := d.Body.(*ast.BlockStmt)
	body.Symbols = ast.NewSymbolTable(r.CurrentTable)
	for i, l := 0, len(d.ParamNames); i < l; i++ {
		body.Symbols.InsertVar(d.ParamNames[i].Literal, d.ParamTypes[i].Type)
	}
	return d.Body.Accept(r)
}

func (r *Resolver) VisitBadExpr(e *ast.BadExpr) ast.Visitor {
	r.errored = true
	return r
}
func (r *Resolver) VisitIdent(e *ast.Ident) ast.Visitor {
	if _, exists := r.CurrentTable.LookupVar(e.Literal.Literal); !exists {
		r.err(e.Token(), fmt.Sprintf("Der Name '%s' wurde noch nicht als Variable deklariert", e.Literal.Literal))
	}
	return r
}
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
	return e.Rhs.Accept(r)
}
func (r *Resolver) VisitBinaryExpr(e *ast.BinaryExpr) ast.Visitor {
	return e.Rhs.Accept(e.Lhs.Accept(r))
}
func (r *Resolver) VisitGrouping(e *ast.Grouping) ast.Visitor {
	return e.Expr.Accept(r)
}
func (r *Resolver) VisitFuncCall(e *ast.FuncCall) ast.Visitor {
	for _, v := range e.Args {
		r.visit(v)
	}
	return r
}

func (r *Resolver) VisitBadStmt(s *ast.BadStmt) ast.Visitor {
	r.errored = true
	return r
}
func (r *Resolver) VisitDeclStmt(s *ast.DeclStmt) ast.Visitor {
	return s.Decl.Accept(r)
}
func (r *Resolver) VisitExprStmt(s *ast.ExprStmt) ast.Visitor {
	return s.Expr.Accept(r)
}
func (r *Resolver) VisitAssignStmt(s *ast.AssignStmt) ast.Visitor {
	if _, exists := r.CurrentTable.LookupVar(s.Name.Literal); !exists {
		r.err(s.Token(), fmt.Sprintf("Der Name '%s' wurde in noch nicht als Variable deklariert", s.Name.Literal))
	}
	return s.Rhs.Accept(r)
}
func (r *Resolver) VisitBlockStmt(s *ast.BlockStmt) ast.Visitor {
	if s.Symbols == nil {
		s.Symbols = ast.NewSymbolTable(r.CurrentTable)
	}
	r.CurrentTable = s.Symbols
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
	var env *ast.SymbolTable
	if body, ok := s.Body.(*ast.BlockStmt); ok {
		body.Symbols = ast.NewSymbolTable(r.CurrentTable)
		env = body.Symbols
	} else {
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
func (r *Resolver) VisitFuncCallStmt(s *ast.FuncCallStmt) ast.Visitor {
	return s.Call.Accept(r)
}
func (r *Resolver) VisitReturnStmt(s *ast.ReturnStmt) ast.Visitor {
	if _, exists := r.CurrentTable.LookupFunc(s.Func); !exists {
		r.err(s.Token(), "Man kann nur aus Funktionen einen Wert zurückgeben")
	}
	return s.Value.Accept(r)
}
