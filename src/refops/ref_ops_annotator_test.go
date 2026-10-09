package refops

import (
	"testing"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// collects the nodes of a parsed module that the tests look at
type collectedNodes struct {
	ast.BaseVisitor
	vars    map[string]*ast.VarDecl
	assigns []*ast.AssignStmt
	returns []*ast.ReturnStmt
	exprs   []*ast.ExprStmt
}

func (n *collectedNodes) VisitVarDecl(d *ast.VarDecl) ast.VisitResult {
	n.vars[d.Name()] = d
	return ast.VisitRecurse
}

func (n *collectedNodes) VisitAssignStmt(s *ast.AssignStmt) ast.VisitResult {
	n.assigns = append(n.assigns, s)
	return ast.VisitRecurse
}

func (n *collectedNodes) VisitReturnStmt(s *ast.ReturnStmt) ast.VisitResult {
	n.returns = append(n.returns, s)
	return ast.VisitRecurse
}

func (n *collectedNodes) VisitExprStmt(s *ast.ExprStmt) ast.VisitResult {
	n.exprs = append(n.exprs, s)
	return ast.VisitRecurse
}

func parseAnnotated(t *testing.T, src string) *collectedNodes {
	t.Helper()

	collector := ddperror.Collector{}
	module, err := parser.Parse(parser.Options{
		FileName:     "test.ddp",
		Source:       []byte(src),
		ErrorHandler: collector.GetHandler(),
		Annotators: []ast.Annotator{
			&ImplicitRefCastAnnotator{},
			&RefOpsAnnotator{},
		},
	})
	require.NoError(t, err)
	require.False(t, collector.DidError(), "parser errors: %v", collector.Errors)

	checkInvariants(t, module)

	nodes := &collectedNodes{vars: make(map[string]*ast.VarDecl)}
	ast.VisitModuleRec(module, nodes)
	return nodes
}

// checks that every expression of the module (except field names) has a consistent RefOpsMeta
func checkInvariants(t *testing.T, module *ast.Module) {
	t.Helper()

	fieldNames := map[ast.Expression]struct{}{}
	var exprs []ast.Expression
	ast.VisitModuleRec(module, &ast.BaseVisitor{VisitCondition: func(n ast.Node) bool {
		if bin, ok := n.(*ast.BinaryExpr); ok && bin.Operator == ast.BIN_FIELD_ACCESS {
			fieldNames[bin.Lhs] = struct{}{}
		}
		if e, ok := n.(ast.Expression); ok {
			exprs = append(exprs, e)
		}
		return true
	}})

	for _, e := range exprs {
		m, ok := GetRefOps(e)
		if _, isFieldName := fieldNames[e]; isFieldName {
			assert.False(t, ok, "field name %s has a RefOpsMeta", e.Token())
			continue
		}
		if !assert.True(t, ok, "expression %s (%s) has no RefOpsMeta", e, e.Token()) {
			continue
		}

		assert.LessOrEqual(t, m.Derefs, m.BaseLevel, "%s: %s", e.Token(), m)
		assert.False(t, m.Derefs > 0 && m.Promotions > 0, "%s: %s", e.Token(), m)
		expected := m
		expected.updateLevel()
		assert.Equal(t, expected.Level, m.Level, "%s: %s", e.Token(), m)
	}
}

func (n *collectedNodes) initVal(t *testing.T, name string) ast.Expression {
	t.Helper()
	decl, ok := n.vars[name]
	require.True(t, ok, "variable %s not found", name)
	require.NotNil(t, decl.InitVal, "variable %s has no initial value", name)
	return decl.InitVal
}

// returns the i-th assignment of the module
func (n *collectedNodes) assign(t *testing.T, i int) *ast.AssignStmt {
	t.Helper()
	require.Greater(t, len(n.assigns), i, "not enough assignments")
	return n.assigns[i]
}

func requireRefOps(t *testing.T, expr ast.Expression) RefOpsMeta {
	t.Helper()
	m, ok := GetRefOps(expr)
	require.True(t, ok, "expression %s has no RefOpsMeta", expr)
	return m
}

func assertRefOps(t *testing.T, expected RefOpsMeta, expr ast.Expression) {
	t.Helper()
	assert.Equal(t, expected, requireRefOps(t, expr))
}

func assertAssignTarget(t *testing.T, expected AssignKind, expectedRoot *ast.VarDecl, stmt *ast.AssignStmt) {
	t.Helper()
	m, ok := GetAssignTarget(stmt)
	require.True(t, ok, "assignment has no AssignTargetMeta")
	assert.Equal(t, expected, m.Target)
	assert.Equal(t, expectedRoot, m.Root)
}

func TestAddressOfLocalInVarDecl(t *testing.T) {
	nodes := parseAnnotated(t, `
Die Funktion f gibt nichts zurück, macht:
	Die Zahl x ist 1.
	Die Zahlen Referenz r ist x.
	Die Zahl y ist r.
Und kann so benutzt werden:
	"f"
`)

	// address of x
	assertRefOps(t, RefOpsMeta{Op: OpAddr, BaseLevel: 1, Level: 1}, nodes.initVal(t, "r"))
	// the reference is implicitly dereferenced
	assertRefOps(t, RefOpsMeta{Op: OpValue, BaseLevel: 1, Derefs: 1, Level: 0}, nodes.initVal(t, "y"))
	// plain values
	assertRefOps(t, RefOpsMeta{Op: OpValue}, nodes.initVal(t, "x"))
}

func TestPromotionInVarDecl(t *testing.T) {
	nodes := parseAnnotated(t, `
Die Funktion f gibt nichts zurück, macht:
	Die Zahlen Referenz r ist 5.
	Die Text Referenz t ist "Hallo" verkettet mit " Welt".
Und kann so benutzt werden:
	"f"
`)

	assertRefOps(t, RefOpsMeta{Op: OpValue, Promotions: 1, Level: 1}, nodes.initVal(t, "r"))
	assertRefOps(t, RefOpsMeta{Op: OpValue, Promotions: 1, Level: 1}, nodes.initVal(t, "t"))
}

func TestReturn(t *testing.T) {
	nodes := parseAnnotated(t, `
Die Funktion addr gibt eine Text Referenz zurück, macht:
	Der Text t ist "Hallo".
	Gib t zurück.
Und kann so benutzt werden:
	"addr"

Die Funktion promote gibt eine Zahlen Referenz zurück, macht:
	Gib 5 zurück.
Und kann so benutzt werden:
	"promote"

Die Funktion as_ref mit dem Parameter x vom Typ Zahlen Referenz, gibt eine Zahlen Referenz zurück, macht:
	Gib x zurück.
Und kann so benutzt werden:
	"ref <x>"

Die Funktion deref mit dem Parameter x vom Typ Zahlen Referenz, gibt eine Zahl zurück, macht:
	Gib x zurück.
Und kann so benutzt werden:
	"deref <x>"
`)

	require.Len(t, nodes.returns, 4)
	// address of the local t
	assertRefOps(t, RefOpsMeta{Op: OpAddr, BaseLevel: 1, Level: 1}, nodes.returns[0].Value)
	// 5 is copied into a new cell
	assertRefOps(t, RefOpsMeta{Op: OpValue, Promotions: 1, Level: 1}, nodes.returns[1].Value)
	// the reference parameter itself is returned
	assertRefOps(t, RefOpsMeta{Op: OpValue, BaseLevel: 1, Level: 1}, nodes.returns[2].Value)
	// the reference parameter is dereferenced
	assertRefOps(t, RefOpsMeta{Op: OpValue, BaseLevel: 1, Derefs: 1, Level: 0}, nodes.returns[3].Value)
}

func TestFuncCallArguments(t *testing.T) {
	nodes := parseAnnotated(t, `
Die Funktion inc mit dem Parameter a vom Typ Zahlen Referenz, gibt nichts zurück, macht:
	Speichere a plus 1 in a.
Und kann so benutzt werden:
	"Inkrementiere <a>"

Die Funktion val mit dem Parameter a vom Typ Zahl, gibt nichts zurück, macht:
	Speichere a plus 1 in a.
Und kann so benutzt werden:
	"Benutze <a>"

Die Funktion f gibt nichts zurück, macht:
	Die Zahl x ist 1.
	Die Zahlen Referenz r ist x.
	Inkrementiere x.
	Inkrementiere r.
	Benutze r.
	Benutze x.
Und kann so benutzt werden:
	"f"
`)

	require.Len(t, nodes.exprs, 4)
	argOf := func(i int) ast.Expression {
		call, ok := nodes.exprs[i].Expr.(*ast.FuncCall)
		require.True(t, ok)
		return call.Args["a"]
	}

	// call expressions in ExprStmts are visited directly
	assertRefOps(t, RefOpsMeta{Op: OpValue}, nodes.exprs[0].Expr)

	// address of x
	assertRefOps(t, RefOpsMeta{Op: OpAddr, BaseLevel: 1, Level: 1}, argOf(0))
	// the reference r is passed on
	assertRefOps(t, RefOpsMeta{Op: OpValue, BaseLevel: 1, Level: 1}, argOf(1))
	// r is dereferenced for a value parameter
	assertRefOps(t, RefOpsMeta{Op: OpValue, BaseLevel: 1, Derefs: 1, Level: 0}, argOf(2))
	// x is passed by value
	assertRefOps(t, RefOpsMeta{Op: OpValue}, argOf(3))
}

func TestAssignTargets(t *testing.T) {
	nodes := parseAnnotated(t, `
Die Zahl g ist 0.
Speichere 1 in g.

Die Funktion f gibt nichts zurück, macht:
	Die Zahl x ist 1.
	Die Zahl y ist 2.
	Die Zahlen Referenz r ist x.
	Die Zahlen Referenz r2 ist y.
	Speichere 3 in x.
	Speichere 4 in r.
	Speichere r2 in r.
	Speichere 5 in der Wert von r.
	Der Text t ist "abc".
	Speichere 'x' in t an der Stelle 1.
	Die Zahlen Liste l ist eine Liste, die aus 1, 2, 3 besteht.
	Speichere 6 in l an der Stelle 1.
	Die Zahlen Listen Referenz lr ist l.
	Speichere 7 in lr an der Stelle 2.
Und kann so benutzt werden:
	"f"
`)

	require.Len(t, nodes.assigns, 8)

	// global
	assertAssignTarget(t, AssignStoreIntoGlobal, nodes.vars["g"], nodes.assign(t, 0))
	assertRefOps(t, RefOpsMeta{Op: OpAddr, BaseLevel: 1, Level: 1}, nodes.assign(t, 0).Var)

	// local
	assertAssignTarget(t, AssignStoreIntoVar, nodes.vars["x"], nodes.assign(t, 1))

	// through the reference r
	assertAssignTarget(t, AssignStoreThrough, nil, nodes.assign(t, 2))
	assertRefOps(t, RefOpsMeta{Op: OpValue, BaseLevel: 1, Level: 1}, nodes.assign(t, 2).Var)

	// rebinding r
	assertAssignTarget(t, AssignRebind, nodes.vars["r"], nodes.assign(t, 3))
	assertRefOps(t, RefOpsMeta{Op: OpAddr, BaseLevel: 2, Level: 2}, nodes.assign(t, 3).Var)
	assertRefOps(t, RefOpsMeta{Op: OpValue, BaseLevel: 1, Level: 1}, nodes.assign(t, 3).Rhs)

	// explicit dereference
	assertAssignTarget(t, AssignStoreThrough, nil, nodes.assign(t, 4))
	assertRefOps(t, RefOpsMeta{Op: OpAddr, BaseLevel: 1, Level: 1}, nodes.assign(t, 4).Var)

	// string indexing
	assertAssignTarget(t, AssignStringIndexing, nil, nodes.assign(t, 5))

	// list element of a local list
	assertAssignTarget(t, AssignStoreIntoList, nil, nodes.assign(t, 6))
	assertRefOps(t, RefOpsMeta{Op: OpAddr, BaseLevel: 1, Level: 1}, nodes.assign(t, 6).Var)

	// list element through a list reference
	assertAssignTarget(t, AssignStoreIntoList, nil, nodes.assign(t, 7))
	assertRefOps(t, RefOpsMeta{Op: OpAddr, BaseLevel: 1, Level: 1}, nodes.assign(t, 7).Var)
}

const structSrc = `
Wir nennen die Kombination aus
	der Zahl a mit Standardwert 0,
	der Zahlen Referenz ref mit Standardwert 1,
einen Punkt, und erstellen sie so:
	"ein Punkt" oder
	"ein Punkt mit ref <ref>"
`

func TestStructFields(t *testing.T) {
	nodes := parseAnnotated(t, structSrc+`
Die Funktion f gibt nichts zurück, macht:
	Der Punkt p ist ein Punkt.
	Speichere 1 in a von p.
	Die Punkt Referenz pr ist p.
	Speichere 2 in a von pr.
	Die Zahl z ist 3.
	Der Punkt p2 ist ein Punkt mit ref z.
	Die Zahl w ist ref von p2.
Und kann so benutzt werden:
	"f"
`)

	// the default value of the reference field is promoted
	assertRefOps(t, RefOpsMeta{Op: OpValue, Promotions: 1, Level: 1}, nodes.initVal(t, "ref"))

	// into a field of the local p
	stmt := nodes.assign(t, 0)
	assertAssignTarget(t, AssignStoreIntoVar, nodes.vars["p"], stmt)
	assertRefOps(t, RefOpsMeta{Op: OpAddr, BaseLevel: 1, Level: 1}, stmt.Var)
	assertRefOps(t, RefOpsMeta{Op: OpAddr, BaseLevel: 1, Level: 1}, stmt.Var.(*ast.BinaryExpr).Rhs)
	// the field name is not evaluated
	_, hasMeta := GetRefOps(stmt.Var.(*ast.BinaryExpr).Lhs)
	assert.False(t, hasMeta)

	// into a field through the reference pr
	stmt = nodes.assign(t, 1)
	assertAssignTarget(t, AssignStoreThrough, nil, stmt)
	assertRefOps(t, RefOpsMeta{Op: OpAddr, BaseLevel: 1, Level: 1}, stmt.Var)
	assertRefOps(t, RefOpsMeta{Op: OpValue, BaseLevel: 1, Level: 1}, stmt.Var.(*ast.BinaryExpr).Rhs)

	literal, ok := nodes.initVal(t, "p2").(*ast.StructLiteral)
	require.True(t, ok)
	assertRefOps(t, RefOpsMeta{Op: OpAddr, BaseLevel: 1, Level: 1}, literal.Args["ref"])

	// reading the reference field dereferences it
	assertRefOps(t, RefOpsMeta{Op: OpValue, BaseLevel: 1, Derefs: 1, Level: 0}, nodes.initVal(t, "w"))
}

func TestCasts(t *testing.T) {
	nodes := parseAnnotated(t, `
Die Funktion f gibt nichts zurück, macht:
	Die Zahl x ist 1.
	Die Zahlen Referenz r ist x als Zahlen Referenz.
	Die Zahl y ist r als Zahl.
	Die Kommazahl k ist r als Kommazahl.
	Die Variable v ist x.
	Die Zahl z ist v als Zahl.
Und kann so benutzt werden:
	"f"
`)

	// a non-assignable cast to a reference copies the value
	assertRefOps(t, RefOpsMeta{Op: OpPromote, BaseLevel: 1, Level: 1}, nodes.initVal(t, "r"))
	// casting a reference to a value dereferences it
	assertRefOps(t, RefOpsMeta{Op: OpDeref, BaseLevel: 0, Level: 0}, nodes.initVal(t, "y"))
	assertRefOps(t, RefOpsMeta{Op: OpDeref, BaseLevel: 0, Level: 0}, nodes.initVal(t, "k"))
	// implicit cast to any
	assertRefOps(t, RefOpsMeta{Op: OpValue, BoxAny: true, Level: 0}, nodes.initVal(t, "v"))
	assertRefOps(t, RefOpsMeta{Op: OpValue}, nodes.initVal(t, "z"))
}

func TestAnyReferences(t *testing.T) {
	nodes := parseAnnotated(t, `
Die Funktion f gibt nichts zurück, macht:
	Die Variablen Referenz vr ist 5.
	Die Variable v ist 1.
	Die Variablen Referenz vr2 ist v.
Und kann so benutzt werden:
	"f"
`)

	// 5 is boxed and then promoted (compiler.VisitVarDecl)
	assertRefOps(t, RefOpsMeta{Op: OpValue, BoxAny: true, BoxPromotions: 1, Level: 1}, nodes.initVal(t, "vr"))
	// TODO:
	// the address of the any v is boxed into a new any, which is then promoted.
	// This mirrors the current code generator (compiler.VisitVarDecl compares the IR type of the
	// initial value, a reference, with ddpany) and is most likely a bug there; if it is fixed,
	// the expectation becomes RefOpsMeta{Op: OpAddr, BaseLevel: 1, Level: 1}
	assertRefOps(t, RefOpsMeta{Op: OpAddr, BaseLevel: 1, BoxAny: true, BoxPromotions: 1, Level: 1}, nodes.initVal(t, "vr2"))
}

func TestGrouping(t *testing.T) {
	nodes := parseAnnotated(t, `
Die Funktion inc mit dem Parameter a vom Typ Zahlen Referenz, gibt nichts zurück, macht:
	Speichere a plus 1 in a.
Und kann so benutzt werden:
	"Inkrementiere <a>"

Die Funktion f gibt nichts zurück, macht:
	Die Zahl x ist 1.
	Die Zahlen Referenz r ist x.
	Die Zahl y ist (r) plus 1.
	Inkrementiere (x).
Und kann so benutzt werden:
	"f"
`)

	// the grouping is dereferenced, the inner expression is untouched
	binary, ok := nodes.initVal(t, "y").(*ast.BinaryExpr)
	require.True(t, ok)
	grouping, ok := binary.Lhs.(*ast.Grouping)
	require.True(t, ok)
	assertRefOps(t, RefOpsMeta{Op: OpValue, BaseLevel: 1, Derefs: 1, Level: 0}, grouping)
	assertRefOps(t, RefOpsMeta{Op: OpValue, BaseLevel: 1, Level: 1}, grouping.Expr)

	// the address of the grouped x is passed
	call, ok := nodes.exprs[0].Expr.(*ast.FuncCall)
	require.True(t, ok)
	grouping, ok = call.Args["a"].(*ast.Grouping)
	require.True(t, ok)
	assertRefOps(t, RefOpsMeta{Op: OpAddr, BaseLevel: 1, Level: 1}, grouping)
	assertRefOps(t, RefOpsMeta{Op: OpAddr, BaseLevel: 1, Level: 1}, grouping.Expr)
}

func TestListElements(t *testing.T) {
	nodes := parseAnnotated(t, `
Die Funktion inc mit dem Parameter a vom Typ Zahlen Referenz, gibt nichts zurück, macht:
	Speichere a plus 1 in a.
Und kann so benutzt werden:
	"Inkrementiere <a>"

Die Funktion f gibt nichts zurück, macht:
	Die Zahlen Liste l ist eine Liste, die aus 1, 2, 3 besteht.
	Inkrementiere (l an der Stelle 1).
	Die Zahl z ist l an der Stelle 2.
Und kann so benutzt werden:
	"f"
`)

	call, ok := nodes.exprs[0].Expr.(*ast.FuncCall)
	require.True(t, ok)
	grouping, ok := call.Args["a"].(*ast.Grouping)
	require.True(t, ok)
	// address of the list element, computed from the address of l
	index := grouping.Expr.(*ast.BinaryExpr)
	assertRefOps(t, RefOpsMeta{Op: OpAddr, BaseLevel: 1, Level: 1}, index)
	assertRefOps(t, RefOpsMeta{Op: OpAddr, BaseLevel: 1, Level: 1}, index.Lhs)
	assertRefOps(t, RefOpsMeta{Op: OpValue}, index.Rhs)

	// reading an element
	assertRefOps(t, RefOpsMeta{Op: OpValue}, nodes.initVal(t, "z"))
	assertRefOps(t, RefOpsMeta{Op: OpValue}, nodes.initVal(t, "z").(*ast.BinaryExpr).Lhs)
}

func TestLoops(t *testing.T) {
	nodes := parseAnnotated(t, `
Die Funktion f gibt nichts zurück, macht:
	Die Zahlen Liste l ist eine Liste, die aus 1, 2, 3 besteht.
	Die Zahlen Listen Referenz lr ist l.
	Die Zahl summe ist 0.
	Für jede Zahl z in lr, mache:
		Speichere summe plus z in summe.
	Die Zahlen Referenz grenze ist summe.
	Für jede Zahl i von 1 bis grenze, mache:
		Speichere summe plus i in summe.
	Solange summe kleiner als grenze ist, mache:
		Speichere summe plus 1 in summe.
Und kann so benutzt werden:
	"f"
`)

	// the for-range variable is not treated like a normal declaration,
	// its initial value is the annotated In expression
	assertRefOps(t, RefOpsMeta{Op: OpValue, BaseLevel: 1, Level: 1}, nodes.initVal(t, "z"))

	// the counter is a normal declaration
	assertRefOps(t, RefOpsMeta{Op: OpValue}, nodes.initVal(t, "i"))
	for _, stmt := range nodes.assigns {
		assertAssignTarget(t, AssignStoreIntoVar, nodes.vars["summe"], stmt)
	}
}

func TestGenericInstantiations(t *testing.T) {
	nodes := parseAnnotated(t, `
Die generische Funktion id mit dem Parameter x vom Typ T Referenz, gibt eine T Referenz zurück, macht:
	Gib x zurück.
Und kann so benutzt werden:
	"id <x>"

Die Funktion f gibt nichts zurück, macht:
	Die Zahl z ist 1.
	Die Zahlen Referenz r ist id z.
Und kann so benutzt werden:
	"f"
`)

	call, ok := nodes.initVal(t, "r").(*ast.FuncCall)
	require.True(t, ok)
	assertRefOps(t, RefOpsMeta{Op: OpValue, BaseLevel: 1, Level: 1}, call)
	assertRefOps(t, RefOpsMeta{Op: OpAddr, BaseLevel: 1, Level: 1}, call.Args["x"])

	// the body of the instantiation was annotated
	require.True(t, ast.IsGenericInstantiation(call.Func))
	ret, ok := call.Func.Body.Statements[0].(*ast.ReturnStmt)
	require.True(t, ok)
	assertRefOps(t, RefOpsMeta{Op: OpValue, BaseLevel: 1, Level: 1}, ret.Value)
}

func TestTernary(t *testing.T) {
	nodes := parseAnnotated(t, `
Die Funktion f gibt nichts zurück, macht:
	Die Zahl x ist 1.
	Die Zahlen Referenz r ist x.
	Die Zahl y ist r, falls wahr, ansonsten 2.
Und kann so benutzt werden:
	"f"
`)

	ternary, ok := nodes.initVal(t, "y").(*ast.TernaryExpr)
	require.True(t, ok)
	assertRefOps(t, RefOpsMeta{Op: OpValue}, ternary)
	assertRefOps(t, RefOpsMeta{Op: OpValue, BaseLevel: 1, Derefs: 1, Level: 0}, ternary.Lhs)
}

func TestExplicitDeref(t *testing.T) {
	nodes := parseAnnotated(t, `
Die Funktion val mit dem Parameter a vom Typ Zahl, gibt nichts zurück, macht:
	Speichere a plus 1 in a.
Und kann so benutzt werden:
	"Benutze <a>"

Die Funktion f gibt nichts zurück, macht:
	Die Zahl x ist 1.
	Die Zahlen Referenz r ist x.
	Die Zahl y ist der Wert von r.
	Benutze (der Wert von r).
Und kann so benutzt werden:
	"f"
`)

	deref := nodes.initVal(t, "y")
	assertRefOps(t, RefOpsMeta{Op: OpDeref, BaseLevel: 0, Level: 0}, deref)
	assertRefOps(t, RefOpsMeta{Op: OpValue, BaseLevel: 1, Level: 1}, deref.(*ast.UnaryExpr).Rhs)

	call, ok := nodes.exprs[0].Expr.(*ast.FuncCall)
	require.True(t, ok)
	grouping, ok := call.Args["a"].(*ast.Grouping)
	require.True(t, ok)
	assertRefOps(t, RefOpsMeta{Op: OpDeref, BaseLevel: 0, Level: 0}, grouping.Expr)
	assertRefOps(t, RefOpsMeta{Op: OpDeref, BaseLevel: 0, Level: 0}, grouping)
}

func TestConstants(t *testing.T) {
	nodes := parseAnnotated(t, `
Die Konstante C ist 5.

Die Funktion f gibt nichts zurück, macht:
	Die Zahl x ist C.
Und kann so benutzt werden:
	"f"
`)

	assertRefOps(t, RefOpsMeta{Op: OpValue}, nodes.initVal(t, "x"))
}

func TestOperatorOverloads(t *testing.T) {
	nodes := parseAnnotated(t, `
Die Funktion text_größer mit den Parametern a und b vom Typ Text Referenz und Text Referenz, gibt einen Wahrheitswert zurück, macht:
	Gib wahr zurück.
Und überlädt den "größer als" Operator.

Die Funktion f gibt nichts zurück, macht:
	Der Text a ist "a".
	Der Text b ist "b".
	Der Wahrheitswert w ist a größer als b ist.
Und kann so benutzt werden:
	"f"
`)

	binary, ok := nodes.initVal(t, "w").(*ast.BinaryExpr)
	require.True(t, ok)
	require.NotNil(t, binary.OverloadedBy)
	assertRefOps(t, RefOpsMeta{Op: OpValue}, binary)
	// the operands are passed by address
	for _, arg := range binary.OverloadedBy.Call.Args {
		assertRefOps(t, RefOpsMeta{Op: OpAddr, BaseLevel: 1, Level: 1}, arg)
	}
}
