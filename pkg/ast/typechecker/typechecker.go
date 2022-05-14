package typechecker

import (
	"DDP/pkg/ast"
	"DDP/pkg/scanner"
	"DDP/pkg/token"
	"fmt"
)

type Typechecker struct {
	errorHandler       scanner.ErrorHandler
	CurrentTable       *ast.SymbolTable // SymbolTable of the current scope (needed for name type-checking)
	errored            bool
	latestReturnedType token.TokenType                       // type of the last visited expression
	funcArgs           map[string]map[string]token.TokenType // for function parameter types
}

func NewTypechecker(symbols *ast.SymbolTable, errorHandler scanner.ErrorHandler) *Typechecker {
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
	t := NewTypechecker(Ast.Symbols, errorHandler)

	for i, l := 0, len(Ast.Statements); i < l; i++ {
		Ast.Statements[i].Accept(t)
	}

	if t.errored {
		Ast.Faulty = true
	}
}

func (t *Typechecker) TypecheckNode(node ast.Node) *Typechecker {
	return node.Accept(t).(*Typechecker)
}

func (t *Typechecker) Errored() bool {
	return t.errored
}

func (t *Typechecker) visit(node ast.Node) {
	t = node.Accept(t).(*Typechecker)
}

// helper for errors
func (t *Typechecker) err(tok token.Token, msg string) {
	t.errored = true
	t.errorHandler(fmt.Sprintf("Fehler in %s in Zeile %d, Spalte %d: %s", tok.File, tok.Line, tok.Column, msg))
}

func (t *Typechecker) errMismatch(tok token.Token, typ1, typ2 token.TokenType) {
	t.err(tok, fmt.Sprintf("Der Typ '%s' stimmt nicht mit dem Typ '%s' überein", typ1.String(), typ2.String()))
}

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

func (t *Typechecker) errExpectedBin(tok token.Token, t1, t2, op token.TokenType) {
	t.err(tok, fmt.Sprintf("Die Typen Kombination aus '%s' und '%s' passt nicht zu dem '%s' Operator", t1, t2, op))
}

