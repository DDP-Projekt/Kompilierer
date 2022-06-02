package typechecker

import (
	"fmt"

	"github.com/DDP-Projekt/Kompilierer/pkg/ast"
	"github.com/DDP-Projekt/Kompilierer/pkg/scanner"
	"github.com/DDP-Projekt/Kompilierer/pkg/token"
)

// holds state to check if the types of an AST are valid
type Typechecker struct {
	errorHandler       scanner.ErrorHandler                  // function to which errors are passed
	CurrentTable       *ast.SymbolTable                      // SymbolTable of the current scope (needed for name type-checking)
	errored            bool                                  // wether the typechecker found an error
	latestReturnedType token.TokenType                       // type of the last visited expression
	funcArgs           map[string]map[string]token.TokenType // for function parameter types
}

func New(symbols *ast.SymbolTable, errorHandler scanner.ErrorHandler) *Typechecker {
	return &Typechecker{
		errorHandler:       errorHandler,
		CurrentTable:       symbols,
		errored:            false,
		latestReturnedType: token.NICHTS,
		funcArgs:           make(map[string]map[string]token.TokenType),
	}
}

// checks that all ast nodes fulfill type requirements
func TypecheckAst(Ast *ast.Ast, errorHandler scanner.ErrorHandler) {
	t := New(Ast.Symbols, errorHandler)

	for i, l := 0, len(Ast.Statements); i < l; i++ {
		Ast.Statements[i].Accept(t)
	}

	if t.errored {
		Ast.Faulty = true
	}
}

// typecheck a single node
func (t *Typechecker) TypecheckNode(node ast.Node) *Typechecker {
	return node.Accept(t).(*Typechecker)
}

// getter
func (t *Typechecker) Errored() bool {
	return t.errored
}

// helper to visit a node
func (t *Typechecker) visit(node ast.Node) {
	t = node.Accept(t).(*Typechecker)
}

// helper that returns the evaluated type of the node
func (t *Typechecker) evaluate(node ast.Node) token.TokenType {
	t.visit(node)
	return t.latestReturnedType
}

// helper for errors
func (t *Typechecker) err(tok token.Token, msg string) {
	t.errored = true
	t.errorHandler(fmt.Sprintf("Fehler in %s in Zeile %d, Spalte %d: %s", tok.File, tok.Line, tok.Column, msg))
}

// helper for commmon error message
func (t *Typechecker) errMismatch(tok token.Token, typ1, typ2 token.TokenType) {
	t.err(tok, fmt.Sprintf("Der Typ '%s' stimmt nicht mit dem Typ '%s' überein", typ1.String(), typ2.String()))
}

// helper for commmon error message
func (t *Typechecker) errExpected(tok token.Token, got token.TokenType, expected ...token.TokenType) {
	msg := "Es wurde ein Ausdruck vom Typ "
	if len(expected) == 1 {
		msg = "Es wurde ein Ausdruck vom Typ " + expected[0].String() + " erwartet aber '" + got.String() + "' gefunden"
	} else {
		for i, v := range expected {
			if i >= len(expected)-1 {
				break
			}
			msg += "'" + v.String() + "', "
		}
		msg += "oder '" + expected[len(expected)-1].String() + "' erwartet aber '" + got.String() + "' gefunden"
	}
	t.err(tok, msg)
}

// helper for commmon error message
func (t *Typechecker) errExpectedBin(tok token.Token, t1, t2, op token.TokenType) {
	t.err(tok, fmt.Sprintf("Die Typen Kombination aus '%s' und '%s' passt nicht zu dem '%s' Operator", t1, t2, op))
}

func (t *Typechecker) VisitBadDecl(d *ast.BadDecl) ast.Visitor {
	t.errored = true
	t.latestReturnedType = token.NICHTS
	return t
}
func (t *Typechecker) VisitVarDecl(d *ast.VarDecl) ast.Visitor {
	ty := t.evaluate(d.InitVal)
	if ty != d.Type.Type {
		t.errMismatch(d.InitVal.Token(), d.Type.Type, ty)
	}
	return t
}
func (t *Typechecker) VisitFuncDecl(d *ast.FuncDecl) ast.Visitor {
	t.funcArgs[d.Name.Literal] = make(map[string]token.TokenType)
	for i, l := 0, len(d.ParamNames); i < l; i++ {
		t.funcArgs[d.Name.Literal][d.ParamNames[i].Literal] = d.ParamTypes[i].Type
	}
	return d.Body.Accept(t)
}

