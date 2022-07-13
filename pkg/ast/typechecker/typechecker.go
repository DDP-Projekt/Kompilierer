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
	typechecker := New(Ast.Symbols, errorHandler)

	for i, l := 0, len(Ast.Statements); i < l; i++ {
		Ast.Statements[i].Accept(typechecker)
	}

	if typechecker.Errored {
		Ast.Faulty = true
	}
}

// typecheck a single node
func (t *Typechecker) TypecheckNode(node ast.Node) *Typechecker {
	return node.Accept(t).(*Typechecker)
}

// helper to visit a node
func (t *Typechecker) visit(node ast.Node) {
	node.Accept(t)
}

// Evaluates the type of an expression
func (t *Typechecker) Evaluate(expr ast.Expression) token.TokenType {
	t.visit(expr)
	return t.latestReturnedType
}

// helper for errors
func (t *Typechecker) err(tok token.Token, msg string) {
	t.Errored = true
	t.ErrorHandler(tok, msg)
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

// helper for commmon error message
func (t *Typechecker) errExpectedTern(tok token.Token, t1, t2, t3, op token.TokenType) {
	t.err(tok, fmt.Sprintf("Die Typen Kombination aus '%s', '%s' und '%s' passt nicht zu dem '%s' Operator", t1, t2, t3, op))
}

func (t *Typechecker) VisitBadDecl(decl *ast.BadDecl) ast.Visitor {
	t.Errored = true
	t.latestReturnedType = token.NICHTS
	return t
}
func (t *Typechecker) VisitVarDecl(decl *ast.VarDecl) ast.Visitor {
	tokenType := t.Evaluate(decl.InitVal)
	if tokenType != decl.Type.Type {
		t.err(decl.InitVal.Token(), fmt.Sprintf(
			"Ein Wert vom Typ %s kann keiner Variable vom Typ %s zugewiesen werden",
			tokenType,
			decl.Type.Type,
		))
	}
	return t
}
func (t *Typechecker) VisitFuncDecl(decl *ast.FuncDecl) ast.Visitor {
	t.funcArgs[decl.Name.Literal] = make(map[string]token.TokenType)
	for i, l := 0, len(decl.ParamNames); i < l; i++ {
		t.funcArgs[decl.Name.Literal][decl.ParamNames[i].Literal] = decl.ParamTypes[i].Type
	}

	return decl.Body.Accept(t)
}

func (t *Typechecker) VisitBadExpr(expr *ast.BadExpr) ast.Visitor {
	t.Errored = true
	t.latestReturnedType = token.NICHTS
	return t
}
func (t *Typechecker) VisitIdent(expr *ast.Ident) ast.Visitor {
	t.latestReturnedType, _ = t.CurrentTable.LookupVar(expr.Literal.Literal)
	return t
}
func (t *Typechecker) VisitIndexing(expr *ast.Indexing) ast.Visitor {
	if t.Evaluate(expr.Index) != token.ZAHL {
		t.err(expr.Index.Token(), "Der STELLE Operator erwartet eine Zahl als zweiten Operanden")
	}

	if t.Evaluate(expr.Name) != token.TEXT {
		t.err(expr.Name.Token(), "Der STELLE Operator erwartet einen Text als ersten Operanden")
	}

	t.latestReturnedType = token.BUCHSTABE // later on the list element type
	return t
}
func (t *Typechecker) VisitIntLit(expr *ast.IntLit) ast.Visitor {
	t.latestReturnedType = token.ZAHL
	return t
}
func (t *Typechecker) VisitFLoatLit(expr *ast.FloatLit) ast.Visitor {
	t.latestReturnedType = token.KOMMAZAHL
	return t
}
func (t *Typechecker) VisitBoolLit(expr *ast.BoolLit) ast.Visitor {
	t.latestReturnedType = token.BOOLEAN
	return t
}
func (t *Typechecker) VisitCharLit(expr *ast.CharLit) ast.Visitor {
	t.latestReturnedType = token.BUCHSTABE
	return t
}
func (t *Typechecker) VisitStringLit(expr *ast.StringLit) ast.Visitor {
	t.latestReturnedType = token.TEXT
	return t
}
func (t *Typechecker) VisitUnaryExpr(expr *ast.UnaryExpr) ast.Visitor {
	// Evaluate the rhs expression and check if the operator fits it
	rhs := t.Evaluate(expr.Rhs)
	// boolean vs bitwise negate (compound assignement)
	/*if expr.Operator.Type == token.NEGIERE && rhs == token.ZAHL {
		expr.Operator.Type == token.LOGISCHNICHT
	}*/
	switch expr.Operator.Type {
	case token.BETRAG, token.NEGATE:
		if !isType(rhs, token.ZAHL, token.KOMMAZAHL) {
			t.errExpected(expr.Operator, rhs, token.ZAHL, token.KOMMAZAHL)
		}
	case token.NICHT:
		if !isType(rhs, token.BOOLEAN) {
			t.errExpected(expr.Operator, rhs, token.BOOLEAN)
		}

		t.latestReturnedType = token.BOOLEAN
	case token.NEGIERE:
		if !isType(rhs, token.BOOLEAN, token.ZAHL) {
			t.errExpected(expr.Operator, rhs, token.BOOLEAN, token.ZAHL)
		}
	case token.LOGISCHNICHT:
		if !isType(rhs, token.ZAHL) {
			t.errExpected(expr.Operator, rhs, token.ZAHL)
		}

		t.latestReturnedType = token.ZAHL
	case token.LÄNGE:
		if !isType(rhs, token.TEXT) {
			t.errExpected(expr.Operator, rhs, token.TEXT)
		}

		t.latestReturnedType = token.ZAHL // some operators change the type of the rhs expression, so we set that
	case token.GRÖßE:
		t.latestReturnedType = token.ZAHL
	case token.SINUS, token.KOSINUS, token.TANGENS,
		token.ARKSIN, token.ARKKOS, token.ARKTAN,
		token.HYPSIN, token.HYPKOS, token.HYPTAN:
		if !isType(rhs, token.ZAHL, token.KOMMAZAHL) {
			t.errExpected(expr.Operator, rhs, token.ZAHL, token.KOMMAZAHL)
		}

		t.latestReturnedType = token.KOMMAZAHL
	case token.ZAHL:
		if !isType(rhs, token.KOMMAZAHL, token.ZAHL, token.TEXT, token.BOOLEAN, token.BUCHSTABE) {
			t.errExpected(expr.Operator, rhs, token.KOMMAZAHL, token.ZAHL, token.TEXT, token.BOOLEAN, token.BUCHSTABE)
		}

		t.latestReturnedType = token.ZAHL
	case token.KOMMAZAHL:
		if !isType(rhs, token.TEXT, token.ZAHL, token.KOMMAZAHL) {
			t.errExpected(expr.Operator, rhs, token.ZAHL, token.TEXT, token.KOMMAZAHL)
		}

		t.latestReturnedType = token.KOMMAZAHL
	case token.BOOLEAN:
		if !isType(rhs, token.ZAHL, token.BOOLEAN) {
			t.errExpected(expr.Operator, rhs, token.ZAHL, token.BOOLEAN)
		}

		t.latestReturnedType = token.BOOLEAN
	case token.BUCHSTABE:
		if !isType(rhs, token.ZAHL, token.BUCHSTABE) {
			t.errExpected(expr.Operator, rhs, token.ZAHL, token.BUCHSTABE)
		}

		t.latestReturnedType = token.BUCHSTABE
	case token.TEXT:
		if !isType(rhs, token.ZAHL, token.KOMMAZAHL, token.BOOLEAN, token.TEXT, token.BUCHSTABE) {
			t.errExpected(expr.Operator, rhs, token.ZAHL, token.KOMMAZAHL, token.BOOLEAN, token.TEXT, token.BUCHSTABE)
		}

		t.latestReturnedType = token.TEXT
	default:
		t.err(expr.Operator, fmt.Sprintf("Unbekannter unärer Operator '%s'", expr.Operator.String()))
	}
	return t
}
func (t *Typechecker) VisitBinaryExpr(expr *ast.BinaryExpr) ast.Visitor {
	lhs := t.Evaluate(expr.Lhs)
	rhs := t.Evaluate(expr.Rhs)

	// helper to validate if types match
	validate := func(op token.TokenType, valid ...token.TokenType) {
		if !isTypeBin(lhs, rhs, valid...) {
			t.errExpectedBin(expr.Token(), lhs, rhs, op)
		}
	}

	switch op := expr.Operator.Type; op {
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
			t.err(expr.Lhs.Token(), "Der STELLE Operator erwartet einen Text als ersten Operanden")
		}
		if rhs != token.ZAHL {
			t.err(expr.Lhs.Token(), "Der STELLE Operator erwartet eine Zahl als zweiten Operanden")
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
			t.errExpectedBin(expr.Token(), lhs, rhs, op)
		}

		t.latestReturnedType = token.BOOLEAN
	case token.UNGLEICH:
		if lhs != rhs {
			t.errExpectedBin(expr.Token(), lhs, rhs, op)
		}

		t.latestReturnedType = token.BOOLEAN
	case token.GRÖßERODER, token.KLEINER, token.KLEINERODER, token.GRÖßER:
		validate(op, token.ZAHL, token.KOMMAZAHL)
		t.latestReturnedType = token.BOOLEAN
	case token.LOGISCHODER, token.LOGISCHUND, token.KONTRA:
		validate(op, token.ZAHL)
		t.latestReturnedType = token.ZAHL
	default:
		t.err(expr.Operator, fmt.Sprintf("Unbekannter binärer Operator '%s'", expr.Operator.String()))
	}
	return t
}
func (t *Typechecker) VisitTernaryExpr(expr *ast.TernaryExpr) ast.Visitor {
	lhs := t.Evaluate(expr.Lhs)
	mid := t.Evaluate(expr.Mid)
	rhs := t.Evaluate(expr.Rhs)

	// helper to validate if types match
	validateBin := func(op token.TokenType, valid ...token.TokenType) {
		if !isTypeBin(mid, rhs, valid...) {
			t.errExpectedTern(expr.Token(), lhs, mid, rhs, op)
		}
	}

	switch expr.Operator.Type {
	case token.VONBIS:
		if lhs != token.TEXT {
			t.errExpectedTern(expr.Token(), lhs, mid, rhs, expr.Operator.Type)
		}

		validateBin(expr.Operator.Type, token.ZAHL)
		t.latestReturnedType = token.TEXT
	default:
		t.err(expr.Operator, fmt.Sprintf("Unbekannter ternärer Operator '%s'", expr.Operator.String()))
	}
	return t
}
func (t *Typechecker) VisitGrouping(expr *ast.Grouping) ast.Visitor {
	return expr.Expr.Accept(t)
}
func (t *Typechecker) VisitFuncCall(callExpr *ast.FuncCall) ast.Visitor {
	for k, expr := range callExpr.Args {
		tokenType := t.Evaluate(expr)

		if tokenType != t.funcArgs[callExpr.Name][k] {
			t.err(expr.Token(), fmt.Sprintf(
				"Die Funktion %s erwartet einen Wert vom Typ %s für den Parameter %s, aber hat %s bekommen",
				callExpr.Name,
				t.funcArgs[callExpr.Name][k],
				k,
				tokenType,
			))
		}
	}
	fun, _ := t.CurrentTable.LookupFunc(callExpr.Name)
	t.latestReturnedType = fun.Type.Type
	return t
}