func (t *Typechecker) VisitBadDecl(d *ast.BadDecl) ast.Visitor {
	t.errored = true
	t.latestReturnedType = token.NICHTS
	return t
}
func (t *Typechecker) VisitVarDecl(d *ast.VarDecl) ast.Visitor {
	t.visit(d.InitVal)
	if t.latestReturnedType != d.Type.Type {
		t.errMismatch(d.InitVal.Token(), d.Type.Type, t.latestReturnedType)
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
	t.visit(e.Rhs)
	switch e.Operator.Type {
	case token.BETRAG, token.NEGATE:
		if !isType(t.latestReturnedType, token.ZAHL, token.KOMMAZAHL) {
			t.errExpected(e.Operator, t.latestReturnedType, token.ZAHL, token.KOMMAZAHL)
		}
	case token.NICHT:
		if !isType(t.latestReturnedType, token.BOOLEAN) {
			t.errExpected(e.Operator, t.latestReturnedType, token.BOOLEAN)
		}
		t.latestReturnedType = token.BOOLEAN
	case token.LÄNGE:
		if !isType(t.latestReturnedType, token.TEXT) {
			t.errExpected(e.Operator, t.latestReturnedType, token.TEXT)
		}
		t.latestReturnedType = token.ZAHL
	case token.GRÖßE:
		t.latestReturnedType = token.ZAHL
	case token.ZAHL:
		t.latestReturnedType = token.ZAHL
	case token.KOMMAZAHL:
		if !isType(t.latestReturnedType, token.TEXT, token.ZAHL, token.KOMMAZAHL) {
			t.errExpected(e.Operator, t.latestReturnedType, token.ZAHL, token.TEXT, token.KOMMAZAHL)
		}
		t.latestReturnedType = token.KOMMAZAHL
	case token.BOOLEAN:
		if !isType(t.latestReturnedType, token.ZAHL, token.BOOLEAN) {
			t.errExpected(e.Operator, t.latestReturnedType, token.ZAHL, token.BOOLEAN)
		}
		t.latestReturnedType = token.BOOLEAN
	case token.BUCHSTABE:
		if !isType(t.latestReturnedType, token.ZAHL, token.BUCHSTABE) {
			t.errExpected(e.Operator, t.latestReturnedType, token.ZAHL, token.BUCHSTABE)
		}
	case token.TEXT:
		if !isType(t.latestReturnedType, token.ZAHL, token.KOMMAZAHL, token.BUCHSTABE, token.TEXT) {
			t.errExpected(e.Operator, t.latestReturnedType, token.ZAHL, token.KOMMAZAHL, token.BUCHSTABE, token.TEXT)
		}
		t.latestReturnedType = token.TEXT
	}
	return t
}
func (t *Typechecker) VisitBinaryExpr(e *ast.BinaryExpr) ast.Visitor {
	t.visit(e.Lhs)
	lhs := t.latestReturnedType
	t.visit(e.Rhs)
	rhs := t.latestReturnedType

	validate := func(tok token.Token, op token.TokenType, valid ...token.TokenType) {
		if !isTypeBin(lhs, rhs, valid...) {
			t.errExpectedBin(e.Token(), lhs, rhs, op)
		}
	}

	switch op := e.Operator.Type; op {
	case token.VERKETTET:
		validate(e.Token(), op, token.TEXT, token.BUCHSTABE)
		t.latestReturnedType = token.TEXT
	case token.PLUS:
		validate(e.Token(), op, token.ZAHL, token.KOMMAZAHL)
		if lhs == token.ZAHL && rhs == token.ZAHL {
			t.latestReturnedType = token.ZAHL
		} else {
			t.latestReturnedType = token.KOMMAZAHL
		}
	case token.MINUS:
		validate(e.Token(), op, token.ZAHL, token.KOMMAZAHL)
		if lhs == token.ZAHL && rhs == token.ZAHL {
			t.latestReturnedType = token.ZAHL
		} else {
			t.latestReturnedType = token.KOMMAZAHL
		}
	case token.MAL:
		validate(e.Token(), op, token.ZAHL, token.KOMMAZAHL)
		if lhs == token.ZAHL && rhs == token.ZAHL {
			t.latestReturnedType = token.ZAHL
		} else {
			t.latestReturnedType = token.KOMMAZAHL
		}
	case token.DURCH:
		validate(e.Token(), op, token.ZAHL, token.KOMMAZAHL)
		t.latestReturnedType = token.KOMMAZAHL
	case token.MODULO:
		validate(e.Token(), op, token.ZAHL)
	case token.HOCH:
		validate(e.Token(), op, token.ZAHL, token.KOMMAZAHL)
		if lhs == token.ZAHL && rhs == token.ZAHL {
			t.latestReturnedType = token.ZAHL
		} else {
			t.latestReturnedType = token.KOMMAZAHL
		}
	case token.UND:
		validate(e.Token(), op, token.BOOLEAN)
	case token.ODER:
		validate(e.Token(), op, token.BOOLEAN)
	case token.LINKS:
		validate(e.Token(), op, token.ZAHL)
	case token.RECHTS:
		validate(e.Token(), op, token.ZAHL)
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
		validate(e.Token(), op, token.ZAHL, token.KOMMAZAHL)
		t.latestReturnedType = token.BOOLEAN
	}
	return t
}
func (t *Typechecker) VisitGrouping(e *ast.Grouping) ast.Visitor {
	return e.Expr.Accept(t)
}
func (t *Typechecker) VisitFuncCall(e *ast.FuncCall) ast.Visitor {
	for k, v := range e.Args {
		t.visit(v)
		if t.latestReturnedType != t.funcArgs[e.Name][k] {
			t.errExpected(v.Token(), t.latestReturnedType, t.funcArgs[e.Name][k])
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
	t.visit(s.Rhs)
	if typ, exists := t.CurrentTable.LookupVar(s.Name.Literal); exists && typ != t.latestReturnedType {
		t.errMismatch(s.Token(), typ, t.latestReturnedType)
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
	t.visit(s.Condition)
	if t.latestReturnedType != token.BOOLEAN {
		t.errMismatch(s.Condition.Token(), token.BOOLEAN, t.latestReturnedType)
	}
	t.visit(s.Then)
	if s.Else != nil {
		t.visit(s.Else)
	}
	return t
}
func (t *Typechecker) VisitWhileStmt(s *ast.WhileStmt) ast.Visitor {
	t.visit(s.Condition)
	if t.latestReturnedType != token.BOOLEAN {
		t.errMismatch(s.Condition.Token(), token.BOOLEAN, t.latestReturnedType)
	}
	return s.Body.Accept(t)
}
func (t *Typechecker) VisitForStmt(s *ast.ForStmt) ast.Visitor {
	t.visit(s.Initializer)
	t.visit(s.To)
	if t.latestReturnedType != token.ZAHL {
		t.errMismatch(s.To.Token(), token.ZAHL, t.latestReturnedType)
	}
	if s.StepSize != nil {
		t.visit(s.StepSize)
		if t.latestReturnedType != token.ZAHL {
			t.errMismatch(s.StepSize.Token(), token.ZAHL, t.latestReturnedType)
		}
	}
	return s.Body.Accept(t)
}
func (t *Typechecker) VisitFuncCallStmt(s *ast.FuncCallStmt) ast.Visitor {
	return s.Call.Accept(t)
}
func (t *Typechecker) VisitReturnStmt(s *ast.ReturnStmt) ast.Visitor {
	t.visit(s.Value)
	if fun, exists := t.CurrentTable.LookupFunc(s.Func); exists && fun.Type.Type != t.latestReturnedType {
		t.errMismatch(s.Token(), fun.Type.Type, t.latestReturnedType)
	}
	return t
}

func isType(t token.TokenType, types ...token.TokenType) bool {
	for _, v := range types {
		if t == v {
			return true
		}
	}
	return false
}

func isTypeBin(t1, t2 token.TokenType, types ...token.TokenType) bool {
	return isType(t1, types...) && isType(t2, types...)
}
