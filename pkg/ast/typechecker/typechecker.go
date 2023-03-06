package typechecker

import (
	"fmt"

	"github.com/DDP-Projekt/Kompilierer/pkg/ast"
	"github.com/DDP-Projekt/Kompilierer/pkg/ddperror"
	"github.com/DDP-Projekt/Kompilierer/pkg/ddptypes"
	"github.com/DDP-Projekt/Kompilierer/pkg/token"
)

// holds state to check if the types of an AST are valid
type Typechecker struct {
	ErrorHandler       ddperror.Handler // function to which errors are passed
	CurrentTable       *ast.SymbolTable // SymbolTable of the current scope (needed for name type-checking)
	Errored            bool             // wether the typechecker found an error
	latestReturnedType ddptypes.Type    // type of the last visited expression
}

func New(symbols *ast.SymbolTable, errorHandler ddperror.Handler) *Typechecker {
	if errorHandler == nil {
		errorHandler = ddperror.EmptyHandler
	}
	return &Typechecker{
		ErrorHandler:       errorHandler,
		CurrentTable:       symbols,
		Errored:            false,
		latestReturnedType: ddptypes.Void(),
	}
}

// checks that all ast nodes fulfill type requirements
func TypecheckAst(Ast *ast.Ast, errorHandler ddperror.Handler) {
	typechecker := New(Ast.Symbols, errorHandler)

	for i, l := 0, len(Ast.Statements); i < l; i++ {
		Ast.Statements[i].Accept(typechecker)
	}

	if typechecker.Errored {
		Ast.Faulty = true
	}
}

// typecheck a single node
func (t *Typechecker) TypecheckNode(node ast.Node) {
	node.Accept(t)
}

// helper to visit a node
func (t *Typechecker) visit(node ast.Node) {
	node.Accept(t)
}

// Evaluates the type of an expression
func (t *Typechecker) Evaluate(expr ast.Expression) ddptypes.Type {
	t.visit(expr)
	return t.latestReturnedType
}

// calls Evaluate but uses ddperror.EmptyHandler as error handler
func (t *Typechecker) EvaluateSilent(expr ast.Expression) ddptypes.Type {
	errHndl, errored := t.ErrorHandler, t.Errored
	t.ErrorHandler = ddperror.EmptyHandler
	ty := t.Evaluate(expr)
	t.ErrorHandler, t.Errored = errHndl, errored
	return ty
}

// helper for errors
func (t *Typechecker) err(code ddperror.Code, Range token.Range, msg string, file string) {
	t.Errored = true
	t.ErrorHandler(ddperror.New(code, Range, msg, file))
}

// helper to not always pass range and file
func (t *Typechecker) errExpr(code ddperror.Code, expr ast.Expression, msgfmt string, fmtargs ...any) {
	t.err(code, expr.GetRange(), fmt.Sprintf(msgfmt, fmtargs...), expr.Token().File)
}

// helper for commmon error message
func (t *Typechecker) errExpected(tok token.Token, expr ast.Expression, got ddptypes.Type, expected ...ddptypes.Type) {
	msg := fmt.Sprintf("Der %s Operator erwartet einen Ausdruck vom Typ ", tok)
	if len(expected) == 1 {
		msg = fmt.Sprintf("Der %s Operator erwartet einen Ausdruck vom Typ %s", tok, expected[0])
	} else {
		for i, v := range expected {
			if i >= len(expected)-1 {
				break
			}
			msg += fmt.Sprintf("'%s', ", v)
		}
		msg += fmt.Sprintf("oder '%s'", expected[len(expected)-1])
	}
	t.errExpr(ddperror.TYP_TYPE_MISMATCH, expr, msg+" aber hat '%s' bekommen", got)
}

func (*Typechecker) BaseVisitor() {}

func (t *Typechecker) VisitBadDecl(decl *ast.BadDecl) {
	t.latestReturnedType = ddptypes.Void()
}
func (t *Typechecker) VisitVarDecl(decl *ast.VarDecl) {
	initialType := t.Evaluate(decl.InitVal)
	if initialType != decl.Type {
		msg := fmt.Sprintf("Ein Wert vom Typ %s kann keiner Variable vom Typ %s zugewiesen werden", initialType, decl.Type)
		t.errExpr(ddperror.TYP_BAD_ASSIGNEMENT,
			decl.InitVal,
			msg,
		)
	}
}
func (t *Typechecker) VisitFuncDecl(decl *ast.FuncDecl) {
	if !ast.IsExternFunc(decl) {
		decl.Body.Accept(t)
	}
}