func (t *Typechecker) VisitBadExpr(e *ast.BadExpr) ast.Visitor {
	t.errored = true
	t.latestReturnedType = token.NICHTS
	return t
}
func (t *Typechecker) VisitIdent(e *ast.Ident) ast.Visitor {
	t.latestReturnedType, _ = t.CurrentTable.LookupVar(e.Literal.Literal)
	return t
}
func (t *Typechecker) VisitIntLit(e *ast.IntLit) ast.Visitor {
	t.latestReturnedType = token.ZAHL
	return t
}
func (t *Typechecker) VisitFLoatLit(e *ast.FloatLit) ast.Visitor {
	t.latestReturnedType = token.KOMMAZAHL
	return t
}
func (t *Typechecker) VisitBoolLit(e *ast.BoolLit) ast.Visitor {
	t.latestReturnedType = token.BOOLEAN
	return t
}
func (t *Typechecker) VisitCharLit(e *ast.CharLit) ast.Visitor {
	t.latestReturnedType = token.BUCHSTABE
	return t
}
func (t *Typechecker) VisitStringLit(e *ast.StringLit) ast.Visitor {
	t.latestReturnedType = token.TEXT
	return t
}
func (t *Typechecker) VisitUnaryExpr(e *ast.UnaryExpr) ast.Visitor {
	// evaluate the rhs expression and check if the operator fits it
	rhs := t.evaluate(e.Rhs)
	switch e.Operator.Type {
	case token.BETRAG, token.NEGATE:
		if !isType(rhs, token.ZAHL, token.KOMMAZAHL) {
			t.errExpected(e.Operator, rhs, token.ZAHL, token.KOMMAZAHL)
		}
	case token.NICHT:
		if !isType(rhs, token.BOOLEAN) {
			t.errExpected(e.Operator, rhs, token.BOOLEAN)
		}
	case token.LÄNGE:
		if !isType(rhs, token.TEXT) {
			t.errExpected(e.Operator, rhs, token.TEXT)
		}
		t.latestReturnedType = token.ZAHL // some operators change the type of the rhs expression, so we set that
	case token.GRÖßE:
		t.latestReturnedType = token.ZAHL
	case token.ZAHL:
		t.latestReturnedType = token.ZAHL
	case token.KOMMAZAHL:
		if !isType(rhs, token.TEXT, token.ZAHL, token.KOMMAZAHL) {
			t.errExpected(e.Operator, rhs, token.ZAHL, token.TEXT, token.KOMMAZAHL)
		}
		t.latestReturnedType = token.KOMMAZAHL
	case token.BOOLEAN:
		if !isType(rhs, token.ZAHL, token.BOOLEAN) {
			t.errExpected(e.Operator, rhs, token.ZAHL, token.BOOLEAN)
		}
		t.latestReturnedType = token.BOOLEAN
	case token.BUCHSTABE:
		if !isType(rhs, token.ZAHL, token.BUCHSTABE) {
			t.errExpected(e.Operator, rhs, token.ZAHL, token.BUCHSTABE)
		}
	case token.TEXT:
		if !isType(rhs, token.ZAHL, token.KOMMAZAHL, token.BUCHSTABE, token.TEXT) {
			t.errExpected(e.Operator, rhs, token.ZAHL, token.KOMMAZAHL, token.BUCHSTABE, token.TEXT)
		}
		t.latestReturnedType = token.TEXT
	}
	return t
}
func (t *Typechecker) VisitBinaryExpr(e *ast.BinaryExpr) ast.Visitor {
	lhs := t.evaluate(e.Lhs)
	rhs := t.evaluate(e.Rhs)

	// helper to validate if types match
	validate := func(op token.TokenType, valid ...token.TokenType) {
		if !isTypeBin(lhs, rhs, valid...) {
			t.errExpectedBin(e.Token(), lhs, rhs, op)
		}
	}

	switch op := e.Operator.Type; op {
	case token.VERKETTET:
		validate(op, token.TEXT, token.BUCHSTABE)
		t.latestReturnedType = token.TEXT // some operators change the type of the expression, so we set that here
	case token.PLUS:
		validate(op, token.ZAHL, token.KOMMAZAHL)
		if lhs == token.ZAHL && rhs == token.ZAHL {
			t.latestReturnedType = token.ZAHL
		} else {
			t.latestReturnedType = token.KOMMAZAHL
		}
	case token.MINUS:
		validate(op, token.ZAHL, token.KOMMAZAHL)
		if lhs == token.ZAHL && rhs == token.ZAHL {
			t.latestReturnedType = token.ZAHL
		} else {
			t.latestReturnedType = token.KOMMAZAHL
		}
	case token.MAL:
		validate(op, token.ZAHL, token.KOMMAZAHL)
		if lhs == token.ZAHL && rhs == token.ZAHL {
			t.latestReturnedType = token.ZAHL
		} else {
			t.latestReturnedType = token.KOMMAZAHL
		}
	case token.DURCH:
		validate(op, token.ZAHL, token.KOMMAZAHL)
		t.latestReturnedType = token.KOMMAZAHL
	case token.MODULO:
		validate(op, token.ZAHL)
	case token.HOCH:
		validate(op, token.ZAHL, token.KOMMAZAHL)
		if lhs == token.ZAHL && rhs == token.ZAHL {
			t.latestReturnedType = token.ZAHL
		} else {
			t.latestReturnedType = token.KOMMAZAHL
		}
	case token.UND:
		validate(op, token.BOOLEAN)
	case token.ODER:
		validate(op, token.BOOLEAN)
	case token.LINKS:
		validate(op, token.ZAHL)
	case token.RECHTS:
		validate(op, token.ZAHL)
	case token.GLEICH:
		if lhs != rhs {
			t.errExpectedBin(e.Token(), lhs, rhs, op)
		}
		t.latestReturnedType = token.BOOLEAN
	case token.UNGLEICH:
		if lhs != rhs {
			t.errExpectedBin(e.Token(), lhs, rhs, op)
		}
		t.latestReturnedType = token.BOOLEAN
	case token.KLEINER:
		fallthrough
	case token.KLEINERODER:
		fallthrough
	case token.GRÖßER:
		fallthrough
	case token.GRÖßERODER:
		validate(op, token.ZAHL, token.KOMMAZAHL)
		t.latestReturnedType = token.BOOLEAN
	}
	return t
}
func (t *Typechecker) VisitGrouping(e *ast.Grouping) ast.Visitor {
	return e.Expr.Accept(t)
}
func (t *Typechecker) VisitFuncCall(e *ast.FuncCall) ast.Visitor {
	for k, v := range e.Args {
		ty := t.evaluate(v)
		if ty != t.funcArgs[e.Name][k] {
			t.errExpected(v.Token(), ty, t.funcArgs[e.Name][k])
		}
	}
	fun, _ := t.CurrentTable.LookupFunc(e.Name)
	t.latestReturnedType = fun.Type.Type
	return t
}

