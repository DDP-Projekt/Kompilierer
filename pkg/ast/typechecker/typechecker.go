package typechecker

import (
	"fmt"

	"github.com/DDP-Projekt/Kompilierer/pkg/ast"
	"github.com/DDP-Projekt/Kompilierer/pkg/scanner"
	"github.com/DDP-Projekt/Kompilierer/pkg/token"
)

// holds state to check if the types of an AST are valid
type Typechecker struct {
	ErrorHandler       scanner.ErrorHandler                  // function to which errors are passed
	CurrentTable       *ast.SymbolTable                      // SymbolTable of the current scope (needed for name type-checking)
	Errored            bool                                  // wether the typechecker found an error
	latestReturnedType token.TokenType                       // type of the last visited expression
	funcArgs           map[string]map[string]token.TokenType // for function parameter types
}

func New(symbols *ast.SymbolTable, errorHandler scanner.ErrorHandler) *Typechecker {
	return &Typechecker{
		ErrorHandler:       errorHandler,
		CurrentTable:       symbols,
		Errored:            false,
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

	if t.Errored {
		Ast.Faulty = true
	}
}

// typecheck a single node
func (t *Typechecker) TypecheckNode(node ast.Node) *Typechecker {
	return node.Accept(t).(*Typechecker)
}

// helper to visit a node
func (t *Typechecker) visit(node ast.Node) {
	t = node.Accept(t).(*Typechecker)
}

// Evaluates the type of an expression
func (t *Typechecker) Evaluate(expr ast.Expression) token.TokenType {
	t.visit(expr)
	return t.latestReturnedType
}

// helper for errors
func (t *Typechecker) err(tok token.Token, msg string) {
	t.Errored = true
	t.ErrorHandler(fmt.Sprintf("Fehler in %s in Zeile %d, Spalte %d: %s", tok.File, tok.Line, tok.Column, msg))
}

// helper for commmon error message
func (t *Typechecker) errExpected(tok token.Token, got token.TokenType, expected ...token.TokenType) {
	msg := "Der " + tok.String() + " Operator erwartet einen Ausdruck vom Typ "
	if len(expected) == 1 {
		msg = "Der " + tok.String() + " Operator erwartet einen Ausdruck vom Typ " + expected[0].String() + " aber hat '" + got.String() + "' bekommen"
	} else {
		for i, v := range expected {
			if i >= len(expected)-1 {
				break
			}
			msg += "'" + v.String() + "', "
		}
		msg += "oder '" + expected[len(expected)-1].String() + "' aber hat '" + got.String() + "' bekommen"
	}
	t.err(tok, msg)
}

// helper for commmon error message
func (t *Typechecker) errExpectedBin(tok token.Token, t1, t2, op token.TokenType) {
	t.err(tok, fmt.Sprintf("Die Typen Kombination aus '%s' und '%s' passt nicht zu dem '%s' Operator", t1, t2, op))
}

func (t *Typechecker) VisitBadDecl(d *ast.BadDecl) ast.Visitor {
	t.Errored = true
	t.latestReturnedType = token.NICHTS
	return t
}
func (t *Typechecker) VisitVarDecl(d *ast.VarDecl) ast.Visitor {
	ty := t.Evaluate(d.InitVal)
	if ty != d.Type.Type {
		t.err(d.InitVal.Token(), fmt.Sprintf(
			"Ein Wert vom Typ %s kann keiner Variable vom Typ %s zugewiesen werden",
			ty,
			d.Type.Type,
		))
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
	t.Errored = true
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
	// Evaluate the rhs expression and check if the operator fits it
	rhs := t.Evaluate(e.Rhs)
	// boolean vs bitwise negate (compound assignement)
	/*if e.Operator.Type == token.NEGIERE && rhs == token.ZAHL {
		e.Operator.Type == token.LOGISCHNICHT
	}*/
	switch e.Operator.Type {
	case token.BETRAG, token.NEGATE:
		if !isType(rhs, token.ZAHL, token.KOMMAZAHL) {
			t.errExpected(e.Operator, rhs, token.ZAHL, token.KOMMAZAHL)
		}
	case token.NICHT:
		if !isType(rhs, token.BOOLEAN) {
			t.errExpected(e.Operator, rhs, token.BOOLEAN)
		}
		t.latestReturnedType = token.BOOLEAN
	case token.NEGIERE:
		if !isType(rhs, token.BOOLEAN, token.ZAHL) {
			t.errExpected(e.Operator, rhs, token.BOOLEAN, token.ZAHL)
		}
	case token.LOGISCHNICHT:
		if !isType(rhs, token.ZAHL) {
			t.errExpected(e.Operator, rhs, token.ZAHL)
		}
		t.latestReturnedType = token.ZAHL
	case token.LÄNGE:
		if !isType(rhs, token.TEXT) {
			t.errExpected(e.Operator, rhs, token.TEXT)
		}
		t.latestReturnedType = token.ZAHL // some operators change the type of the rhs expression, so we set that
	case token.GRÖßE:
		t.latestReturnedType = token.ZAHL
	case token.SINUS, token.KOSINUS, token.TANGENS,
		token.ARKSIN, token.ARKKOS, token.ARKTAN,
		token.HYPSIN, token.HYPKOS, token.HYPTAN:
		if !isType(rhs, token.ZAHL, token.KOMMAZAHL) {
			t.errExpected(e.Operator, rhs, token.ZAHL, token.KOMMAZAHL)
		}
		t.latestReturnedType = token.KOMMAZAHL
	case token.ZAHL:
		if !isType(rhs, token.KOMMAZAHL, token.ZAHL, token.TEXT, token.BOOLEAN, token.BUCHSTABE) {
			t.errExpected(e.Operator, rhs, token.KOMMAZAHL, token.ZAHL, token.TEXT, token.BOOLEAN, token.BUCHSTABE)
		}
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
		t.latestReturnedType = token.BUCHSTABE
	case token.TEXT:
		if !isType(rhs, token.ZAHL, token.KOMMAZAHL, token.BOOLEAN, token.TEXT, token.BUCHSTABE) {
			t.errExpected(e.Operator, rhs, token.ZAHL, token.KOMMAZAHL, token.BOOLEAN, token.TEXT, token.BUCHSTABE)
		}
		t.latestReturnedType = token.TEXT
	}
	return t
}
func (t *Typechecker) VisitBinaryExpr(e *ast.BinaryExpr) ast.Visitor {
	lhs := t.Evaluate(e.Lhs)
	rhs := t.Evaluate(e.Rhs)

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
	case token.PLUS, token.ADDIERE, token.ERHÖHE,
		token.MINUS, token.SUBTRAHIERE, token.VERRINGERE,
		token.MAL, token.MULTIPLIZIERE, token.VERVIELFACHE:
		validate(op, token.ZAHL, token.KOMMAZAHL)
		if lhs == token.ZAHL && rhs == token.ZAHL {
			t.latestReturnedType = token.ZAHL
		} else {
			t.latestReturnedType = token.KOMMAZAHL
		}
	case token.STELLE:
		if lhs != token.TEXT {
			t.err(e.Lhs.Token(), "Der STELLE Operator erwartet einen Text als ersten Operanden")
		}
		if rhs != token.ZAHL {
			t.err(e.Lhs.Token(), "Der STELLE Operator erwartet eine Zahl als zweiten Operanden")
		}
		t.latestReturnedType = token.BUCHSTABE // later on the list element type
	case token.DURCH, token.DIVIDIERE, token.TEILE, token.HOCH, token.LOGARITHMUS:
		validate(op, token.ZAHL, token.KOMMAZAHL)
		t.latestReturnedType = token.KOMMAZAHL
	case token.MODULO:
		validate(op, token.ZAHL)
		t.latestReturnedType = token.ZAHL
	case token.UND:
		validate(op, token.BOOLEAN)
		t.latestReturnedType = token.BOOLEAN
	case token.ODER:
		validate(op, token.BOOLEAN)
		t.latestReturnedType = token.BOOLEAN
	case token.LINKS:
		validate(op, token.ZAHL)
		t.latestReturnedType = token.ZAHL
	case token.RECHTS:
		validate(op, token.ZAHL)
		t.latestReturnedType = token.ZAHL
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
	case token.GRÖßERODER, token.KLEINER, token.KLEINERODER, token.GRÖßER:
		validate(op, token.ZAHL, token.KOMMAZAHL)
		t.latestReturnedType = token.BOOLEAN
	case token.LOGISCHODER, token.LOGISCHUND, token.KONTRA:
		validate(op, token.ZAHL)
		t.latestReturnedType = token.ZAHL
	}
	return t
}
func (t *Typechecker) VisitGrouping(e *ast.Grouping) ast.Visitor {
	return e.Expr.Accept(t)
}
func (t *Typechecker) VisitFuncCall(e *ast.FuncCall) ast.Visitor {
	for k, v := range e.Args {
		ty := t.Evaluate(v)
		if ty != t.funcArgs[e.Name][k] {
			t.err(v.Token(), fmt.Sprintf(
				"Die Funktion %s erwartet einen Wert vom Typ %s für den Parameter %s, aber hat %s bekommen",
				e.Name,
				t.funcArgs[e.Name][k],
				k,
				ty,
			))
		}
	}
	fun, _ := t.CurrentTable.LookupFunc(e.Name)
	t.latestReturnedType = fun.Type.Type
	return t
}

func (t *Typechecker) VisitBadStmt(s *ast.BadStmt) ast.Visitor {
	t.Errored = true
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
	ty := t.Evaluate(s.Rhs)
	if vartyp, exists := t.CurrentTable.LookupVar(s.Name.Literal); exists && vartyp != ty {
		t.err(s.Rhs.Token(), fmt.Sprintf(
			"Ein Wert vom Typ %s kann keiner Variable vom Typ %s zugewiesen werden",
			ty,
			vartyp,
		))
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
	condty := t.Evaluate(s.Condition)
	if condty != token.BOOLEAN {
		t.err(s.Condition.Token(), fmt.Sprintf(
			"Die Bedingung einer WENN Anweisung muss vom Typ Boolean sein, war aber vom Typ %s",
			condty,
		))
	}
	t.visit(s.Then)
	if s.Else != nil {
		t.visit(s.Else)
	}
	return t
}
func (t *Typechecker) VisitWhileStmt(s *ast.WhileStmt) ast.Visitor {
	condty := t.Evaluate(s.Condition)
	if condty != token.BOOLEAN {
		t.err(s.Condition.Token(), fmt.Sprintf(
			"Die Bedingung einer SOLANGE Anweisung muss vom Typ BOOLEAN sein, war aber vom Typ %s",
			condty,
		))
	}
	return s.Body.Accept(t)
}
func (t *Typechecker) VisitForStmt(s *ast.ForStmt) ast.Visitor {
	t.visit(s.Initializer)
	toty := t.Evaluate(s.To)
	if toty != token.ZAHL {
		t.err(s.To.Token(), fmt.Sprintf(
			"Es wurde ein Ausdruck vom Typ ZAHL erwartet aber %s gefunden",
			toty,
		))
	}
	if s.StepSize != nil {
		stepty := t.Evaluate(s.StepSize)
		if stepty != token.ZAHL {
			t.err(s.To.Token(), fmt.Sprintf(
				"Es wurde ein Ausdruck vom Typ ZAHL erwartet aber %s gefunden",
				stepty,
			))
		}
	}
	return s.Body.Accept(t)
}
func (t *Typechecker) VisitFuncCallStmt(s *ast.FuncCallStmt) ast.Visitor {
	return s.Call.Accept(t)
}
func (t *Typechecker) VisitReturnStmt(s *ast.ReturnStmt) ast.Visitor {
	ty := t.Evaluate(s.Value)
	if fun, exists := t.CurrentTable.LookupFunc(s.Func); exists && fun.Type.Type != ty {
		t.err(s.Token(), fmt.Sprintf(
			"Eine Funktion mit Rückgabetyp %s kann keinen Wert vom Typ %s zurückgeben",
			fun.Type.Type,
			ty,
		))
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