func (t *Typechecker) VisitBadExpr(expr *ast.BadExpr) {
	t.latestReturnedType = ddptypes.Void()
}
func (t *Typechecker) VisitIdent(expr *ast.Ident) {
	decl, ok := t.CurrentTable.LookupVar(expr.Literal.Literal)
	if !ok {
		t.latestReturnedType = ddptypes.Void()
	} else {
		t.latestReturnedType = decl.Type
	}
}
func (t *Typechecker) VisitIndexing(expr *ast.Indexing) {
	if typ := t.Evaluate(expr.Index); typ != ddptypes.Int() {
		t.errExpr(ddperror.TYP_BAD_INDEXING, expr.Index, "Der STELLE Operator erwartet eine Zahl als zweiten Operanden, nicht %s", typ)
	}

	lhs := t.Evaluate(expr.Lhs)
	if !lhs.IsList && lhs.Primitive != ddptypes.TEXT {
		t.errExpr(ddperror.TYP_BAD_INDEXING, expr.Lhs, "Der STELLE Operator erwartet einen Text oder eine Liste als ersten Operanden, nicht %s", lhs)
	}

	if lhs.IsList {
		t.latestReturnedType = ddptypes.Primitive(lhs.Primitive)
	} else {
		t.latestReturnedType = ddptypes.Char() // later on the list element type
	}
}
func (t *Typechecker) VisitIntLit(expr *ast.IntLit) {
	t.latestReturnedType = ddptypes.Int()
}
func (t *Typechecker) VisitFloatLit(expr *ast.FloatLit) {
	t.latestReturnedType = ddptypes.Float()
}
func (t *Typechecker) VisitBoolLit(expr *ast.BoolLit) {
	t.latestReturnedType = ddptypes.Bool()
}
func (t *Typechecker) VisitCharLit(expr *ast.CharLit) {
	t.latestReturnedType = ddptypes.Char()
}
func (t *Typechecker) VisitStringLit(expr *ast.StringLit) {
	t.latestReturnedType = ddptypes.String()
}
func (t *Typechecker) VisitListLit(expr *ast.ListLit) {
	if expr.Values != nil {
		elementType := t.Evaluate(expr.Values[0])
		for _, v := range expr.Values[1:] {
			if ty := t.Evaluate(v); elementType != ty {
				msg := fmt.Sprintf("Falscher Typ (%s) in Listen Literal vom Typ %s", ty, elementType)
				t.errExpr(ddperror.TYP_BAD_LIST_LITERAL, v, msg)
			}
		}
		expr.Type = ddptypes.List(elementType.Primitive)
	} else if expr.Count != nil && expr.Value != nil {
		if count := t.Evaluate(expr.Count); count != ddptypes.Int() {
			t.errExpr(ddperror.TYP_BAD_LIST_LITERAL, expr, "Die Größe einer Liste muss als Zahl angegeben werden, nicht als %s", count)
		}
		if val := t.Evaluate(expr.Value); val != ddptypes.Primitive(expr.Type.Primitive) {
			t.errExpr(ddperror.TYP_BAD_LIST_LITERAL, expr, "Falscher Typ (%s) in Listen Literal vom Typ %s", val, ddptypes.Primitive(expr.Type.Primitive))
		}
	}
	t.latestReturnedType = expr.Type
}
func (t *Typechecker) VisitUnaryExpr(expr *ast.UnaryExpr) {
	// Evaluate the rhs expression and check if the operator fits it
	rhs := t.Evaluate(expr.Rhs)
	switch expr.Operator.Type {
	case token.BETRAG, token.NEGATE:
		if !rhs.IsNumeric() {
			t.errExpected(expr.Operator, expr.Rhs, rhs, ddptypes.Int(), ddptypes.Float())
		}
	case token.NICHT:
		if !isOfType(rhs, ddptypes.Bool()) {
			t.errExpected(expr.Operator, expr.Rhs, rhs, ddptypes.Bool())
		}

		t.latestReturnedType = ddptypes.Bool()
	case token.NEGIERE:
		if !isOfType(rhs, ddptypes.Bool(), ddptypes.Int()) {
			t.errExpected(expr.Operator, expr.Rhs, rhs, ddptypes.Bool(), ddptypes.Int())
		}
	case token.LOGISCHNICHT:
		if !isOfType(rhs, ddptypes.Int()) {
			t.errExpected(expr.Operator, expr.Rhs, rhs, ddptypes.Int())
		}

		t.latestReturnedType = ddptypes.Int()
	case token.LÄNGE:
		if !rhs.IsList && rhs.Primitive != ddptypes.TEXT {
			t.errExpr(ddperror.TYP_TYPE_MISMATCH, expr, "Der LÄNGE Operator erwartet einen Text oder eine Liste als Operanden, nicht %s", rhs)
		}

		t.latestReturnedType = ddptypes.Int()
	case token.GRÖßE:
		t.latestReturnedType = ddptypes.Int()
	default:
		panic(fmt.Errorf("unbekannter unärer Operator '%s'", expr.Operator))
	}
}
func (t *Typechecker) VisitBinaryExpr(expr *ast.BinaryExpr) {
	lhs := t.Evaluate(expr.Lhs)
	rhs := t.Evaluate(expr.Rhs)

	// helper to validate if types match
	validate := func(valid ...ddptypes.Type) {
		if !isOfType(lhs, valid...) {
			t.errExpected(expr.Operator, expr.Lhs, lhs, valid...)
		}
		if !isOfType(rhs, valid...) {
			t.errExpected(expr.Operator, expr.Rhs, rhs, valid...)
		}
	}

	switch op := expr.Operator.Type; op {
	case token.VERKETTET:
		if (!lhs.IsList && !rhs.IsList) && (lhs == ddptypes.String() || rhs == ddptypes.String()) { // string, char edge case
			validate(ddptypes.String(), ddptypes.Char())
			t.latestReturnedType = ddptypes.String()
		} else { // lists
			if lhs.Primitive != rhs.Primitive {
				t.errExpr(ddperror.TYP_TYPE_MISMATCH, expr, "Die Typenkombination aus %s und %s passt nicht zum VERKETTET Operator", lhs, rhs)
			}
			t.latestReturnedType = ddptypes.List(lhs.Primitive)
		}
	case token.PLUS, token.ADDIERE, token.ERHÖHE,
		token.MINUS, token.SUBTRAHIERE, token.VERRINGERE,
		token.MAL, token.MULTIPLIZIERE, token.VERVIELFACHE:
		validate(ddptypes.Int(), ddptypes.Float())

		if lhs == ddptypes.Int() && rhs == ddptypes.Int() {
			t.latestReturnedType = ddptypes.Int()
		} else {
			t.latestReturnedType = ddptypes.Float()
		}
	case token.STELLE:
		if !lhs.IsList && lhs != ddptypes.String() {
			t.errExpr(ddperror.TYP_TYPE_MISMATCH, expr.Lhs, "Der STELLE Operator erwartet einen Text oder eine Liste als ersten Operanden, nicht %s", lhs)
		}
		if rhs != ddptypes.Int() {
			t.errExpr(ddperror.TYP_TYPE_MISMATCH, expr.Rhs, "Der STELLE Operator erwartet eine Zahl als zweiten Operanden, nicht %s", rhs)
		}

		if lhs.IsList {
			t.latestReturnedType = ddptypes.Primitive(lhs.Primitive)
		} else if lhs == ddptypes.String() {
			t.latestReturnedType = ddptypes.Char() // later on the list element type
		}
	case token.DURCH, token.DIVIDIERE, token.TEILE, token.HOCH, token.LOGARITHMUS:
		validate(ddptypes.Int(), ddptypes.Float())
		t.latestReturnedType = ddptypes.Float()
	case token.MODULO:
		validate(ddptypes.Int())
		t.latestReturnedType = ddptypes.Int()
	case token.UND:
		validate(ddptypes.Bool())
		t.latestReturnedType = ddptypes.Bool()
	case token.ODER:
		validate(ddptypes.Bool())
		t.latestReturnedType = ddptypes.Bool()
	case token.LINKS:
		validate(ddptypes.Int())
		t.latestReturnedType = ddptypes.Int()
	case token.RECHTS:
		validate(ddptypes.Int())
		t.latestReturnedType = ddptypes.Int()
	case token.GLEICH:
		if lhs != rhs {
			t.errExpr(ddperror.TYP_TYPE_MISMATCH, expr, "Der '%s' Operator erwartet zwei Operanden gleichen Typs aber hat '%s' und '%s' bekommen", expr.Operator, lhs, rhs)
		}
		t.latestReturnedType = ddptypes.Bool()
	case token.UNGLEICH:
		if lhs != rhs {
			t.errExpr(ddperror.TYP_TYPE_MISMATCH, expr, "Der '%s' Operator erwartet zwei Operanden gleichen Typs aber hat '%s' und '%s' bekommen", expr.Operator, lhs, rhs)
		}
		t.latestReturnedType = ddptypes.Bool()
	case token.GRÖßERODER, token.KLEINER, token.KLEINERODER, token.GRÖßER:
		validate(ddptypes.Int(), ddptypes.Float())
		t.latestReturnedType = ddptypes.Bool()
	case token.LOGISCHODER, token.LOGISCHUND, token.KONTRA:
		validate(ddptypes.Int())
		t.latestReturnedType = ddptypes.Int()
	default:
		panic(fmt.Errorf("unbekannter binärer Operator '%s'", expr.Operator))
	}
}
func (t *Typechecker) VisitTernaryExpr(expr *ast.TernaryExpr) {
	lhs := t.Evaluate(expr.Lhs)
	mid := t.Evaluate(expr.Mid)
	rhs := t.Evaluate(expr.Rhs)

	switch expr.Operator.Type {
	case token.VONBIS:
		if !lhs.IsList && lhs != ddptypes.String() {
			t.errExpr(ddperror.TYP_BAD_INDEXING, expr.Lhs, "Der %s Operator erwartet einen Text oder eine Liste als ersten Operanden, nicht %s", expr.Operator, lhs)
		}

		if !isOfType(mid, ddptypes.Int()) {
			t.errExpected(expr.Operator, expr.Mid, mid, ddptypes.Int())
		}
		if !isOfType(rhs, ddptypes.Int()) {
			t.errExpected(expr.Operator, expr.Rhs, rhs, ddptypes.Int())
		}

		if lhs.IsList {
			t.latestReturnedType = ddptypes.List(lhs.Primitive)
		} else if lhs == ddptypes.String() {
			t.latestReturnedType = ddptypes.String()
		}
	default:
		panic(fmt.Errorf("unbekannter ternärer Operator '%s'", expr.Operator))
	}
}
func (t *Typechecker) VisitCastExpr(expr *ast.CastExpr) {
	lhs := t.Evaluate(expr.Lhs)
	castErr := func() {
		t.errExpr(ddperror.TYP_BAD_CAST, expr, "Ein Ausdruck vom Typ %s kann nicht in den Typ %s umgewandelt werden", lhs, expr.Type)
	}
	if expr.Type.IsList {
		switch expr.Type.Primitive {
		case ddptypes.BUCHSTABE:
			if !isOfType(lhs, ddptypes.Char(), ddptypes.String()) {
				castErr()
			}
		case ddptypes.ZAHL, ddptypes.KOMMAZAHL, ddptypes.BOOLEAN, ddptypes.TEXT:
			if !isOfType(lhs, ddptypes.Primitive(expr.Type.Primitive)) {
				castErr()
			}
		default:
			t.errExpr(ddperror.TYP_BAD_CAST, expr, "Invalide Typumwandlung von %s zu %s", lhs, expr.Type)
		}
	} else {
		switch expr.Type.Primitive {
		case ddptypes.ZAHL:
			if !lhs.IsPrimitive() {
				castErr()
			}
		case ddptypes.KOMMAZAHL:
			if !lhs.IsPrimitive() || !isOfType(lhs, ddptypes.String(), ddptypes.Int(), ddptypes.Float()) {
				castErr()
			}
		case ddptypes.BOOLEAN:
			if !lhs.IsPrimitive() || !isOfType(lhs, ddptypes.Int(), ddptypes.Bool()) {
				castErr()
			}
		case ddptypes.BUCHSTABE:
			if !lhs.IsPrimitive() || !isOfType(lhs, ddptypes.Int(), ddptypes.Char()) {
				castErr()
			}
		case ddptypes.TEXT:
			if lhs.IsList || isOfType(lhs, ddptypes.Void()) {
				castErr()
			}
		default:
			t.errExpr(ddperror.TYP_BAD_CAST, expr, "Invalide Typumwandlung von %s zu %s", lhs, expr.Type)
		}
	}
	t.latestReturnedType = expr.Type
}
func (t *Typechecker) VisitGrouping(expr *ast.Grouping) {
	expr.Expr.Accept(t)
}
func (t *Typechecker) VisitFuncCall(callExpr *ast.FuncCall) {
	for k, expr := range callExpr.Args {
		argType := t.Evaluate(expr)

		var paramType ddptypes.ParameterType
		decl, _ := t.CurrentTable.LookupFunc(callExpr.Name)

		for i, name := range decl.ParamNames {
			if name.Literal == k {
				paramType = decl.ParamTypes[i]
				break
			}
		}

		if ass, ok := expr.(ast.Assigneable); paramType.IsReference && !ok {
			t.errExpr(ddperror.TYP_EXPECTED_REFERENCE, expr, "Es wurde ein Referenz-Typ erwartet aber ein Ausdruck gefunden")
		} else if ass, ok := ass.(*ast.Indexing); paramType.IsReference && paramType.Type == ddptypes.Char() && ok {
			lhs := t.Evaluate(ass.Lhs)
			if lhs.Primitive == ddptypes.TEXT {
				t.errExpr(ddperror.TYP_INVALID_REFERENCE, expr, "Ein Buchstabe in einem Text kann nicht als Buchstaben Referenz übergeben werden")
			}
		}
		if argType != paramType.Type {
			t.errExpr(ddperror.TYP_TYPE_MISMATCH, expr,
				"Die Funktion %s erwartet einen Wert vom Typ %s für den Parameter %s, aber hat %s bekommen",
				callExpr.Name,
				paramType,
				k,
				argType,
			)
		}
	}
	fun, _ := t.CurrentTable.LookupFunc(callExpr.Name)
	t.latestReturnedType = fun.Type
}

