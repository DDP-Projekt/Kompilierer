package typechecker

import (
	"fmt"

	"github.com/DDP-Projekt/Kompilierer/pkg/ast"
	"github.com/DDP-Projekt/Kompilierer/pkg/scanner"
	"github.com/DDP-Projekt/Kompilierer/pkg/token"
)

// holds state to check if the types of an AST are valid
type Typechecker struct {
	ErrorHandler       scanner.ErrorHandler                // function to which errors are passed
	CurrentTable       *ast.SymbolTable                    // SymbolTable of the current scope (needed for name type-checking)
	Errored            bool                                // wether the typechecker found an error
	latestReturnedType token.DDPType                       // type of the last visited expression
	funcArgs           map[string]map[string]token.ArgType // for function parameter types
}

func New(symbols *ast.SymbolTable, errorHandler scanner.ErrorHandler) *Typechecker {
	return &Typechecker{
		ErrorHandler:       errorHandler,
		CurrentTable:       symbols,
		Errored:            false,
		latestReturnedType: token.DDPVoidType(),
		funcArgs:           make(map[string]map[string]token.ArgType),
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
func (t *Typechecker) Evaluate(expr ast.Expression) token.DDPType {
	t.visit(expr)
	return t.latestReturnedType
}

// helper for errors
func (t *Typechecker) err(tok token.Token, msg string, args ...any) {
	t.Errored = true
	t.ErrorHandler(tok, fmt.Sprintf(msg, args...))
}

// helper for commmon error message
func (t *Typechecker) errExpected(tok token.Token, got token.DDPType, expected ...token.DDPType) {
	msg := fmt.Sprintf("Der %s Operator erwartet einen Ausdruck vom Typ ", tok)
	if len(expected) == 1 {
		msg = fmt.Sprintf("Der %s Operator erwartet einen Ausdruck vom Typ %s aber hat '%s' bekommen", tok, expected[0], got)
	} else {
		for i, v := range expected {
			if i >= len(expected)-1 {
				break
			}
			msg += fmt.Sprintf("'%s', ", v)
		}
		msg += fmt.Sprintf("oder '%s' aber hat '%s' bekommen", expected[len(expected)-1], got)
	}
	t.err(tok, msg)
}

// helper for commmon error message
func (t *Typechecker) errExpectedBin(tok token.Token, t1, t2 token.DDPType, op token.TokenType) {
	t.err(tok, "Die Typen Kombination aus '%s' und '%s' passt nicht zu dem '%s' Operator", t1, t2, op)
}

// helper for commmon error message
func (t *Typechecker) errExpectedTern(tok token.Token, t1, t2, t3 token.DDPType, op token.TokenType) {
	t.err(tok, "Die Typen Kombination aus '%s', '%s' und '%s' passt nicht zu dem '%s' Operator", t1, t2, t3, op)
}

func (t *Typechecker) VisitBadDecl(decl *ast.BadDecl) ast.Visitor {
	t.Errored = true
	t.latestReturnedType = token.DDPVoidType()
	return t
}
func (t *Typechecker) VisitVarDecl(decl *ast.VarDecl) ast.Visitor {
	initialType := t.Evaluate(decl.InitVal)
	if initialType != decl.Type {
		t.err(decl.InitVal.Token(),
			"Ein Wert vom Typ %s kann keiner Variable vom Typ %s zugewiesen werden",
			initialType,
			decl.Type,
		)
	}
	return t
}
func (t *Typechecker) VisitFuncDecl(decl *ast.FuncDecl) ast.Visitor {
	t.funcArgs[decl.Name.Literal] = make(map[string]token.ArgType)
	for i, l := 0, len(decl.ParamNames); i < l; i++ {
		t.funcArgs[decl.Name.Literal][decl.ParamNames[i].Literal] = token.ArgType{
			Type:        decl.ParamTypes[i],
			IsReference: decl.IsReference[i],
		}
	}

	if !ast.IsExternFunc(decl) {
		return decl.Body.Accept(t)
	}
	return t
}

func (t *Typechecker) VisitBadExpr(expr *ast.BadExpr) ast.Visitor {
	t.Errored = true
	t.latestReturnedType = token.DDPVoidType()
	return t
}
func (t *Typechecker) VisitIdent(expr *ast.Ident) ast.Visitor {
	t.latestReturnedType, _ = t.CurrentTable.LookupVar(expr.Literal.Literal)
	return t
}
func (t *Typechecker) VisitIndexing(expr *ast.Indexing) ast.Visitor {
	if typ := t.Evaluate(expr.Index); typ != token.DDPIntType() {
		t.err(expr.Index.Token(), "Der STELLE Operator erwartet eine Zahl als zweiten Operanden, nicht %s", typ)
	}

	lhs := t.Evaluate(expr.Lhs)
	if !lhs.IsList && lhs.PrimitiveType != token.TEXT {
		t.err(expr.Lhs.Token(), "Der STELLE Operator erwartet einen Text oder eine Liste als ersten Operanden, nicht %s", lhs)
	}

	if lhs.IsList {
		t.latestReturnedType = token.NewPrimitiveType(lhs.PrimitiveType)
	} else {
		t.latestReturnedType = token.DDPCharType() // later on the list element type
	}
	return t
}
func (t *Typechecker) VisitIntLit(expr *ast.IntLit) ast.Visitor {
	t.latestReturnedType = token.DDPIntType()
	return t
}
func (t *Typechecker) VisitFLoatLit(expr *ast.FloatLit) ast.Visitor {
	t.latestReturnedType = token.DDPFloatType()
	return t
}
func (t *Typechecker) VisitBoolLit(expr *ast.BoolLit) ast.Visitor {
	t.latestReturnedType = token.DDPBoolType()
	return t
}
func (t *Typechecker) VisitCharLit(expr *ast.CharLit) ast.Visitor {
	t.latestReturnedType = token.DDPCharType()
	return t
}
func (t *Typechecker) VisitStringLit(expr *ast.StringLit) ast.Visitor {
	t.latestReturnedType = token.DDPStringType()
	return t
}
func (t *Typechecker) VisitListLit(expr *ast.ListLit) ast.Visitor {
	if expr.Values != nil {
		elementType := t.Evaluate(expr.Values[0])
		for _, v := range expr.Values[1:] {
			if ty := t.Evaluate(v); elementType != ty {
				t.err(v.Token(), "Falscher Typ (%s) in Listen Literal vom Typ %s", ty, elementType)
			}
		}
		expr.Type = token.NewListType(elementType.PrimitiveType)
	} else if expr.Count != nil && expr.Value != nil {
		if count := t.Evaluate(expr.Count); count != token.DDPIntType() {
			t.err(expr.Count.Token(), "Die Größe einer Liste muss als Zahl angegeben werden, nicht als %s", count)
		}
		if val := t.Evaluate(expr.Value); val != token.NewPrimitiveType(expr.Type.PrimitiveType) {
			t.err(expr.Value.Token(), "Falscher Typ (%s) in Listen Literal vom Typ %s", val, token.NewPrimitiveType(expr.Type.PrimitiveType))
		}
	}
	t.latestReturnedType = expr.Type
	return t
}
func (t *Typechecker) VisitUnaryExpr(expr *ast.UnaryExpr) ast.Visitor {
	// Evaluate the rhs expression and check if the operator fits it
	rhs := t.Evaluate(expr.Rhs)
	switch expr.Operator.Type {
	case token.BETRAG, token.NEGATE:
		if !rhs.IsNumeric() {
			t.errExpected(expr.Operator, rhs, token.DDPIntType(), token.DDPFloatType())
		}
	case token.NICHT:
		if !isOfType(rhs, token.DDPBoolType()) {
			t.errExpected(expr.Operator, rhs, token.DDPBoolType())
		}

		t.latestReturnedType = token.DDPBoolType()
	case token.NEGIERE:
		if !isOfType(rhs, token.DDPBoolType(), token.DDPIntType()) {
			t.errExpected(expr.Operator, rhs, token.DDPBoolType(), token.DDPIntType())
		}
	case token.LOGISCHNICHT:
		if !isOfType(rhs, token.DDPIntType()) {
			t.errExpected(expr.Operator, rhs, token.DDPIntType())
		}

		t.latestReturnedType = token.DDPIntType()
	case token.LÄNGE:
		if !rhs.IsList && rhs.PrimitiveType != token.TEXT {
			t.err(expr.Token(), "Der LÄNGE Operator erwartet einen Text oder eine Liste als Operanden, nicht %s", rhs)
		}

		t.latestReturnedType = token.DDPIntType()
	case token.GRÖßE:
		t.latestReturnedType = token.DDPIntType()
	case token.SINUS, token.KOSINUS, token.TANGENS,
		token.ARKSIN, token.ARKKOS, token.ARKTAN,
		token.HYPSIN, token.HYPKOS, token.HYPTAN:
		if !rhs.IsNumeric() {
			t.errExpected(expr.Operator, rhs, token.DDPIntType(), token.DDPFloatType())
		}

		t.latestReturnedType = token.DDPFloatType()
	default:
		t.err(expr.Operator, "Unbekannter unärer Operator '%s'", expr.Operator)
	}
	return t
}
func (t *Typechecker) VisitBinaryExpr(expr *ast.BinaryExpr) ast.Visitor {
	lhs := t.Evaluate(expr.Lhs)
	rhs := t.Evaluate(expr.Rhs)

	// helper to validate if types match
	validate := func(op token.TokenType, valid ...token.DDPType) {
		if !isOfTypeBin(lhs, rhs, valid...) {
			t.errExpectedBin(expr.Token(), lhs, rhs, op)
		}
	}

	switch op := expr.Operator.Type; op {
	case token.VERKETTET:
		if (!lhs.IsList && !rhs.IsList) && (lhs == token.DDPStringType() || rhs == token.DDPStringType()) { // string, char edge case
			validate(expr.Operator.Type, token.DDPStringType(), token.DDPCharType())
			t.latestReturnedType = token.DDPStringType()
		} else { // lists
			if lhs.PrimitiveType != rhs.PrimitiveType {
				t.err(expr.Operator, "Die Typenkombination aus %s und %s passt nicht zum VERKETTET Operator", lhs, rhs)
			}
			t.latestReturnedType = token.NewListType(lhs.PrimitiveType)
		}
	case token.PLUS, token.ADDIERE, token.ERHÖHE,
		token.MINUS, token.SUBTRAHIERE, token.VERRINGERE,
		token.MAL, token.MULTIPLIZIERE, token.VERVIELFACHE:
		validate(op, token.DDPIntType(), token.DDPFloatType())

		if lhs == token.DDPIntType() && rhs == token.DDPIntType() {
			t.latestReturnedType = token.DDPIntType()
		} else {
			t.latestReturnedType = token.DDPFloatType()
		}
	case token.STELLE:
		if !lhs.IsList && lhs != token.DDPStringType() {
			t.err(expr.Lhs.Token(), "Der STELLE Operator erwartet einen Text oder eine Liste als ersten Operanden, nicht %s", lhs)
		}
		if rhs != token.DDPIntType() {
			t.err(expr.Lhs.Token(), "Der STELLE Operator erwartet eine Zahl als zweiten Operanden, nicht %s", rhs)
		}

		if lhs.IsList {
			t.latestReturnedType = token.NewPrimitiveType(lhs.PrimitiveType)
		} else if lhs == token.DDPStringType() {
			t.latestReturnedType = token.DDPCharType() // later on the list element type
		}
	case token.DURCH, token.DIVIDIERE, token.TEILE, token.HOCH, token.LOGARITHMUS:
		validate(op, token.DDPIntType(), token.DDPFloatType())
		t.latestReturnedType = token.DDPFloatType()
	case token.MODULO:
		validate(op, token.DDPIntType())
		t.latestReturnedType = token.DDPIntType()
	case token.UND:
		validate(op, token.DDPBoolType())
		t.latestReturnedType = token.DDPBoolType()
	case token.ODER:
		validate(op, token.DDPBoolType())
		t.latestReturnedType = token.DDPBoolType()
	case token.LINKS:
		validate(op, token.DDPIntType())
		t.latestReturnedType = token.DDPIntType()
	case token.RECHTS:
		validate(op, token.DDPIntType())
		t.latestReturnedType = token.DDPIntType()
	case token.GLEICH:
		if lhs != rhs {
			t.errExpectedBin(expr.Token(), lhs, rhs, op)
		}

		t.latestReturnedType = token.DDPBoolType()
	case token.UNGLEICH:
		if lhs != rhs {
			t.errExpectedBin(expr.Token(), lhs, rhs, op)
		}

		t.latestReturnedType = token.DDPBoolType()
	case token.GRÖßERODER, token.KLEINER, token.KLEINERODER, token.GRÖßER:
		validate(op, token.DDPIntType(), token.DDPFloatType())
		t.latestReturnedType = token.DDPBoolType()
	case token.LOGISCHODER, token.LOGISCHUND, token.KONTRA:
		validate(op, token.DDPIntType())
		t.latestReturnedType = token.DDPIntType()
	default:
		t.err(expr.Operator, "Unbekannter binärer Operator '%s'", expr.Operator)
	}
	return t
}
func (t *Typechecker) VisitTernaryExpr(expr *ast.TernaryExpr) ast.Visitor {
	lhs := t.Evaluate(expr.Lhs)
	mid := t.Evaluate(expr.Mid)
	rhs := t.Evaluate(expr.Rhs)

	// helper to validate if types match
	validateBin := func(op token.TokenType, valid ...token.DDPType) {
		if !isOfTypeBin(mid, rhs, valid...) {
			t.errExpectedTern(expr.Token(), lhs, mid, rhs, op)
		}
	}

	switch expr.Operator.Type {
	case token.VONBIS:
		if !lhs.IsList && lhs != token.DDPStringType() {
			t.err(expr.Lhs.Token(), "Der VON_BIS Operator erwartet einen Text oder eine Liste als ersten Operanden, nicht %s", lhs)
		}

		validateBin(expr.Operator.Type, token.DDPIntType())
		if lhs.IsList {
			t.latestReturnedType = token.NewListType(lhs.PrimitiveType)
		} else if lhs == token.DDPStringType() {
			t.latestReturnedType = token.DDPStringType()
		}
	default:
		t.err(expr.Operator, "Unbekannter ternärer Operator '%s'", expr.Operator)
	}
	return t
}
func (t *Typechecker) VisitCastExpr(expr *ast.CastExpr) ast.Visitor {
	lhs := t.Evaluate(expr.Lhs)
	if expr.Type.IsList {
		switch expr.Type.PrimitiveType {
		case token.ZAHL:
			if !isOfType(lhs, token.DDPIntType()) {
				t.err(expr.Token(), "Ein Ausdruck vom Typ %s kann nicht zu einer Zahlen Liste umgewandelt werden", lhs)
			}
		case token.KOMMAZAHL:
			if !isOfType(lhs, token.DDPFloatType()) {
				t.err(expr.Token(), "Ein Ausdruck vom Typ %s kann nicht zu einer Kommazahlen Liste umgewandelt werden", lhs)
			}
		case token.BOOLEAN:
			if !isOfType(lhs, token.DDPBoolType()) {
				t.err(expr.Token(), "Ein Ausdruck vom Typ %s kann nicht zu einer Boolean Liste umgewandelt werden", lhs)
			}
		case token.BUCHSTABE:
			if !isOfType(lhs, token.DDPCharType(), token.DDPStringType()) {
				t.err(expr.Token(), "Ein Ausdruck vom Typ %s kann nicht zu einer Buchstaben Liste umgewandelt werden", lhs)
			}
		case token.TEXT:
			if !isOfType(lhs, token.DDPStringType()) {
				t.err(expr.Token(), "Ein Ausdruck vom Typ %s kann nicht zu einer Text Liste umgewandelt werden", lhs)
			}
		default:
			t.err(expr.Token(), "Invalide Typumwandlung von %s zu %s", lhs, expr.Type)
		}
	} else {
		switch expr.Type.PrimitiveType {
		case token.ZAHL:
			if !lhs.IsPrimitive() {
				t.err(expr.Token(), "Eine %s kann nicht zu einer Zahl umgewandelt werden", lhs)
			}
		case token.KOMMAZAHL:
			if !lhs.IsPrimitive() {
				t.err(expr.Token(), "Eine %s kann nicht zu einer Kommazahl umgewandelt werden", lhs)
			}
			if !isOfType(lhs, token.DDPStringType(), token.DDPIntType(), token.DDPFloatType()) {
				t.err(expr.Token(), "Ein Ausdruck vom Typ %s kann nicht zu einer Kommazahl umgewandelt werden", lhs)
			}
		case token.BOOLEAN:
			if !lhs.IsPrimitive() {
				t.err(expr.Token(), "Eine %s kann nicht zu einem Boolean umgewandelt werden", lhs)
			}
			if !isOfType(lhs, token.DDPIntType(), token.DDPBoolType()) {
				t.err(expr.Token(), "Ein Ausdruck vom Typ %s kann nicht zu einem Boolean umgewandelt werden", lhs)
			}
		case token.BUCHSTABE:
			if !lhs.IsPrimitive() {
				t.err(expr.Token(), "Eine %s kann nicht zu einem Buchstaben umgewandelt werden", lhs)
			}
			if !isOfType(lhs, token.DDPIntType(), token.DDPCharType()) {
				t.err(expr.Token(), "Ein Ausdruck vom Typ %s kann nicht zu einem Buchstaben umgewandelt werden", lhs)
			}
		case token.TEXT:
		default:
			t.err(expr.Token(), "Invalide Typumwandlung von %s zu %s", lhs, expr.Type)
		}
	}
	t.latestReturnedType = expr.Type
	return t
}
func (t *Typechecker) VisitGrouping(expr *ast.Grouping) ast.Visitor {
	return expr.Expr.Accept(t)
}
func (t *Typechecker) VisitFuncCall(callExpr *ast.FuncCall) ast.Visitor {
	for k, expr := range callExpr.Args {
		tokenType := t.Evaluate(expr)

		argType := t.funcArgs[callExpr.Name][k]
		if ass, ok := expr.(ast.Assigneable); argType.IsReference && !ok {
			t.err(expr.Token(), "Es wurde ein Referenz-Typ erwartet aber ein Ausdruck gefunden")
		} else if ass, ok := ass.(*ast.Indexing); argType.IsReference && argType.Type == token.DDPCharType() && ok {
			lhs := t.Evaluate(ass.Lhs)
			if lhs.PrimitiveType == token.TEXT {
				t.err(expr.Token(), "Ein Buchstabe in einem Text kann nicht als Buchstaben Referenz übergeben werden")
			}
		}
		if tokenType != argType.Type {
			t.err(expr.Token(),
				"Die Funktion %s erwartet einen Wert vom Typ %s für den Parameter %s, aber hat %s bekommen",
				callExpr.Name,
				argType,
				k,
				tokenType,
			)
		}
	}
	fun, _ := t.CurrentTable.LookupFunc(callExpr.Name)
	t.latestReturnedType = fun.Type
	return t
}

func (t *Typechecker) VisitBadStmt(stmt *ast.BadStmt) ast.Visitor {
	t.Errored = true
	t.latestReturnedType = token.DDPVoidType()
	return t
}
func (t *Typechecker) VisitDeclStmt(stmt *ast.DeclStmt) ast.Visitor {
	return stmt.Decl.Accept(t)
}
func (t *Typechecker) VisitExprStmt(stmt *ast.ExprStmt) ast.Visitor {
	return stmt.Expr.Accept(t)
}
func (t *Typechecker) VisitAssignStmt(stmt *ast.AssignStmt) ast.Visitor {
	rhs := t.Evaluate(stmt.Rhs)
	switch assign := stmt.Var.(type) {
	case *ast.Ident:
		if vartyp, exists := t.CurrentTable.LookupVar(assign.Literal.Literal); exists && vartyp != rhs {
			t.err(stmt.Rhs.Token(),
				"Ein Wert vom Typ %s kann keiner Variable vom Typ %s zugewiesen werden",
				rhs,
				vartyp,
			)
		}
	case *ast.Indexing:
		if typ := t.Evaluate(assign.Index); typ != token.DDPIntType() {
			t.err(assign.Index.Token(), "Der STELLE Operator erwartet eine Zahl als zweiten Operanden, nicht %s", typ)
		}

		lhs := t.Evaluate(assign.Lhs)
		if !lhs.IsList && lhs != token.DDPStringType() {
			t.err(assign.Lhs.Token(), "Der STELLE Operator erwartet einen Text oder eine Liste als ersten Operanden, nicht %s", lhs)
		}
		if lhs.IsList {
			lhs = token.NewPrimitiveType(lhs.PrimitiveType)
		} else if lhs == token.DDPStringType() {
			lhs = token.DDPCharType()
		}

		if lhs != rhs {
			t.err(stmt.Rhs.Token(),
				"Ein Wert vom Typ %s kann keiner Variable vom Typ %s zugewiesen werden",
				rhs,
				lhs,
			)
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
	if conditionType != token.DDPBoolType() {
		t.err(stmt.Condition.Token(),
			"Die Bedingung einer WENN Anweisung muss vom Typ Boolean sein, war aber vom Typ %s",
			conditionType,
		)
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
		if conditionType != token.DDPBoolType() {
			t.err(stmt.Condition.Token(),
				"Die Bedingung einer SOLANGE Anweisung muss vom Typ BOOLEAN sein, war aber vom Typ %s",
				conditionType,
			)
		}
	case token.MAL:
		if conditionType != token.DDPIntType() {
			t.err(stmt.Condition.Token(),
				"Die Anzahl an Wiederholungen einer WIEDERHOLE Anweisung muss vom Typ ZAHL sein, war aber vom Typ %s",
				conditionType,
			)
		}
	}
	return stmt.Body.Accept(t)
}
func (t *Typechecker) VisitForStmt(stmt *ast.ForStmt) ast.Visitor {
	t.visit(stmt.Initializer)
	toType := t.Evaluate(stmt.To)
	if toType != token.DDPIntType() {
		t.err(stmt.To.Token(),
			"Es wurde ein Ausdruck vom Typ ZAHL erwartet aber %s gefunden",
			toType,
		)
	}
	if stmt.StepSize != nil {
		stepType := t.Evaluate(stmt.StepSize)
		if stepType != token.DDPIntType() {
			t.err(stmt.To.Token(),
				"Es wurde ein Ausdruck vom Typ ZAHL erwartet aber %s gefunden",
				stepType,
			)
		}
	}
	return stmt.Body.Accept(t)
}
func (t *Typechecker) VisitForRangeStmt(stmt *ast.ForRangeStmt) ast.Visitor {
	elementType := stmt.Initializer.Type
	inType := t.Evaluate(stmt.In)

	if !inType.IsList && inType != token.DDPStringType() {
		t.err(stmt.In.Token(), "Man kann nur über Texte oder Listen iterieren")
	}

	if inType.IsList && elementType != token.NewPrimitiveType(inType.PrimitiveType) {
		t.err(stmt.Initializer.Token(),
			"Es wurde ein Ausdruck vom Typ %s erwartet aber %s gefunden",
			token.NewListType(elementType.PrimitiveType), inType,
		)
	} else if inType == token.DDPStringType() && elementType != token.DDPCharType() {
		t.err(stmt.Initializer.Token(),
			"Es wurde ein Ausdruck vom Typ Buchstabe erwartet aber %s gefunden",
			elementType,
		)
	}

	return stmt.Body.Accept(t)
}
func (t *Typechecker) VisitFuncCallStmt(stmt *ast.FuncCallStmt) ast.Visitor {
	return stmt.Call.Accept(t)
}
func (t *Typechecker) VisitReturnStmt(stmt *ast.ReturnStmt) ast.Visitor {
	returnType := token.DDPVoidType()
	if stmt.Value != nil {
		returnType = t.Evaluate(stmt.Value)
	}
	if fun, exists := t.CurrentTable.LookupFunc(stmt.Func); exists && fun.Type != returnType {
		t.err(stmt.Token(),
			"Eine Funktion mit Rückgabetyp %s kann keinen Wert vom Typ %s zurückgeben",
			fun.Type,
			returnType,
		)
	}
	return t
}

// checks if t is contained in types
func isOfType(t token.DDPType, types ...token.DDPType) bool {
	for _, v := range types {
		if t == v {
			return true
		}
	}
	return false
}

// checks if t1 and t2 are both contained in types
func isOfTypeBin(t1, t2 token.DDPType, types ...token.DDPType) bool {
	return isOfType(t1, types...) && isOfType(t2, types...)
}