func (t *Typechecker) VisitBadStmt(s *ast.BadStmt) ast.Visitor {
	t.errored = true
	t.latestReturnedType = token.NICHTS
	return t
}
func (t *Typechecker) VisitDeclStmt(s *ast.DeclStmt) ast.Visitor {
	return s.Decl.Accept(t)
}
func (t *Typechecker) VisitExprStmt(s *ast.ExprStmt) ast.Visitor {
	return s.Expr.Accept(t)
}
func (t *Typechecker) VisitAssignStmt(s *ast.AssignStmt) ast.Visitor {
	ty := t.evaluate(s.Rhs)
	if vartyp, exists := t.CurrentTable.LookupVar(s.Name.Literal); exists && vartyp != ty {
		t.errMismatch(s.Token(), vartyp, ty)
	}
	return t
}
func (t *Typechecker) VisitBlockStmt(s *ast.BlockStmt) ast.Visitor {
	t.CurrentTable = s.Symbols
	for _, stmt := range s.Statements {
		t.visit(stmt)
	}
	t.CurrentTable = s.Symbols.Enclosing
	return t
}
func (t *Typechecker) VisitIfStmt(s *ast.IfStmt) ast.Visitor {
	condty := t.evaluate(s.Condition)
	if condty != token.BOOLEAN {
		t.errMismatch(s.Condition.Token(), token.BOOLEAN, condty)
	}
	t.visit(s.Then)
	if s.Else != nil {
		t.visit(s.Else)
	}
	return t
}
func (t *Typechecker) VisitWhileStmt(s *ast.WhileStmt) ast.Visitor {
	condty := t.evaluate(s.Condition)
	if condty != token.BOOLEAN {
		t.errMismatch(s.Condition.Token(), token.BOOLEAN, condty)
	}
	return s.Body.Accept(t)
}
func (t *Typechecker) VisitForStmt(s *ast.ForStmt) ast.Visitor {
	t.visit(s.Initializer)
	toty := t.evaluate(s.To)
	if toty != token.ZAHL {
		t.errMismatch(s.To.Token(), token.ZAHL, toty)
	}
	if s.StepSize != nil {
		stepty := t.evaluate(s.StepSize)
		if stepty != token.ZAHL {
			t.errMismatch(s.StepSize.Token(), token.ZAHL, stepty)
		}
	}
	return s.Body.Accept(t)
}
func (t *Typechecker) VisitFuncCallStmt(s *ast.FuncCallStmt) ast.Visitor {
	return s.Call.Accept(t)
}
func (t *Typechecker) VisitReturnStmt(s *ast.ReturnStmt) ast.Visitor {
	ty := t.evaluate(s.Value)
	if fun, exists := t.CurrentTable.LookupFunc(s.Func); exists && fun.Type.Type != ty {
		t.errMismatch(s.Token(), fun.Type.Type, ty)
	}
	return t
}

// checks if t is contained in types
func isType(t token.TokenType, types ...token.TokenType) bool {
	for _, v := range types {
		if t == v {
			return true
		}
	}
	return false
}

// checks if t1 and t2 are both contained in types
func isTypeBin(t1, t2 token.TokenType, types ...token.TokenType) bool {
	return isType(t1, types...) && isType(t2, types...)
}
