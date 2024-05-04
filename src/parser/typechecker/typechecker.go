package typechecker

import (
	"fmt"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
	"github.com/DDP-Projekt/Kompilierer/src/token"
)

// holds state to check if the types of an AST are valid
//
// even though it is a visitor, it should not be used seperately from the parser
// all it's VisitX return values are dummy returns
//
// TODO: add a snychronize method like in the parser to prevent unnessecary errors
type Typechecker struct {
	ErrorHandler       ddperror.Handler // function to which errors are passed
	CurrentTable       *ast.SymbolTable // SymbolTable of the current scope (needed for name type-checking)
	latestReturnedType ddptypes.Type    // type of the last visited expression
	Module             *ast.Module      // the module that is being typechecked
	panicMode          *bool            // panic mode synchronized with the parser and resolver
}

func New(Mod *ast.Module, errorHandler ddperror.Handler, panicMode *bool) (*Typechecker, error) {
	if errorHandler == nil {
		errorHandler = ddperror.EmptyHandler
	}
	if panicMode == nil {
		return nil, fmt.Errorf("panicMode must not be nil")
	}
	return &Typechecker{
		ErrorHandler:       errorHandler,
		CurrentTable:       Mod.Ast.Symbols,
		latestReturnedType: ddptypes.VoidType{}, // void signals invalid
		Module:             Mod,
		panicMode:          panicMode,
	}, nil
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
// and doesn't change the Module.Ast.Faulty flag
func (t *Typechecker) EvaluateSilent(expr ast.Expression) ddptypes.Type {
	errHndl, faulty, panicMode := t.ErrorHandler, t.Module.Ast.Faulty, *t.panicMode
	t.ErrorHandler = ddperror.EmptyHandler
	ty := t.Evaluate(expr)
	t.ErrorHandler, t.Module.Ast.Faulty, *t.panicMode = errHndl, faulty, panicMode
	return ty
}

// helper for errors
func (t *Typechecker) err(code ddperror.Code, Range token.Range, msg string) {
	t.Module.Ast.Faulty = true
	if !*t.panicMode {
		*t.panicMode = true
		t.ErrorHandler(ddperror.New(code, Range, msg, t.Module.FileName))
	}
}

// helper to not always pass range and file
func (t *Typechecker) errExpr(code ddperror.Code, expr ast.Expression, msgfmt string, fmtargs ...any) {
	t.err(code, expr.GetRange(), fmt.Sprintf(msgfmt, fmtargs...))
}

// helper for commmon error message
func (t *Typechecker) errExpected(operator ast.Operator, expr ast.Expression, got ddptypes.Type, expected ...ddptypes.Type) {
	msg := fmt.Sprintf("Der %s Operator erwartet einen Ausdruck vom Typ ", operator)
	if len(expected) == 1 {
		msg = fmt.Sprintf("Der %s Operator erwartet einen Ausdruck vom Typ %s", operator, expected[0])
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

func (*Typechecker) Visitor() {}

func (t *Typechecker) VisitBadDecl(decl *ast.BadDecl) ast.VisitResult {
	t.latestReturnedType = ddptypes.VoidType{}
	return ast.VisitRecurse
}

func (t *Typechecker) VisitVarDecl(decl *ast.VarDecl) ast.VisitResult {
	initialType := t.Evaluate(decl.InitVal)
	if initialType != decl.Type {
		msg := fmt.Sprintf("Ein Wert vom Typ %s kann keiner Variable vom Typ %s zugewiesen werden", initialType, decl.Type)
		t.errExpr(ddperror.TYP_BAD_ASSIGNEMENT,
			decl.InitVal,
			msg,
		)
	}

	// TODO: error on the type-name range
	if decl.Public() && !IsPublicType(decl.Type, t.CurrentTable) {
		t.err(ddperror.SEM_BAD_PUBLIC_MODIFIER, decl.NameTok.Range, "Der Typ einer öffentlichen Variable muss ebenfalls öffentlich sein")
	}
	return ast.VisitRecurse
}

func (t *Typechecker) VisitFuncDecl(decl *ast.FuncDecl) ast.VisitResult {
	// already typechecked in blockStmt
	/*if !ast.IsExternFunc(decl) {
		decl.Body.Accept(t)
	}*/

	// TODO: error on the type-name ranges
	if decl.IsPublic {
		if !IsPublicType(decl.Type, t.CurrentTable) {
			t.err(ddperror.SEM_BAD_PUBLIC_MODIFIER, decl.NameTok.Range, "Der Rückgabetyp einer öffentlichen Funktion muss ebenfalls öffentlich sein")
		}

		hasNonPublicType := false
		for _, typ := range decl.ParamTypes {
			if !IsPublicType(typ.Type, t.CurrentTable) {
				hasNonPublicType = true
			}
		}
		if hasNonPublicType {
			t.err(ddperror.SEM_BAD_PUBLIC_MODIFIER, decl.NameTok.Range, "Die Parameter Typen einer öffentlichen Funktion müssen ebenfalls öffentlich sein")
		}
	}
	return ast.VisitRecurse
}

func (t *Typechecker) VisitStructDecl(decl *ast.StructDecl) ast.VisitResult {
	for _, field := range decl.Fields {
		// don't check BadDecls
		if varDecl, isVar := field.(*ast.VarDecl); isVar {
			// check that all public fields also are of public type
			if decl.IsPublic && varDecl.IsPublic && !IsPublicType(varDecl.Type, t.CurrentTable) {
				t.err(ddperror.SEM_BAD_PUBLIC_MODIFIER, varDecl.NameTok.Range, "Wenn eine Struktur öffentlich ist, müssen alle ihre öffentlichen Felder von öffentlichem Typ sein")
			}
		}
		t.visit(field)
	}
	return ast.VisitRecurse
}

func (t *Typechecker) VisitExpressionDecl(decl *ast.ExpressionDecl) ast.VisitResult {
	// Don't do this, as the argument types are set to void in the declaration, and are only filled in once it is called
	// t.Evaluate(decl.Expr)
	return ast.VisitRecurse
}

func (t *Typechecker) VisitBadExpr(expr *ast.BadExpr) ast.VisitResult {
	t.latestReturnedType = ddptypes.VoidType{}
	return ast.VisitRecurse
}

func (t *Typechecker) VisitIdent(expr *ast.Ident) ast.VisitResult {
	decl, ok, isVar := t.CurrentTable.LookupDecl(expr.Literal.Literal)
	if !ok || !isVar {
		t.latestReturnedType = ddptypes.VoidType{}
	} else {
		t.latestReturnedType = decl.(*ast.VarDecl).Type
	}
	return ast.VisitRecurse
}

func (t *Typechecker) VisitIndexing(expr *ast.Indexing) ast.VisitResult {
	if typ := t.Evaluate(expr.Index); typ != ddptypes.ZAHL {
		t.errExpr(ddperror.TYP_BAD_INDEXING, expr.Index, "Der STELLE Operator erwartet eine Zahl als zweiten Operanden, nicht %s", typ)
	}

	lhs := t.Evaluate(expr.Lhs)
	if !ddptypes.IsList(lhs) && lhs != ddptypes.TEXT {
		t.errExpr(ddperror.TYP_BAD_INDEXING, expr.Lhs, "Der STELLE Operator erwartet einen Text oder eine Liste als ersten Operanden, nicht %s", lhs)
	}

	if ddptypes.IsList(lhs) {
		t.latestReturnedType = lhs.(ddptypes.ListType).Underlying
	} else {
		t.latestReturnedType = ddptypes.BUCHSTABE // later on the list element type
	}
	return ast.VisitRecurse
}

func (t *Typechecker) VisitFieldAccess(expr *ast.FieldAccess) ast.VisitResult {
	rhs := t.Evaluate(expr.Rhs)
	if structType, isStruct := rhs.(*ddptypes.StructType); !isStruct {
		t.errExpr(ddperror.TYP_BAD_FIELD_ACCESS, expr.Rhs, "Der VON Operator erwartet eine Struktur als rechten Operanden, nicht %s", rhs)
		t.latestReturnedType = ddptypes.VoidType{}
	} else {
		t.latestReturnedType = t.checkFieldAccess(expr.Field, structType)
	}
	return ast.VisitRecurse
}

func (t *Typechecker) VisitIntLit(expr *ast.IntLit) ast.VisitResult {
	t.latestReturnedType = ddptypes.ZAHL
	return ast.VisitRecurse
}

func (t *Typechecker) VisitFloatLit(expr *ast.FloatLit) ast.VisitResult {
	t.latestReturnedType = ddptypes.KOMMAZAHL
	return ast.VisitRecurse
}

func (t *Typechecker) VisitBoolLit(expr *ast.BoolLit) ast.VisitResult {
	t.latestReturnedType = ddptypes.WAHRHEITSWERT
	return ast.VisitRecurse
}

func (t *Typechecker) VisitCharLit(expr *ast.CharLit) ast.VisitResult {
	t.latestReturnedType = ddptypes.BUCHSTABE
	return ast.VisitRecurse
}

func (t *Typechecker) VisitStringLit(expr *ast.StringLit) ast.VisitResult {
	t.latestReturnedType = ddptypes.TEXT
	return ast.VisitRecurse
}

func (t *Typechecker) VisitListLit(expr *ast.ListLit) ast.VisitResult {
	if expr.Values != nil {
		elementType := t.Evaluate(expr.Values[0])
		for _, v := range expr.Values[1:] {
			if ty := t.Evaluate(v); elementType != ty {
				msg := fmt.Sprintf("Falscher Typ (%s) in Listen Literal vom Typ %s", ty, elementType)
				t.errExpr(ddperror.TYP_BAD_LIST_LITERAL, v, msg)
			}
		}
		expr.Type = ddptypes.ListType{Underlying: elementType}
	} else if expr.Count != nil && expr.Value != nil {
		if count := t.Evaluate(expr.Count); count != ddptypes.ZAHL {
			t.errExpr(ddperror.TYP_BAD_LIST_LITERAL, expr, "Die Größe einer Liste muss als Zahl angegeben werden, nicht als %s", count)
		}
		if val := t.Evaluate(expr.Value); val != expr.Type.Underlying {
			t.errExpr(ddperror.TYP_BAD_LIST_LITERAL, expr, "Falscher Typ (%s) in Listen Literal vom Typ %s", val, expr.Type.Underlying)
		}
	}
	t.latestReturnedType = expr.Type
	return ast.VisitRecurse
}

func (t *Typechecker) VisitUnaryExpr(expr *ast.UnaryExpr) ast.VisitResult {
	// Evaluate the rhs expression and check if the operator fits it
	rhs := t.Evaluate(expr.Rhs)
	switch expr.Operator {
	case ast.UN_ABS, ast.UN_NEGATE:
		if !ddptypes.IsNumeric(rhs) {
			t.errExpected(expr.Operator, expr.Rhs, rhs, ddptypes.ZAHL, ddptypes.KOMMAZAHL)
		}
	case ast.UN_NOT:
		if !isOfType(rhs, ddptypes.WAHRHEITSWERT) {
			t.errExpected(expr.Operator, expr.Rhs, rhs, ddptypes.WAHRHEITSWERT)
		}

		t.latestReturnedType = ddptypes.WAHRHEITSWERT
	case ast.UN_LOGIC_NOT:
		if !isOfType(rhs, ddptypes.ZAHL) {
			t.errExpected(expr.Operator, expr.Rhs, rhs, ddptypes.ZAHL)
		}

		t.latestReturnedType = ddptypes.ZAHL
	case ast.UN_LEN:
		if !ddptypes.IsList(rhs) && rhs != ddptypes.TEXT {
			t.errExpr(ddperror.TYP_TYPE_MISMATCH, expr, "Der %s Operator erwartet einen Text oder eine Liste als Operanden, nicht %s", ast.UN_LEN, rhs)
		}

		t.latestReturnedType = ddptypes.ZAHL
	default:
		panic(fmt.Errorf("unbekannter unärer Operator '%s'", expr.Operator))
	}
	return ast.VisitRecurse
}

func (t *Typechecker) VisitBinaryExpr(expr *ast.BinaryExpr) ast.VisitResult {
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

	switch expr.Operator {
	case ast.BIN_CONCAT:
		if (!ddptypes.IsList(lhs) && !ddptypes.IsList(rhs)) && (lhs == ddptypes.TEXT || rhs == ddptypes.TEXT) { // string, char edge case
			validate(ddptypes.TEXT, ddptypes.BUCHSTABE)
			t.latestReturnedType = ddptypes.TEXT
		} else { // lists
			getOldUnderlyingType := func(t ddptypes.Type) ddptypes.Type {
				if listType, isList := t.(ddptypes.ListType); isList {
					return listType.Underlying
				}
				return t
			}

			if getOldUnderlyingType(lhs) != getOldUnderlyingType(rhs) {
				t.errExpr(ddperror.TYP_TYPE_MISMATCH, expr, "Die Typenkombination aus %s und %s passt nicht zum VERKETTET Operator", lhs, rhs)
			}
			t.latestReturnedType = ddptypes.ListType{Underlying: getOldUnderlyingType(lhs)}
		}
	case ast.BIN_PLUS, ast.BIN_MINUS, ast.BIN_MULT:
		validate(ddptypes.ZAHL, ddptypes.KOMMAZAHL)

		if lhs == ddptypes.ZAHL && rhs == ddptypes.ZAHL {
			t.latestReturnedType = ddptypes.ZAHL
		} else {
			t.latestReturnedType = ddptypes.KOMMAZAHL
		}
	case ast.BIN_INDEX:
		if !ddptypes.IsList(lhs) && lhs != ddptypes.TEXT {
			t.errExpr(ddperror.TYP_TYPE_MISMATCH, expr.Lhs, "Der STELLE Operator erwartet einen Text oder eine Liste als ersten Operanden, nicht %s", lhs)
		}
		if rhs != ddptypes.ZAHL {
			t.errExpr(ddperror.TYP_TYPE_MISMATCH, expr.Rhs, "Der STELLE Operator erwartet eine Zahl als zweiten Operanden, nicht %s", rhs)
		}

		if ddptypes.IsList(lhs) {
			t.latestReturnedType = lhs.(ddptypes.ListType).Underlying
		} else if lhs == ddptypes.TEXT {
			t.latestReturnedType = ddptypes.BUCHSTABE // later on the list element type
		}
	case ast.BIN_SLICE_FROM, ast.BIN_SLICE_TO:
		if !ddptypes.IsList(lhs) && lhs != ddptypes.TEXT {
			t.errExpr(ddperror.TYP_BAD_INDEXING, expr.Lhs, "Der '%s' Operator erwartet einen Text oder eine Liste als ersten Operanden, nicht %s", expr.Operator, lhs)
		}
		if !isOfType(rhs, ddptypes.ZAHL) {
			t.errExpected(expr.Operator, expr.Rhs, rhs, ddptypes.ZAHL)
		}

		if ddptypes.IsList(lhs) {
			t.latestReturnedType = lhs
		} else if lhs == ddptypes.TEXT {
			t.latestReturnedType = ddptypes.TEXT
		}
	case ast.BIN_FIELD_ACCESS:
		if ident, isIdent := expr.Lhs.(*ast.Ident); isIdent {
			if structType, isStruct := rhs.(*ddptypes.StructType); !isStruct {
				// error was already reported by the resolver
				t.latestReturnedType = ddptypes.VoidType{}
			} else {
				t.latestReturnedType = t.checkFieldAccess(ident, structType)
			}
		} else {
			t.latestReturnedType = ddptypes.VoidType{}
		}
	case ast.BIN_DIV, ast.BIN_POW, ast.BIN_LOG:
		validate(ddptypes.ZAHL, ddptypes.KOMMAZAHL)
		t.latestReturnedType = ddptypes.KOMMAZAHL
	case ast.BIN_MOD:
		validate(ddptypes.ZAHL)
		t.latestReturnedType = ddptypes.ZAHL
	case ast.BIN_AND, ast.BIN_OR:
		validate(ddptypes.WAHRHEITSWERT)
		t.latestReturnedType = ddptypes.WAHRHEITSWERT
	case ast.BIN_LEFT_SHIFT, ast.BIN_RIGHT_SHIFT:
		validate(ddptypes.ZAHL)
		t.latestReturnedType = ddptypes.ZAHL
	case ast.BIN_EQUAL, ast.BIN_UNEQUAL:
		if lhs != rhs {
			t.errExpr(ddperror.TYP_TYPE_MISMATCH, expr, "Der '%s' Operator erwartet zwei Operanden gleichen Typs aber hat '%s' und '%s' bekommen", expr.Operator, lhs, rhs)
		}
		t.latestReturnedType = ddptypes.WAHRHEITSWERT
	case ast.BIN_GREATER, ast.BIN_LESS, ast.BIN_GREATER_EQ, ast.BIN_LESS_EQ:
		validate(ddptypes.ZAHL, ddptypes.KOMMAZAHL)
		t.latestReturnedType = ddptypes.WAHRHEITSWERT
	case ast.BIN_LOGIC_AND, ast.BIN_LOGIC_OR, ast.BIN_LOGIC_XOR:
		validate(ddptypes.ZAHL)
		t.latestReturnedType = ddptypes.ZAHL
	default:
		panic(fmt.Errorf("unbekannter binärer Operator '%s'", expr.Operator))
	}
	return ast.VisitRecurse
}

func (t *Typechecker) VisitTernaryExpr(expr *ast.TernaryExpr) ast.VisitResult {
	lhs := t.Evaluate(expr.Lhs)
	mid := t.Evaluate(expr.Mid)
	rhs := t.Evaluate(expr.Rhs)

	switch expr.Operator {
	case ast.TER_SLICE:
		if !ddptypes.IsList(lhs) && lhs != ddptypes.TEXT {
			t.errExpr(ddperror.TYP_BAD_INDEXING, expr.Lhs, "Der %s Operator erwartet einen Text oder eine Liste als ersten Operanden, nicht %s", expr.Operator, lhs)
		}

		if !isOfType(mid, ddptypes.ZAHL) {
			t.errExpected(expr.Operator, expr.Mid, mid, ddptypes.ZAHL)
		}
		if !isOfType(rhs, ddptypes.ZAHL) {
			t.errExpected(expr.Operator, expr.Rhs, rhs, ddptypes.ZAHL)
		}

		if ddptypes.IsList(lhs) {
			t.latestReturnedType = lhs
		} else if lhs == ddptypes.TEXT {
			t.latestReturnedType = ddptypes.TEXT
		}
	case ast.TER_BETWEEN:
		if !isOfType(lhs, ddptypes.ZAHL, ddptypes.KOMMAZAHL) {
			t.errExpected(expr.Operator, expr.Lhs, lhs, ddptypes.ZAHL, ddptypes.KOMMAZAHL)
		}
		if !isOfType(mid, ddptypes.ZAHL, ddptypes.KOMMAZAHL) {
			t.errExpected(expr.Operator, expr.Mid, mid, ddptypes.ZAHL, ddptypes.KOMMAZAHL)
		}
		if !isOfType(rhs, ddptypes.ZAHL, ddptypes.KOMMAZAHL) {
			t.errExpected(expr.Operator, expr.Rhs, rhs, ddptypes.ZAHL, ddptypes.KOMMAZAHL)
		}
		t.latestReturnedType = ddptypes.WAHRHEITSWERT
	default:
		panic(fmt.Errorf("unbekannter ternärer Operator '%s'", expr.Operator))
	}
	return ast.VisitRecurse
}

func (t *Typechecker) VisitCastExpr(expr *ast.CastExpr) ast.VisitResult {
	lhs := t.Evaluate(expr.Lhs)
	castErr := func() {
		t.errExpr(ddperror.TYP_BAD_CAST, expr, "Ein Ausdruck vom Typ %s kann nicht in den Typ %s umgewandelt werden", lhs, expr.Type)
	}
	if exprType, ok := expr.Type.(ddptypes.ListType); ok {
		switch exprType.Underlying {
		case ddptypes.BUCHSTABE:
			if !isOfType(lhs, ddptypes.BUCHSTABE, ddptypes.TEXT) {
				castErr()
			}
		case ddptypes.ZAHL, ddptypes.KOMMAZAHL, ddptypes.WAHRHEITSWERT, ddptypes.TEXT:
			if !isOfType(lhs, exprType.Underlying) {
				castErr()
			}
		default:
			t.errExpr(ddperror.TYP_BAD_CAST, expr, "Invalide Typumwandlung von %s zu %s", lhs, expr.Type)
		}
	} else if exprType, ok := expr.Type.(ddptypes.PrimitiveType); ok {
		switch exprType {
		case ddptypes.ZAHL:
			if !ddptypes.IsPrimitive(lhs) {
				castErr()
			}
		case ddptypes.KOMMAZAHL:
			if !ddptypes.IsPrimitive(lhs) || !isOfType(lhs, ddptypes.TEXT, ddptypes.ZAHL, ddptypes.KOMMAZAHL) {
				castErr()
			}
		case ddptypes.WAHRHEITSWERT:
			if !ddptypes.IsPrimitive(lhs) || !isOfType(lhs, ddptypes.ZAHL, ddptypes.WAHRHEITSWERT) {
				castErr()
			}
		case ddptypes.BUCHSTABE:
			if !ddptypes.IsPrimitive(lhs) || !isOfType(lhs, ddptypes.ZAHL, ddptypes.BUCHSTABE) {
				castErr()
			}
		case ddptypes.TEXT:
			if ddptypes.IsList(lhs) || isOfType(lhs, ddptypes.VoidType{}) {
				castErr()
			}
		default:
			t.errExpr(ddperror.TYP_BAD_CAST, expr, "Invalide Typumwandlung von %s zu %s", lhs, expr.Type)
		}
	} else if exprType, ok := expr.Type.(*ddptypes.StructType); ok {
		t.errExpr(ddperror.TYP_BAD_CAST, expr, "Invalide Typumwandlung von %s zu %s", lhs, exprType)
	} else {
		t.errExpr(ddperror.TYP_BAD_CAST, expr, "Invalide Typumwandlung von %s zu %s", lhs, expr.Type)
	}
	t.latestReturnedType = expr.Type
	return ast.VisitRecurse
}

func (t *Typechecker) VisitTypeOpExpr(expr *ast.TypeOpExpr) ast.VisitResult {
	switch expr.Operator {
	case ast.TYPE_SIZE:
		t.latestReturnedType = ddptypes.ZAHL
	case ast.TYPE_DEFAULT:
		t.latestReturnedType = expr.Rhs
	default:
		panic(fmt.Errorf("unbekannter Typ-Operator '%s'", expr.Operator))
	}
	return ast.VisitRecurse
}

func (t *Typechecker) VisitGrouping(expr *ast.Grouping) ast.VisitResult {
	expr.Expr.Accept(t)
	return ast.VisitRecurse
}

func (t *Typechecker) VisitFuncCall(callExpr *ast.FuncCall) ast.VisitResult {
	symbol, _, _ := t.CurrentTable.LookupDecl(callExpr.Name)
	decl := symbol.(*ast.FuncDecl)

	for k, expr := range callExpr.Args {
		argType := t.Evaluate(expr)

		var paramType ddptypes.ParameterType

		for i, name := range decl.ParamNames {
			if name.Literal == k {
				paramType = decl.ParamTypes[i]
				break
			}
		}

		if ass, ok := expr.(ast.Assigneable); paramType.IsReference && !ok {
			t.errExpr(ddperror.TYP_EXPECTED_REFERENCE, expr, "Es wurde ein Referenz-Typ erwartet aber ein Ausdruck gefunden")
		} else if ass, ok := ass.(*ast.Indexing); paramType.IsReference && paramType.Type == ddptypes.BUCHSTABE && ok {
			lhs := t.Evaluate(ass.Lhs)
			if lhs == ddptypes.TEXT {
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

	t.latestReturnedType = decl.Type
	return ast.VisitRecurse
}

func (t *Typechecker) VisitStructLiteral(expr *ast.StructLiteral) ast.VisitResult {
	for argName, arg := range expr.Args {
		argType := t.Evaluate(arg)

		var paramType ddptypes.Type
		for _, field := range expr.Struct.Type.Fields {
			if field.Name == argName {
				paramType = field.Type
				break
			}
		}

		if argType != paramType {
			t.errExpr(ddperror.TYP_TYPE_MISMATCH, arg,
				"Die Struktur %s erwartet einen Wert vom Typ %s für das Feld %s, aber hat %s bekommen",
				expr.Struct.Name(),
				paramType,
				argName,
				argType,
			)
		}
	}

	t.latestReturnedType = expr.Struct.Type
	return ast.VisitRecurse
}

func (t *Typechecker) VisitExpressionCall(expr *ast.ExpressionCall) ast.VisitResult {
	lastTable := t.CurrentTable
	t.CurrentTable = expr.FilledSymbols
	errHndl := t.ErrorHandler
	t.ErrorHandler = func(err ddperror.Error) {
		err.File = t.Module.FileName
		err.Range = expr.GetRange()
		err.Msg = fmt.Sprintf("Im Ausdruck %s[%s]: %s", expr.Decl.Name(), expr.Decl.Mod.FileName, err.Msg)
		errHndl(err)
	}

	if expr.Expr != nil {
		t.latestReturnedType = t.Evaluate(expr.Expr)
	} else {
		t.latestReturnedType = t.Evaluate(expr.Decl.Expr)
	}

	t.ErrorHandler = errHndl
	t.CurrentTable = lastTable
	return ast.VisitRecurse
}

func (t *Typechecker) VisitBadStmt(stmt *ast.BadStmt) ast.VisitResult {
	t.latestReturnedType = ddptypes.VoidType{}
	return ast.VisitRecurse
}

func (t *Typechecker) VisitDeclStmt(stmt *ast.DeclStmt) ast.VisitResult {
	stmt.Decl.Accept(t)
	return ast.VisitRecurse
}

func (t *Typechecker) VisitExprStmt(stmt *ast.ExprStmt) ast.VisitResult {
	stmt.Expr.Accept(t)
	return ast.VisitRecurse
}

func (t *Typechecker) VisitImportStmt(stmt *ast.ImportStmt) ast.VisitResult {
	return ast.VisitRecurse
}

func (t *Typechecker) VisitAssignStmt(stmt *ast.AssignStmt) ast.VisitResult {
	rhs := t.Evaluate(stmt.Rhs)
	var lhs ddptypes.Type

	switch assign := stmt.Var.(type) {
	case *ast.Ident:
		if decl, exists, isVar := t.CurrentTable.LookupDecl(assign.Literal.Literal); exists && isVar && decl.(*ast.VarDecl).Type != rhs {
			t.errExpr(ddperror.TYP_BAD_ASSIGNEMENT, stmt.Rhs,
				"Ein Wert vom Typ %s kann keiner Variable vom Typ %s zugewiesen werden",
				rhs,
				decl.(*ast.VarDecl).Type,
			)
		} else if exists && isVar {
			lhs = decl.(*ast.VarDecl).Type
		} else {
			lhs = ddptypes.VoidType{}
		}
	case *ast.Indexing:
		lhs = t.Evaluate(assign)
	case *ast.FieldAccess:
		lhs = t.Evaluate(assign)
	}

	if lhs != rhs {
		t.errExpr(ddperror.TYP_BAD_ASSIGNEMENT, stmt.Rhs,
			"Ein Wert vom Typ %s kann keiner Variable vom Typ %s zugewiesen werden",
			rhs,
			lhs,
		)
	}
	return ast.VisitRecurse
}

func (t *Typechecker) VisitBlockStmt(stmt *ast.BlockStmt) ast.VisitResult {
	t.CurrentTable = stmt.Symbols
	for _, stmt := range stmt.Statements {
		t.visit(stmt)
	}
	t.CurrentTable = t.CurrentTable.Enclosing
	return ast.VisitRecurse
}

func (t *Typechecker) VisitIfStmt(stmt *ast.IfStmt) ast.VisitResult {
	conditionType := t.Evaluate(stmt.Condition)
	if conditionType != ddptypes.WAHRHEITSWERT {
		t.errExpr(ddperror.TYP_BAD_CONDITION, stmt.Condition,
			"Die Bedingung einer Wenn-Anweisung muss ein Wahrheitswert sein, war aber vom Typ %s",
			conditionType,
		)
	}
	t.visit(stmt.Then)
	if stmt.Else != nil {
		t.visit(stmt.Else)
	}
	return ast.VisitRecurse
}

func (t *Typechecker) VisitWhileStmt(stmt *ast.WhileStmt) ast.VisitResult {
	conditionType := t.Evaluate(stmt.Condition)
	switch stmt.While.Type {
	case token.SOLANGE, token.MACHE:
		if conditionType != ddptypes.WAHRHEITSWERT {
			t.errExpr(ddperror.TYP_BAD_CONDITION, stmt.Condition,
				"Die Bedingung einer %s muss ein Wahrheitswert sein, war aber vom Typ %s",
				stmt.While.Type,
				conditionType,
			)
		}
	case token.WIEDERHOLE:
		if conditionType != ddptypes.ZAHL {
			t.errExpr(ddperror.TYP_TYPE_MISMATCH, stmt.Condition,
				"Die Anzahl an Wiederholungen einer WIEDERHOLE Anweisung muss vom Typ ZAHL sein, war aber vom Typ %s",
				conditionType,
			)
		}
	}
	stmt.Body.Accept(t)
	return ast.VisitRecurse
}

func (t *Typechecker) VisitForStmt(stmt *ast.ForStmt) ast.VisitResult {
	t.visit(stmt.Initializer)
	iter_type := stmt.Initializer.Type
	if !ddptypes.IsNumeric(iter_type) {
		t.err(ddperror.TYP_BAD_FOR, stmt.Initializer.GetRange(), "Der Zähler in einer zählenden-Schleife muss eine Zahl oder Kommazahl sein")
	}
	if toType := t.Evaluate(stmt.To); toType != iter_type {
		t.errExpr(ddperror.TYP_BAD_FOR, stmt.To,
			"Der Endwert in einer Zählenden-Schleife muss vom selben Typ wie der Zähler (%s) sein, aber war %s",
			iter_type,
			toType,
		)
	}
	if stmt.StepSize != nil {
		if stepType := t.Evaluate(stmt.StepSize); stepType != iter_type {
			t.errExpr(ddperror.TYP_BAD_FOR, stmt.StepSize,
				"Die Schrittgröße in einer Zählenden-Schleife muss vom selben Typ wie der Zähler (%s) sein, aber war %s",
				iter_type,
				stepType,
			)
		}
	}
	stmt.Body.Accept(t)
	return ast.VisitRecurse
}

func (t *Typechecker) VisitForRangeStmt(stmt *ast.ForRangeStmt) ast.VisitResult {
	elementType := stmt.Initializer.Type
	inType := t.Evaluate(stmt.In)

	if !ddptypes.IsList(inType) && inType != ddptypes.TEXT {
		t.errExpr(ddperror.TYP_BAD_FOR, stmt.In, "Man kann nur über Texte oder Listen iterieren")
	}

	if inTypeList, isList := inType.(ddptypes.ListType); isList && elementType != inTypeList.Underlying {
		t.err(ddperror.TYP_BAD_FOR, stmt.Initializer.GetRange(),
			fmt.Sprintf("Es wurde eine %s erwartet (Listen-Typ des Iterators), aber ein Ausdruck vom Typ %s gefunden",
				elementType, inTypeList),
		)
	} else if inType == ddptypes.TEXT && elementType != ddptypes.BUCHSTABE {
		t.err(ddperror.TYP_BAD_FOR, stmt.Initializer.GetRange(),
			fmt.Sprintf("Es wurde ein Ausdruck vom Typ Buchstabe erwartet aber %s gefunden",
				elementType),
		)
	}
	stmt.Body.Accept(t)
	return ast.VisitRecurse
}

func (t *Typechecker) VisitBreakContinueStmt(stmt *ast.BreakContinueStmt) ast.VisitResult {
	return ast.VisitRecurse
}

func (t *Typechecker) VisitReturnStmt(stmt *ast.ReturnStmt) ast.VisitResult {
	var returnType ddptypes.Type = ddptypes.VoidType{}
	if stmt.Value != nil {
		returnType = t.Evaluate(stmt.Value)
	}
	if fun, exists, _ := t.CurrentTable.LookupDecl(stmt.Func); exists && fun.(*ast.FuncDecl).Type != returnType {
		if stmt.Value == nil {
			t.err(ddperror.TYP_WRONG_RETURN_TYPE, stmt.Range,
				fmt.Sprintf("Eine Funktion mit Rückgabetyp %s kann keinen Wert vom Typ %s zurückgeben",
					fun.(*ast.FuncDecl).Type,
					returnType),
			)
		} else {
			t.errExpr(ddperror.TYP_WRONG_RETURN_TYPE, stmt.Value,
				"Eine Funktion mit Rückgabetyp %s kann keinen Wert vom Typ %s zurückgeben",
				fun.(*ast.FuncDecl).Type,
				returnType,
			)
		}
	}
	return ast.VisitRecurse
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

// helper for
func (t *Typechecker) checkFieldAccess(Lhs *ast.Ident, structType *ddptypes.StructType) ddptypes.Type {
	var fieldType ddptypes.Type = ddptypes.VoidType{}

	for _, field := range structType.Fields {
		if field.Name == Lhs.Literal.Literal {
			fieldType = field.Type
			break
		}
	}

	if fieldType == ddptypes.Type(ddptypes.VoidType{}) {
		article := "Ein"
		switch structType.Gender() {
		case ddptypes.FEMININ:
			article = "Eine"
		}
		t.errExpr(ddperror.TYP_BAD_FIELD_ACCESS, Lhs, "%s %s hat kein Feld mit Name %s", article, structType.Name, Lhs.Literal.Literal)
		return ddptypes.VoidType{}
	}

	// if the type was imported, check for public/private fields
	if structDecl, exists, _ := t.CurrentTable.LookupDecl(structType.Name); exists {
		structDecl := structDecl.(*ast.StructDecl)
		if structDecl.Mod != t.Module {
			for _, field := range structDecl.Fields {
				if field.Name() == Lhs.Literal.Literal {
					if field, ok := field.(*ast.VarDecl); ok && !field.IsPublic {
						t.errExpr(ddperror.TYP_PRIVATE_FIELD_ACCESS, Lhs, "Das Feld %s der Struktur %s ist nicht öffentlich", Lhs.Literal.Literal, structType.Name)
					}
					break
				}
			}
		}
	}

	return fieldType
}

// reports wether the given type from this module of the given table is public
// should only be called from the global scope
// and with the SymbolTable that was in use when the type was declared
func IsPublicType(typ ddptypes.Type, table *ast.SymbolTable) bool {
	// a list-type is public if its underlying type is public
	typ = ddptypes.GetNestedUnderlying(typ)

	// a struct type is public if a corresponding struct-decl is public or if it was imported from another module
	if structTyp, isStruct := typ.(*ddptypes.StructType); isStruct {
		// get the corresponding decl from the current scope
		// because it contains imported types as well
		decl, _, _ := table.LookupDecl(structTyp.Name)
		if structDecl, isStruct := decl.(*ast.StructDecl); isStruct {
			// if the decl is from the current module check if it is public
			// if it is from another module, it has to be public
			return structDecl.IsPublic
		}
		return false // the corresponding name was not a struct decl
	}

	return true // non-struct types are predeclared and always "public"
}
