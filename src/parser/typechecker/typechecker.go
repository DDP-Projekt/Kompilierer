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

func New(Mod *ast.Module, errorHandler ddperror.Handler, file string, panicMode *bool) *Typechecker {
	if errorHandler == nil {
		errorHandler = ddperror.EmptyHandler
	}
	if panicMode == nil {
		panic(fmt.Errorf("panicMode must not be nil"))
	}
	return &Typechecker{
		ErrorHandler:       errorHandler,
		CurrentTable:       Mod.Ast.Symbols,
		latestReturnedType: ddptypes.VoidType{}, // void signals invalid
		Module:             Mod,
		panicMode:          panicMode,
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
		t.ErrorHandler(ddperror.New(code, ddperror.LEVEL_ERROR, Range, msg, t.Module.FileName))
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
	if !ddptypes.Equal(initialType, decl.Type) && (!ddptypes.Equal(decl.Type, ddptypes.VARIABLE) || ddptypes.Equal(initialType, ddptypes.VoidType{})) {
		msg := fmt.Sprintf("Ein Wert vom Typ %s kann keiner Variable vom Typ %s zugewiesen werden", initialType, decl.Type)
		t.errExpr(ddperror.TYP_BAD_ASSIGNEMENT,
			decl.InitVal,
			msg,
		)
	}

	if decl.Public() && !IsPublicType(decl.Type, t.CurrentTable) {
		t.err(ddperror.SEM_BAD_PUBLIC_MODIFIER, decl.TypeRange, "Der Typ einer öffentlichen Variable muss ebenfalls öffentlich sein")
	}
	return ast.VisitRecurse
}

func (t *Typechecker) VisitFuncDecl(decl *ast.FuncDecl) ast.VisitResult {
	// already typechecked in blockStmt
	/*if !ast.IsExternFunc(decl) {
		decl.Body.Accept(t)
	}*/

	if decl.IsPublic {
		if !IsPublicType(decl.ReturnType, t.CurrentTable) {
			t.err(ddperror.SEM_BAD_PUBLIC_MODIFIER, decl.ReturnTypeRange, "Der Rückgabetyp einer öffentlichen Funktion muss ebenfalls öffentlich sein")
		}

		for _, param := range decl.Parameters {
			if !IsPublicType(param.Type.Type, t.CurrentTable) {
				t.err(ddperror.SEM_BAD_PUBLIC_MODIFIER, param.TypeRange, "Die Parameter Typen einer öffentlichen Funktion müssen ebenfalls öffentlich sein")
			}
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

func (t *Typechecker) VisitTypeAliasDecl(decl *ast.TypeAliasDecl) ast.VisitResult {
	if decl.IsPublic && !IsPublicType(decl.Underlying, t.CurrentTable) {
		t.err(ddperror.SEM_BAD_PUBLIC_MODIFIER, decl.NameTok.Range, "Der unterliegende Typ eines öffentlichen Typ-Aliases muss ebenfalls öffentlich sein")
	}

	return ast.VisitRecurse
}

func (t *Typechecker) VisitTypeDefDecl(decl *ast.TypeDefDecl) ast.VisitResult {
	if decl.IsPublic && !IsPublicType(decl.Underlying, t.CurrentTable) {
		t.err(ddperror.SEM_BAD_PUBLIC_MODIFIER, decl.NameTok.Range, "Der unterliegende Typ eines öffentlichen Typ-Aliases muss ebenfalls öffentlich sein")
	}

	return ast.VisitRecurse
}

func (t *Typechecker) VisitBadExpr(expr *ast.BadExpr) ast.VisitResult {
	t.latestReturnedType = ddptypes.VoidType{}
	return ast.VisitRecurse
}

func (t *Typechecker) VisitIdent(expr *ast.Ident) ast.VisitResult {
	decl, ok, isVar := t.CurrentTable.LookupDecl(expr.Literal.Literal)
	if !ok || !isVar || decl == nil {
		t.latestReturnedType = ddptypes.VoidType{}
	} else {
		t.latestReturnedType = decl.(*ast.VarDecl).Type
	}
	return ast.VisitRecurse
}

func (t *Typechecker) VisitIndexing(expr *ast.Indexing) ast.VisitResult {
	if typ := t.Evaluate(expr.Index); !ddptypes.Equal(typ, ddptypes.ZAHL) {
		t.errExpr(ddperror.TYP_BAD_INDEXING, expr.Index, "Der STELLE Operator erwartet eine Zahl als zweiten Operanden, nicht %s", typ)
	}

	lhs := t.Evaluate(expr.Lhs)
	if !ddptypes.IsList(lhs) && !ddptypes.Equal(lhs, ddptypes.TEXT) {
		t.errExpr(ddperror.TYP_BAD_INDEXING, expr.Lhs, "Der STELLE Operator erwartet einen Text oder eine Liste als ersten Operanden, nicht %s", lhs)
	}

	if ddptypes.IsList(lhs) {
		t.latestReturnedType = ddptypes.GetListUnderlying(lhs)
	} else {
		t.latestReturnedType = ddptypes.BUCHSTABE // later on the list element type
	}
	return ast.VisitRecurse
}

func (t *Typechecker) VisitFieldAccess(expr *ast.FieldAccess) ast.VisitResult {
	rhs := t.Evaluate(expr.Rhs)
	if !ddptypes.IsStruct(rhs) {
		t.errExpr(ddperror.TYP_BAD_FIELD_ACCESS, expr.Rhs, "Der VON Operator erwartet eine Struktur als rechten Operanden, nicht %s", rhs)
		t.latestReturnedType = ddptypes.VoidType{}
	} else {
		t.latestReturnedType = t.checkFieldAccess(expr.Field, rhs)
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
			if ty := t.Evaluate(v); !ddptypes.Equal(elementType, ty) {
				msg := fmt.Sprintf("Falscher Typ (%s) in Listen Literal vom Typ %s", ty, elementType)
				t.errExpr(ddperror.TYP_BAD_LIST_LITERAL, v, msg)
			}
		}
		expr.Type = ddptypes.ListType{Underlying: elementType}
	} else if expr.Count != nil && expr.Value != nil {
		if count := t.Evaluate(expr.Count); !ddptypes.Equal(count, ddptypes.ZAHL) {
			t.errExpr(ddperror.TYP_BAD_LIST_LITERAL, expr, "Die Größe einer Liste muss als Zahl angegeben werden, nicht als %s", count)
		}
		if val := t.Evaluate(expr.Value); !ddptypes.Equal(val, expr.Type.Underlying) {
			t.errExpr(ddperror.TYP_BAD_LIST_LITERAL, expr, "Falscher Typ (%s) in Listen Literal vom Typ %s", val, expr.Type.Underlying)
		}
	}
	t.latestReturnedType = expr.Type
	return ast.VisitRecurse
}

func (t *Typechecker) VisitUnaryExpr(expr *ast.UnaryExpr) ast.VisitResult {
	// Evaluate the rhs expression and check if the operator fits it
	rhs := t.Evaluate(expr.Rhs)

	if overload := t.findOverload(expr.Operator, rhs); overload != nil {
		expr.OverloadedBy = overload
		t.latestReturnedType = overload.ReturnType
		return ast.VisitRecurse
	}

	switch expr.Operator {
	case ast.UN_ABS, ast.UN_NEGATE:
		if !ddptypes.IsNumeric(rhs) {
			t.errExpected(expr.Operator, expr.Rhs, rhs, ddptypes.ZAHL, ddptypes.KOMMAZAHL)
		}
	case ast.UN_NOT:
		if !isOneOf(rhs, ddptypes.WAHRHEITSWERT) {
			t.errExpected(expr.Operator, expr.Rhs, rhs, ddptypes.WAHRHEITSWERT)
		}

		t.latestReturnedType = ddptypes.WAHRHEITSWERT
	case ast.UN_LOGIC_NOT:
		if !isOneOf(rhs, ddptypes.ZAHL) {
			t.errExpected(expr.Operator, expr.Rhs, rhs, ddptypes.ZAHL)
		}

		t.latestReturnedType = ddptypes.ZAHL
	case ast.UN_LEN:
		if !ddptypes.IsList(rhs) && !ddptypes.Equal(rhs, ddptypes.TEXT) {
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

	if overload := t.findOverload(expr.Operator, lhs, rhs); overload != nil {
		expr.OverloadedBy = overload
		t.latestReturnedType = overload.ReturnType
		return ast.VisitRecurse
	}

	// helper to validate if types match
	validate := func(valid ...ddptypes.Type) {
		if !isOneOf(lhs, valid...) {
			t.errExpected(expr.Operator, expr.Lhs, lhs, valid...)
		}
		if !isOneOf(rhs, valid...) {
			t.errExpected(expr.Operator, expr.Rhs, rhs, valid...)
		}
	}

	switch expr.Operator {
	case ast.BIN_CONCAT:
		if (!ddptypes.IsList(lhs) && !ddptypes.IsList(rhs)) && (ddptypes.Equal(lhs, ddptypes.TEXT) || ddptypes.Equal(rhs, ddptypes.TEXT)) { // string, char edge case
			validate(ddptypes.TEXT, ddptypes.BUCHSTABE)
			t.latestReturnedType = ddptypes.TEXT
		} else { // lists
			if !ddptypes.Equal(ddptypes.GetListUnderlying(lhs), ddptypes.GetListUnderlying(rhs)) {
				t.errExpr(ddperror.TYP_TYPE_MISMATCH, expr, "Die Typenkombination aus %s und %s passt nicht zum VERKETTET Operator", lhs, rhs)
			}
			t.latestReturnedType = ddptypes.ListType{Underlying: ddptypes.GetListUnderlying(lhs)}
		}
	case ast.BIN_PLUS, ast.BIN_MINUS, ast.BIN_MULT:
		validate(ddptypes.ZAHL, ddptypes.KOMMAZAHL)

		if ddptypes.Equal(lhs, ddptypes.ZAHL) && ddptypes.Equal(rhs, ddptypes.ZAHL) {
			t.latestReturnedType = ddptypes.ZAHL
		} else {
			t.latestReturnedType = ddptypes.KOMMAZAHL
		}
	case ast.BIN_INDEX:
		if !ddptypes.IsList(lhs) && !ddptypes.Equal(lhs, ddptypes.TEXT) {
			t.errExpr(ddperror.TYP_TYPE_MISMATCH, expr.Lhs, "Der STELLE Operator erwartet einen Text oder eine Liste als ersten Operanden, nicht %s", lhs)
		}
		if !ddptypes.Equal(rhs, ddptypes.ZAHL) {
			t.errExpr(ddperror.TYP_TYPE_MISMATCH, expr.Rhs, "Der STELLE Operator erwartet eine Zahl als zweiten Operanden, nicht %s", rhs)
		}

		if listType, isList := ddptypes.CastList(lhs); isList {
			t.latestReturnedType = listType.Underlying
		} else if ddptypes.Equal(lhs, ddptypes.TEXT) {
			t.latestReturnedType = ddptypes.BUCHSTABE // later on the list element type
		}
	case ast.BIN_SLICE_FROM, ast.BIN_SLICE_TO:
		if !ddptypes.IsList(lhs) && !ddptypes.Equal(lhs, ddptypes.TEXT) {
			t.errExpr(ddperror.TYP_BAD_INDEXING, expr.Lhs, "Der '%s' Operator erwartet einen Text oder eine Liste als ersten Operanden, nicht %s", expr.Operator, lhs)
		}
		if !isOneOf(rhs, ddptypes.ZAHL) {
			t.errExpected(expr.Operator, expr.Rhs, rhs, ddptypes.ZAHL)
		}

		if ddptypes.IsList(lhs) {
			t.latestReturnedType = lhs
		} else if ddptypes.Equal(lhs, ddptypes.TEXT) {
			t.latestReturnedType = ddptypes.TEXT
		}
	case ast.BIN_FIELD_ACCESS:
		if ident, isIdent := expr.Lhs.(*ast.Ident); isIdent {
			if !ddptypes.IsStruct(rhs) {
				// error was already reported by the resolver
				t.latestReturnedType = ddptypes.VoidType{}
			} else {
				t.latestReturnedType = t.checkFieldAccess(ident, rhs)
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
	case ast.BIN_AND, ast.BIN_OR, ast.BIN_XOR:
		validate(ddptypes.WAHRHEITSWERT)
		t.latestReturnedType = ddptypes.WAHRHEITSWERT
	case ast.BIN_LEFT_SHIFT, ast.BIN_RIGHT_SHIFT:
		validate(ddptypes.ZAHL)
		t.latestReturnedType = ddptypes.ZAHL
	case ast.BIN_EQUAL, ast.BIN_UNEQUAL:
		if !ddptypes.Equal(lhs, rhs) {
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

	if overload := t.findOverload(expr.Operator, lhs, mid, rhs); overload != nil {
		expr.OverloadedBy = overload
		t.latestReturnedType = overload.ReturnType
		return ast.VisitRecurse
	}

	switch expr.Operator {
	case ast.TER_SLICE:
		if !ddptypes.IsList(lhs) && !ddptypes.Equal(lhs, ddptypes.TEXT) {
			t.errExpr(ddperror.TYP_BAD_INDEXING, expr.Lhs, "Der %s Operator erwartet einen Text oder eine Liste als ersten Operanden, nicht %s", expr.Operator, lhs)
		}

		if !isOneOf(mid, ddptypes.ZAHL) {
			t.errExpected(expr.Operator, expr.Mid, mid, ddptypes.ZAHL)
		}
		if !isOneOf(rhs, ddptypes.ZAHL) {
			t.errExpected(expr.Operator, expr.Rhs, rhs, ddptypes.ZAHL)
		}

		if ddptypes.IsList(lhs) {
			t.latestReturnedType = lhs
		} else if ddptypes.Equal(lhs, ddptypes.TEXT) {
			t.latestReturnedType = ddptypes.TEXT
		}
	case ast.TER_BETWEEN:
		if !isOneOf(lhs, ddptypes.ZAHL, ddptypes.KOMMAZAHL) {
			t.errExpected(expr.Operator, expr.Lhs, lhs, ddptypes.ZAHL, ddptypes.KOMMAZAHL)
		}
		if !isOneOf(mid, ddptypes.ZAHL, ddptypes.KOMMAZAHL) {
			t.errExpected(expr.Operator, expr.Mid, mid, ddptypes.ZAHL, ddptypes.KOMMAZAHL)
		}
		if !isOneOf(rhs, ddptypes.ZAHL, ddptypes.KOMMAZAHL) {
			t.errExpected(expr.Operator, expr.Rhs, rhs, ddptypes.ZAHL, ddptypes.KOMMAZAHL)
		}
		t.latestReturnedType = ddptypes.WAHRHEITSWERT
	case ast.TER_FALLS:
		if !ddptypes.Equal(lhs, rhs) {
			t.errExpr(ddperror.TYP_TYPE_MISMATCH, expr, "Die linke und rechte Seite des 'falls' Ausdrucks müssen den selben Typ haben, aber es wurde %s und %s gefunden", lhs, rhs)
		}
		if !isOneOf(mid, ddptypes.WAHRHEITSWERT) {
			t.errExpr(ddperror.TYP_TYPE_MISMATCH, expr, "Die Bedingung des 'falls' Ausdrucks muss vom Typ %s sein, aber es wurde %s gefunden", ddptypes.WAHRHEITSWERT, mid)
		}
		t.latestReturnedType = lhs
	default:
		panic(fmt.Errorf("unbekannter ternärer Operator '%s'", expr.Operator))
	}
	return ast.VisitRecurse
}

func (t *Typechecker) VisitCastExpr(expr *ast.CastExpr) ast.VisitResult {
	lhs := t.Evaluate(expr.Lhs)
	castErr := func() {
		t.errExpr(ddperror.TYP_BAD_CAST, expr, "Ein Ausdruck vom Typ %s kann nicht in den Typ %s umgewandelt werden", lhs, expr.TargetType)
	}

	overloads := t.Module.Operators[ast.CAST_OP]
	if len(overloads) > 0 {
		for _, overload := range overloads {
			if ddptypes.Equal(overload.Parameters[0].Type.Type, lhs) && ddptypes.Equal(overload.ReturnType, expr.TargetType) {
				expr.OverloadedBy = overload
				t.latestReturnedType = expr.TargetType
				return ast.VisitRecurse
			}
		}
	}

	targetTypeDef, isTargetTypeDef := ddptypes.CastTypeDef(expr.TargetType)
	lhsTypeDef, isLhsTypeDef := ddptypes.CastTypeDef(lhs)

	if ddptypes.IsAny(lhs) || (ddptypes.IsAny(expr.TargetType) && !ddptypes.IsVoid(lhs)) {
		// casts from/to any are always valid but might error at runtime
		t.latestReturnedType = expr.TargetType
		return ast.VisitRecurse
	} else if isTargetTypeDef && isLhsTypeDef {
		// typedefs can only be converted to/from their underlying type
		if !ddptypes.Equal(lhsTypeDef.Underlying, expr.TargetType) && !ddptypes.Equal(targetTypeDef.Underlying, lhs) {
			castErr()
		}
	} else if isTargetTypeDef {
		// typedefs can only be converted to/from their underlying type
		if !ddptypes.Equal(lhs, targetTypeDef.Underlying) {
			castErr()
		}
	} else if isLhsTypeDef {
		// typedefs can only be converted to/from their underlying type
		if !ddptypes.Equal(expr.TargetType, lhsTypeDef.Underlying) {
			castErr()
		}
	} else if ddptypes.IsList(expr.TargetType) { // non-list types can be converted to their list-type with a single element
		underlying := ddptypes.GetUnderlying(ddptypes.GetListUnderlying(expr.TargetType))
		if !isOneOf(lhs, underlying) {
			castErr()
		}
	} else if primitiveType, isPrimitive := ddptypes.CastPrimitive(expr.TargetType); isPrimitive {
		// special rules for primitive conversions
		switch primitiveType {
		case ddptypes.ZAHL:
			if !ddptypes.IsPrimitive(lhs) {
				castErr()
			}
		case ddptypes.KOMMAZAHL:
			if !ddptypes.IsPrimitive(lhs) || !isOneOf(lhs, ddptypes.TEXT, ddptypes.ZAHL, ddptypes.KOMMAZAHL) {
				castErr()
			}
		case ddptypes.WAHRHEITSWERT:
			if !ddptypes.IsPrimitive(lhs) || !isOneOf(lhs, ddptypes.ZAHL, ddptypes.WAHRHEITSWERT) {
				castErr()
			}
		case ddptypes.BUCHSTABE:
			if !ddptypes.IsPrimitive(lhs) || !isOneOf(lhs, ddptypes.ZAHL, ddptypes.BUCHSTABE) {
				castErr()
			}
		case ddptypes.TEXT:
			if ddptypes.IsList(lhs) || isOneOf(lhs, ddptypes.VoidType{}) {
				castErr()
			}
		default:
			castErr()
		}
	} else {
		castErr()
	}
	t.latestReturnedType = expr.TargetType
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

func (t *Typechecker) VisitTypeCheck(expr *ast.TypeCheck) ast.VisitResult {
	lhs := t.Evaluate(expr.Lhs)
	if !ddptypes.Equal(lhs, ddptypes.VARIABLE) {
		t.errExpr(ddperror.TYP_TYPE_MISMATCH, expr.Lhs,
			"Der '%s' Operator erwartet einen Ausdruck vom Typ '%s' aber hat '%s' bekommen",
			expr.Tok.Literal,
			ddptypes.VARIABLE,
			lhs,
		)
	}
	if ddptypes.Equal(expr.CheckType, ddptypes.VARIABLE) {
		t.errExpr(ddperror.TYP_TYPE_MISMATCH, expr, "Dieser Ausdruck ist immer 'wahr'")
	}
	t.latestReturnedType = ddptypes.WAHRHEITSWERT
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

		for _, param := range decl.Parameters {
			if param.Name.Literal == k {
				paramType = param.Type
				break
			}
		}

		if ass, ok := expr.(ast.Assigneable); paramType.IsReference && !ok {
			t.errExpr(ddperror.TYP_EXPECTED_REFERENCE, expr, "Es wurde ein Referenz-Typ erwartet aber ein Ausdruck gefunden")
		} else if ass, ok := ass.(*ast.Indexing); paramType.IsReference && ddptypes.Equal(paramType.Type, ddptypes.BUCHSTABE) && ok {
			lhs := t.Evaluate(ass.Lhs)
			if ddptypes.Equal(lhs, ddptypes.TEXT) {
				t.errExpr(ddperror.TYP_INVALID_REFERENCE, expr, "Ein Buchstabe in einem Text kann nicht als Buchstaben Referenz übergeben werden")
			}
		}
		if !ddptypes.Equal(argType, paramType.Type) {
			t.errExpr(ddperror.TYP_TYPE_MISMATCH, expr,
				"Die Funktion %s erwartet einen Wert vom Typ %s für den Parameter %s, aber hat %s bekommen",
				callExpr.Name,
				paramType,
				k,
				argType,
			)
		}
	}

	t.latestReturnedType = decl.ReturnType
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

		if !ddptypes.Equal(argType, paramType) {
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
	target := t.Evaluate(stmt.Var)

	if !ddptypes.Equal(target, rhs) && (!ddptypes.Equal(target, ddptypes.VARIABLE) || ddptypes.Equal(rhs, ddptypes.VoidType{})) {
		t.errExpr(ddperror.TYP_BAD_ASSIGNEMENT, stmt.Rhs,
			"Ein Wert vom Typ %s kann keiner Variable vom Typ %s zugewiesen werden",
			rhs,
			target,
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
	if !ddptypes.Equal(conditionType, ddptypes.WAHRHEITSWERT) {
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
		if !ddptypes.Equal(conditionType, ddptypes.WAHRHEITSWERT) {
			t.errExpr(ddperror.TYP_BAD_CONDITION, stmt.Condition,
				"Die Bedingung einer %s muss ein Wahrheitswert sein, war aber vom Typ %s",
				stmt.While.Type,
				conditionType,
			)
		}
	case token.WIEDERHOLE:
		if !ddptypes.Equal(conditionType, ddptypes.ZAHL) {
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
	if toType := t.Evaluate(stmt.To); !ddptypes.Equal(toType, iter_type) {
		t.errExpr(ddperror.TYP_BAD_FOR, stmt.To,
			"Der Endwert in einer Zählenden-Schleife muss vom selben Typ wie der Zähler (%s) sein, aber war %s",
			iter_type,
			toType,
		)
	}
	if stmt.StepSize != nil {
		if stepType := t.Evaluate(stmt.StepSize); !ddptypes.Equal(stepType, iter_type) {
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

	if !ddptypes.IsList(inType) && !ddptypes.Equal(inType, ddptypes.TEXT) {
		t.errExpr(ddperror.TYP_BAD_FOR, stmt.In, "Man kann nur über Texte oder Listen iterieren")
	}

	if inTypeList, isList := ddptypes.CastList(inType); isList && !ddptypes.Equal(elementType, inTypeList.Underlying) {
		t.err(ddperror.TYP_BAD_FOR, stmt.Initializer.GetRange(),
			fmt.Sprintf("Es wurde eine %s erwartet (Listen-Typ des Iterators), aber ein Ausdruck vom Typ %s gefunden",
				elementType, inTypeList),
		)
	} else if ddptypes.Equal(inType, ddptypes.TEXT) && !ddptypes.Equal(elementType, ddptypes.BUCHSTABE) {
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
	if fun, exists, _ := t.CurrentTable.LookupDecl(stmt.Func); exists && !ddptypes.Equal(fun.(*ast.FuncDecl).ReturnType, returnType) &&
		(!ddptypes.Equal(fun.(*ast.FuncDecl).ReturnType, ddptypes.VARIABLE) || ddptypes.Equal(returnType, ddptypes.VoidType{})) {
		errRange := stmt.Range
		if stmt.Value != nil {
			errRange = stmt.Value.GetRange()
		}

		t.err(ddperror.TYP_WRONG_RETURN_TYPE, errRange,
			fmt.Sprintf("Eine Funktion mit Rückgabetyp %s kann keinen Wert vom Typ %s zurückgeben",
				fun.(*ast.FuncDecl).ReturnType,
				returnType),
		)
	}
	return ast.VisitRecurse
}

func (*Typechecker) VisitTodoStmt(*ast.TodoStmt) ast.VisitResult {
	return ast.VisitRecurse
}

// checks if t is contained in types
func isOneOf(t ddptypes.Type, types ...ddptypes.Type) bool {
	for _, v := range types {
		if ddptypes.Equal(t, v) {
			return true
		}
	}
	return false
}

// helper for field access
// panics if originalType is not a struct type
func (t *Typechecker) checkFieldAccess(Lhs *ast.Ident, originalType ddptypes.Type) ddptypes.Type {
	structType, ok := ddptypes.GetUnderlying(originalType).(*ddptypes.StructType)
	if !ok {
		panic(fmt.Sprintf("non struct type (%s) passed to checkFieldAccess", originalType))
	}

	var fieldType ddptypes.Type = ddptypes.VoidType{}

	for _, field := range structType.Fields {
		if field.Name == Lhs.Literal.Literal {
			fieldType = field.Type
			break
		}
	}

	if ddptypes.Equal(fieldType, ddptypes.VoidType{}) {
		article := "Ein"
		switch structType.Gender() {
		case ddptypes.FEMININ:
			article = "Eine"
		}
		t.errExpr(ddperror.TYP_BAD_FIELD_ACCESS, Lhs, "%s %s hat kein Feld mit Name %s", article, originalType.String(), Lhs.Literal.Literal)
		return ddptypes.VoidType{}
	}

	// if the type was imported, check for public/private fields
	if structDecl, exists, _ := t.CurrentTable.LookupDecl(structType.Name); exists {
		structDecl := structDecl.(*ast.StructDecl)
		if structDecl.Mod != t.Module {
			for _, field := range structDecl.Fields {
				if field.Name() == Lhs.Literal.Literal {
					if field, ok := field.(*ast.VarDecl); ok && !field.IsPublic {
						t.errExpr(ddperror.TYP_PRIVATE_FIELD_ACCESS, Lhs, "Das Feld %s der Struktur %s ist nicht öffentlich", Lhs.Literal.Literal, originalType.String())
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
	typ = ddptypes.GetNestedListUnderlying(typ)

	// a struct type is public if a corresponding struct-decl is public or if it was imported from another module
	if ddptypes.IsTypeAlias(typ) || ddptypes.IsStruct(typ) {
		// get the corresponding decl from the current scope
		// because it contains imported types as well
		decl, _, _ := table.LookupDecl(typ.String())
		return decl.Public()
	}

	return true // non-struct types are predeclared and always "public"
}

func (t *Typechecker) findOverload(operator ast.Operator, operands ...ddptypes.Type) *ast.FuncDecl {
	overloads := t.Module.Operators[operator]
	if len(overloads) > 0 {
	overload_loop:
		for _, overload := range overloads {
			for i, operand := range operands {
				if !ddptypes.Equal(overload.Parameters[i].Type.Type, operand) {
					continue overload_loop
				}
			}
			return overload
		}
	}
	return nil
}