func (t *Typechecker) VisitBadStmt(stmt *ast.BadStmt) ast.Visitor {
	t.Errored = true
	t.latestReturnedType = token.NICHTS
	return t
}
func (t *Typechecker) VisitDeclStmt(stmt *ast.DeclStmt) ast.Visitor {
	return stmt.Decl.Accept(t)
}
func (t *Typechecker) VisitExprStmt(stmt *ast.ExprStmt) ast.Visitor {
	return stmt.Expr.Accept(t)
}
func (t *Typechecker) VisitAssignStmt(stmt *ast.AssignStmt) ast.Visitor {
	tokenType := t.Evaluate(stmt.Rhs)
	switch assign := stmt.Var.(type) {
	case *ast.Ident:
		if vartyp, exists := t.CurrentTable.LookupVar(assign.Literal.Literal); exists && vartyp != tokenType {
			t.err(stmt.Rhs.Token(), fmt.Sprintf(
				"Ein Wert vom Typ %s kann keiner Variable vom Typ %s zugewiesen werden",
				tokenType,
				vartyp,
			))
		}
	case *ast.Indexing:
		if t.Evaluate(assign.Index) != token.ZAHL {
			t.err(assign.Index.Token(), "Der STELLE Operator erwartet eine Zahl als zweiten Operanden")
		}

		lhs := t.Evaluate(assign.Name)
		switch lhs {
		case token.TEXT:
			lhs = token.BUCHSTABE
		default:
			t.err(assign.Name.Token(), "Der STELLE Operator erwartet einen Text oder eine Liste als ersten Operanden")
		}

		if lhs != tokenType {
			t.err(stmt.Rhs.Token(), fmt.Sprintf(
				"Ein Wert vom Typ %s kann keiner Variable vom Typ %s zugewiesen werden",
				tokenType,
				lhs,
			))
		}
	}

	return t
}
func (t *Typechecker) VisitBlockStmt(stmt *ast.BlockStmt) ast.Visitor {
	t.CurrentTable = stmt.Symbols
	for _, stmt := range stmt.Statements {
		t.visit(stmt)
	}

	t.CurrentTable = stmt.Symbols.Enclosing
	return t
}
func (t *Typechecker) VisitIfStmt(stmt *ast.IfStmt) ast.Visitor {
	conditionType := t.Evaluate(stmt.Condition)
	if conditionType != token.BOOLEAN {
		t.err(stmt.Condition.Token(), fmt.Sprintf(
			"Die Bedingung einer WENN Anweisung muss vom Typ Boolean sein, war aber vom Typ %s",
			conditionType,
		))
	}
	t.visit(stmt.Then)
	if stmt.Else != nil {
		t.visit(stmt.Else)
	}
	return t
}
func (t *Typechecker) VisitWhileStmt(stmt *ast.WhileStmt) ast.Visitor {
	conditionType := t.Evaluate(stmt.Condition)
	switch stmt.While.Type {
	case token.SOLANGE, token.MACHE:
		if conditionType != token.BOOLEAN {
			t.err(stmt.Condition.Token(), fmt.Sprintf(
				"Die Bedingung einer SOLANGE Anweisung muss vom Typ BOOLEAN sein, war aber vom Typ %s",
				conditionType,
			))
		}
	case token.MAL:
		if conditionType != token.ZAHL {
			t.err(stmt.Condition.Token(), fmt.Sprintf(
				"Die Anzahl an Wiederholungen einer WIEDERHOLE Anweisung muss vom Typ ZAHL sein, war aber vom Typ %s",
				conditionType,
			))
		}
	}
	return stmt.Body.Accept(t)
}
func (t *Typechecker) VisitForStmt(stmt *ast.ForStmt) ast.Visitor {
	t.visit(stmt.Initializer)
	toType := t.Evaluate(stmt.To)
	if toType != token.ZAHL {
		t.err(stmt.To.Token(), fmt.Sprintf(
			"Es wurde ein Ausdruck vom Typ ZAHL erwartet aber %s gefunden",
			toType,
		))
	}
	if stmt.StepSize != nil {
		stepType := t.Evaluate(stmt.StepSize)
		if stepType != token.ZAHL {
			t.err(stmt.To.Token(), fmt.Sprintf(
				"Es wurde ein Ausdruck vom Typ ZAHL erwartet aber %s gefunden",
				stepType,
			))
		}
	}
	return stmt.Body.Accept(t)
}
func (t *Typechecker) VisitForRangeStmt(stmt *ast.ForRangeStmt) ast.Visitor {
	elementType := stmt.Initializer.Type.Type
	inType := t.Evaluate(stmt.In)
	switch inType {
	case token.TEXT:
		if elementType != token.BUCHSTABE {
			t.err(stmt.Initializer.Token(), fmt.Sprintf(
				"Es wurde ein Ausdruck vom Typ BUCHSTABE erwartet aber %s gefunden",
				elementType,
			))
		}
	default:
		t.err(stmt.Token(), fmt.Sprintf(
			"Es wurde ein Ausdruck vom Typ TEXT erwartet aber %s gefunden",
			inType,
		))
	}
	return stmt.Body.Accept(t)
}
func (t *Typechecker) VisitFuncCallStmt(stmt *ast.FuncCallStmt) ast.Visitor {
	return stmt.Call.Accept(t)
}
func (t *Typechecker) VisitReturnStmt(stmt *ast.ReturnStmt) ast.Visitor {
	tokenType := t.Evaluate(stmt.Value)
	if fun, exists := t.CurrentTable.LookupFunc(stmt.Func); exists && fun.Type.Type != tokenType {
		t.err(stmt.Token(), fmt.Sprintf(
			"Eine Funktion mit Rückgabetyp %s kann keinen Wert vom Typ %s zurückgeben",
			fun.Type.Type,
			tokenType,
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