func (t *Typechecker) VisitBadStmt(stmt *ast.BadStmt) {
	t.latestReturnedType = ddptypes.Void()
}
func (t *Typechecker) VisitDeclStmt(stmt *ast.DeclStmt) {
	stmt.Decl.Accept(t)
}
func (t *Typechecker) VisitExprStmt(stmt *ast.ExprStmt) {
	stmt.Expr.Accept(t)
}
func (t *Typechecker) VisitAssignStmt(stmt *ast.AssignStmt) {
	rhs := t.Evaluate(stmt.Rhs)
	switch assign := stmt.Var.(type) {
	case *ast.Ident:
		if decl, exists := t.CurrentTable.LookupVar(assign.Literal.Literal); exists && decl.Type != rhs {
			t.errExpr(ddperror.TYP_BAD_ASSIGNEMENT, stmt.Rhs,
				"Ein Wert vom Typ %s kann keiner Variable vom Typ %s zugewiesen werden",
				rhs,
				decl.Type,
			)
		}
	case *ast.Indexing:
		if typ := t.Evaluate(assign.Index); typ != ddptypes.Int() {
			t.errExpr(ddperror.TYP_BAD_INDEXING, assign.Index, "Der STELLE Operator erwartet eine Zahl als zweiten Operanden, nicht %s", typ)
		}

		lhs := t.Evaluate(assign.Lhs)
		if !lhs.IsList && lhs != ddptypes.String() {
			t.errExpr(ddperror.TYP_BAD_INDEXING, assign.Lhs, "Der STELLE Operator erwartet einen Text oder eine Liste als ersten Operanden, nicht %s", lhs)
		}
		if lhs.IsList {
			lhs = ddptypes.Primitive(lhs.Primitive)
		} else if lhs == ddptypes.String() {
			lhs = ddptypes.Char()
		}

		if lhs != rhs {
			t.errExpr(ddperror.TYP_BAD_ASSIGNEMENT, stmt.Rhs,
				"Ein Wert vom Typ %s kann keiner Variable vom Typ %s zugewiesen werden",
				rhs,
				lhs,
			)
		}
	}
}
func (t *Typechecker) VisitBlockStmt(stmt *ast.BlockStmt) {
	t.CurrentTable = stmt.Symbols
	for _, stmt := range stmt.Statements {
		t.visit(stmt)
	}
	t.CurrentTable = t.CurrentTable.Enclosing
}
func (t *Typechecker) VisitIfStmt(stmt *ast.IfStmt) {
	conditionType := t.Evaluate(stmt.Condition)
	if conditionType != ddptypes.Bool() {
		t.errExpr(ddperror.TYP_BAD_CONDITION, stmt.Condition,
			"Die Bedingung einer Wenn-Anweisung muss vom Typ Boolean sein, war aber vom Typ %s",
			conditionType,
		)
	}
	t.visit(stmt.Then)
	if stmt.Else != nil {
		t.visit(stmt.Else)
	}
}
func (t *Typechecker) VisitWhileStmt(stmt *ast.WhileStmt) {
	conditionType := t.Evaluate(stmt.Condition)
	switch stmt.While.Type {
	case token.SOLANGE, token.MACHE:
		if conditionType != ddptypes.Bool() {
			t.errExpr(ddperror.TYP_BAD_CONDITION, stmt.Condition,
				"Die Bedingung einer %s muss vom Typ Boolean sein, war aber vom Typ %s",
				stmt.While.Type,
				conditionType,
			)
		}
	case token.WIEDERHOLE:
		if conditionType != ddptypes.Int() {
			t.errExpr(ddperror.TYP_TYPE_MISMATCH, stmt.Condition,
				"Die Anzahl an Wiederholungen einer WIEDERHOLE Anweisung muss vom Typ ZAHL sein, war aber vom Typ %s",
				conditionType,
			)
		}
	}
	stmt.Body.Accept(t)
}
func (t *Typechecker) VisitForStmt(stmt *ast.ForStmt) {
	t.visit(stmt.Initializer)
	if toType := t.Evaluate(stmt.To); toType != ddptypes.Int() {
		t.errExpr(ddperror.TYP_BAD_FOR, stmt.To,
			"Der Endwert in einer Zählenden-Schleife muss eine Zahl sein, aber war %s",
			toType,
		)
	}
	if stmt.StepSize != nil {
		if stepType := t.Evaluate(stmt.StepSize); stepType != ddptypes.Int() {
			t.errExpr(ddperror.TYP_BAD_FOR, stmt.StepSize,
				"Die Schrittgröße in einer Zählenden-Schleife muss eine Zahl sein, aber war %s",
				stepType,
			)
		}
	}
	stmt.Body.Accept(t)
}
func (t *Typechecker) VisitForRangeStmt(stmt *ast.ForRangeStmt) {
	elementType := stmt.Initializer.Type
	inType := t.Evaluate(stmt.In)

	if !inType.IsList && inType != ddptypes.String() {
		t.errExpr(ddperror.TYP_BAD_FOR, stmt.In, "Man kann nur über Texte oder Listen iterieren")
	}

	if inType.IsList && elementType != ddptypes.Primitive(inType.Primitive) {
		t.err(ddperror.TYP_BAD_FOR, stmt.Initializer.GetRange(),
			fmt.Sprintf("Es wurde eine %s erwartet (Listen-Typ des Iterators), aber ein Ausdruck vom Typ %s gefunden",
				ddptypes.List(elementType.Primitive), inType),
			stmt.Initializer.Token().File,
		)
	} else if inType == ddptypes.String() && elementType != ddptypes.Char() {
		t.err(ddperror.TYP_BAD_FOR, stmt.Initializer.GetRange(),
			fmt.Sprintf("Es wurde ein Ausdruck vom Typ Buchstabe erwartet aber %s gefunden",
				elementType),
			stmt.Initializer.Token().File,
		)
	}
	stmt.Body.Accept(t)
}
func (t *Typechecker) VisitReturnStmt(stmt *ast.ReturnStmt) {
	returnType := ddptypes.Void()
	if stmt.Value != nil {
		returnType = t.Evaluate(stmt.Value)
	}
	if fun, exists := t.CurrentTable.LookupFunc(stmt.Func); exists && fun.Type != returnType {
		t.errExpr(ddperror.TYP_WRONG_RETURN_TYPE, stmt.Value,
			"Eine Funktion mit Rückgabetyp %s kann keinen Wert vom Typ %s zurückgeben",
			fun.Type,
			returnType,
		)
	}
}

// checks if t is contained in types
func isOfType(t ddptypes.Type, types ...ddptypes.Type) bool {
	for _, v := range types {
		if t == v {
			return true
		}
	}
	return false
}
