/*
The resolver package should not be used independently from the parser.
It is not a complete Visitor itself, but is rather used to resolve single
Nodes while parsing to ensure correct parsing of function-calls etc.
Because of this you will find many r.visit(x) calls to be commented out.
*/
package resolver

import (
	"fmt"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/token"
)

// holds state to resolve the symbols of an AST and its nodes
// and checking if they are valid
// fills the ASTs SymbolTable while doing so
//
// even though it is a visitor, it should not be used seperately from the parser
// all it's VisitX return values are dummy returns
//
// TODO: add a snychronize method like in the parser to prevent unnessecary errors
type Resolver struct {
	ErrorHandler ddperror.Handler // function to which errors are passed
	CurrentTable *ast.SymbolTable // needed state, public for the parser
	Module       *ast.Module      // the module that is being resolved
	LoopDepth    uint             // for break and continue statements
	panicMode    *bool            // panic mode synchronized with the parser and resolver
}

// create a new resolver to resolve the passed AST
func New(Mod *ast.Module, errorHandler ddperror.Handler, file string, panicMode *bool) *Resolver {
	if errorHandler == nil {
		errorHandler = ddperror.EmptyHandler
	}
	if panicMode == nil {
		panic(fmt.Errorf("panicMode must not be nil"))
	}
	return &Resolver{
		ErrorHandler: errorHandler,
		CurrentTable: Mod.Ast.Symbols,
		Module:       Mod,
		panicMode:    panicMode,
	}
}

// resolve a single node
func (r *Resolver) ResolveNode(node ast.Node) {
	r.visit(node)
}

// helper to visit a node
func (r *Resolver) visit(node ast.Node) {
	node.Accept(r)
}

func (r *Resolver) setScope(symbols *ast.SymbolTable) {
	r.CurrentTable = symbols
}

func (r *Resolver) exitScope() {
	r.CurrentTable = r.CurrentTable.Enclosing
}

// helper for errors
func (r *Resolver) err(code ddperror.ErrorCode, Range token.Range, a ...any) {
	r.Module.Ast.Faulty = true
	if !*r.panicMode {
		*r.panicMode = true
		r.ErrorHandler(ddperror.NewError(code, Range, r.Module.FileName, a...))
	}
}

func (*Resolver) Visitor() {}

// if a BadDecl exists the AST is faulty
func (r *Resolver) VisitBadDecl(decl *ast.BadDecl) ast.VisitResult {
	r.Module.Ast.Faulty = true
	return ast.VisitRecurse
}

func (r *Resolver) VisitConstDecl(decl *ast.ConstDecl) ast.VisitResult {
	r.visit(decl.Val)

	// insert the variable into the current scope (SymbolTable)
	if existed := r.CurrentTable.InsertDecl(decl.Name(), decl); existed {
		r.err(ddperror.NAME_ALREADY_IN_USE, decl.NameTok.Range, decl.Name()) // variables may only be declared once in the same scope
	}

	if decl.Public() && !ast.IsGlobalScope(r.CurrentTable) {
		r.err(ddperror.PUBLIC_CONST_MUST_BE_GLOBAL, decl.NameTok.Range)
	} else if _, alreadyExists := r.Module.PublicDecls[decl.Name()]; decl.IsPublic && !alreadyExists { // insert the variable int othe public module decls
		r.Module.PublicDecls[decl.Name()] = decl
	}

	return ast.VisitRecurse
}

func (r *Resolver) VisitVarDecl(decl *ast.VarDecl) ast.VisitResult {
	r.visit(decl.InitVal) // resolve the initial value
	// insert the variable into the current scope (SymbolTable)
	if existed := r.CurrentTable.InsertDecl(decl.Name(), decl); existed {
		r.err(ddperror.NAME_ALREADY_IN_USE, decl.NameTok.Range, decl.Name()) // variables may only be declared once in the same scope
	}

	if decl.Public() && !ast.IsGlobalScope(r.CurrentTable) {
		r.err(ddperror.PUBLIC_VAR_MUST_BE_GLOBAL, decl.NameTok.Range)
	} else if _, alreadyExists := r.Module.PublicDecls[decl.Name()]; decl.IsPublic && !alreadyExists { // insert the variable int othe public module decls
		r.Module.PublicDecls[decl.Name()] = decl
	}
	return ast.VisitRecurse
}

func (r *Resolver) VisitFuncDecl(decl *ast.FuncDecl) ast.VisitResult {
	// all of the below was already resolved by the parser

	/*
		if existed := r.CurrentTable.InsertFunc(decl.Name.Literal, decl); existed {
			r.err(decl.Name, "Die Funktion '%s' existiert bereits", decl.Name.Literal) // functions may only be declared once
		}
		if !ast.IsExternFunc(decl) {
			decl.Body.Symbols = ast.NewSymbolTable(r.CurrentTable) // create a new scope for the function body
			// add the function parameters to the scope of the function body
			for i, l := 0, len(decl.ParamNames); i < l; i++ {
				decl.Body.Symbols.InsertVar(decl.ParamNames[i].Literal, &ast.VarDecl{Name: decl.ParamNames[i], Type: decl.ParamTypes[i].Type, Range: token.NewRange(decl.ParamNames[i], decl.ParamNames[i])})
			}

			r.visit(decl.Body) // resolve the function body
		}
	*/
	return ast.VisitRecurse
}

func (r *Resolver) VisitFuncDef(def *ast.FuncDef) ast.VisitResult {
	def.Func.Def = def
	return ast.VisitRecurse
}

func (r *Resolver) VisitStructDecl(decl *ast.StructDecl) ast.VisitResult {
	if !ast.IsGlobalScope(r.CurrentTable) {
		r.err(ddperror.TYPE_MUST_BE_GLOBAL, decl.NameTok.Range)
	}

	for _, field := range decl.Fields {
		if varDecl, isVar := field.(*ast.VarDecl); isVar {
			r.visit(varDecl.InitVal)
		} else { // BadDecl
			r.visit(field)
		}
	}
	// insert the struct into the current scope (SymbolTable)
	if existed := r.CurrentTable.InsertDecl(decl.Name(), decl); existed {
		r.err(ddperror.NAME_ALREADY_IN_USE, decl.NameTok.Range, decl.Name()) // structs may only be declared once in the same module
	}
	// insert the struct into the public module decls
	if _, alreadyExists := r.Module.PublicDecls[decl.Name()]; decl.IsPublic && !alreadyExists {
		r.Module.PublicDecls[decl.Name()] = decl
	}
	return ast.VisitRecurse
}

func (r *Resolver) VisitTypeAliasDecl(decl *ast.TypeAliasDecl) ast.VisitResult {
	if !ast.IsGlobalScope(r.CurrentTable) {
		r.err(ddperror.TYPE_MUST_BE_GLOBAL, decl.NameTok.Range)
	}

	if existed := r.CurrentTable.InsertDecl(decl.Name(), decl); existed {
		r.err(ddperror.NAME_ALREADY_IN_USE, decl.NameTok.Range, decl.Name()) // type aliases may only be declared once in the same module
	}
	// insert the type decl into the public module decls
	if _, alreadyExists := r.Module.PublicDecls[decl.Name()]; decl.IsPublic && !alreadyExists {
		r.Module.PublicDecls[decl.Name()] = decl
	}
	return ast.VisitRecurse
}

func (r *Resolver) VisitTypeDefDecl(decl *ast.TypeDefDecl) ast.VisitResult {
	if !ast.IsGlobalScope(r.CurrentTable) {
		r.err(ddperror.TYPE_MUST_BE_GLOBAL, decl.NameTok.Range)
	}

	if existed := r.CurrentTable.InsertDecl(decl.Name(), decl); existed {
		r.err(ddperror.NAME_ALREADY_IN_USE, decl.NameTok.Range, decl.Name()) // type defs may only be declared once in the same module
	}
	// insert the type decl into the public module decls
	if _, alreadyExists := r.Module.PublicDecls[decl.Name()]; decl.IsPublic && !alreadyExists {
		r.Module.PublicDecls[decl.Name()] = decl
	}
	return ast.VisitRecurse
}

// if a BadExpr exists the AST is faulty
func (r *Resolver) VisitBadExpr(expr *ast.BadExpr) ast.VisitResult {
	r.Module.Ast.Faulty = true
	return ast.VisitRecurse
}

func (r *Resolver) VisitIdent(expr *ast.Ident) ast.VisitResult {
	// check if the variable exists
	if decl, exists, isVar := r.CurrentTable.LookupDecl(expr.Literal.Literal); !exists {
		r.err(ddperror.VAR_NOT_DECLARED, expr.Token().Range, expr.Literal.Literal)
	} else if !isVar {
		r.err(ddperror.NAME_NOT_A_VAR, expr.Token().Range, expr.Literal.Literal)
	} else { // set the reference to the declaration
		expr.Declaration = decl
	}
	return ast.VisitRecurse
}

func (r *Resolver) VisitIndexing(expr *ast.Indexing) ast.VisitResult {
	r.visit(expr.Lhs)
	r.visit(expr.Index)
	return ast.VisitRecurse
}

func (r *Resolver) VisitFieldAccess(expr *ast.FieldAccess) ast.VisitResult {
	r.visit(expr.Rhs)
	return ast.VisitRecurse
}

// nothing to do for literals
func (r *Resolver) VisitIntLit(expr *ast.IntLit) ast.VisitResult {
	return ast.VisitRecurse
}

func (r *Resolver) VisitFloatLit(expr *ast.FloatLit) ast.VisitResult {
	return ast.VisitRecurse
}

func (r *Resolver) VisitBoolLit(expr *ast.BoolLit) ast.VisitResult {
	return ast.VisitRecurse
}

func (r *Resolver) VisitCharLit(expr *ast.CharLit) ast.VisitResult {
	return ast.VisitRecurse
}

func (r *Resolver) VisitStringLit(expr *ast.StringLit) ast.VisitResult {
	return ast.VisitRecurse
}

func (r *Resolver) VisitListLit(expr *ast.ListLit) ast.VisitResult {
	if expr.Values != nil {
		for _, v := range expr.Values {
			r.visit(v)
		}
	} else if expr.Count != nil && expr.Value != nil {
		r.visit(expr.Count)
		r.visit(expr.Value)
	}
	return ast.VisitRecurse
}

func (r *Resolver) VisitUnaryExpr(expr *ast.UnaryExpr) ast.VisitResult {
	r.visit(expr.Rhs)
	return ast.VisitRecurse
}

func (r *Resolver) VisitBinaryExpr(expr *ast.BinaryExpr) ast.VisitResult {
	// for field access the left operand should always be an *Ident
	if expr.Operator != ast.BIN_FIELD_ACCESS {
		r.visit(expr.Lhs)
	} else if _, isIdent := expr.Lhs.(*ast.Ident); !isIdent {
		r.err(ddperror.OPERATOR_VON_EXPECTS_NAME, expr.Lhs.GetRange(), expr.Lhs)
	}
	r.visit(expr.Rhs)
	return ast.VisitRecurse
}

func (r *Resolver) VisitTernaryExpr(expr *ast.TernaryExpr) ast.VisitResult {
	r.visit(expr.Lhs)
	r.visit(expr.Mid)
	r.visit(expr.Rhs) // visit the actual expressions
	return ast.VisitRecurse
}

func (r *Resolver) VisitCastExpr(expr *ast.CastExpr) ast.VisitResult {
	r.visit(expr.Lhs) // visit the actual expressions
	return ast.VisitRecurse
}

func (r *Resolver) VisitTypeOpExpr(expr *ast.TypeOpExpr) ast.VisitResult {
	return ast.VisitRecurse
}

func (r *Resolver) VisitTypeCheck(expr *ast.TypeCheck) ast.VisitResult {
	r.visit(expr.Lhs)
	return ast.VisitRecurse
}

func (r *Resolver) VisitGrouping(expr *ast.Grouping) ast.VisitResult {
	r.visit(expr.Expr)
	return ast.VisitRecurse
}

func (r *Resolver) VisitFuncCall(expr *ast.FuncCall) ast.VisitResult {
	// visit the passed arguments
	for _, v := range expr.Args {
		r.visit(v)
	}
	return ast.VisitRecurse
}

func (r *Resolver) VisitStructLiteral(expr *ast.StructLiteral) ast.VisitResult {
	for _, arg := range expr.Args {
		r.visit(arg)
	}
	return ast.VisitRecurse
}

// if a BadStmt exists the AST is faulty
func (r *Resolver) VisitBadStmt(stmt *ast.BadStmt) ast.VisitResult {
	r.Module.Ast.Faulty = true
	return ast.VisitRecurse
}

func (r *Resolver) VisitDeclStmt(stmt *ast.DeclStmt) ast.VisitResult {
	r.visit(stmt.Decl)
	return ast.VisitRecurse
}

func (r *Resolver) VisitExprStmt(stmt *ast.ExprStmt) ast.VisitResult {
	r.visit(stmt.Expr)
	return ast.VisitRecurse
}

func (r *Resolver) VisitImportStmt(stmt *ast.ImportStmt) ast.VisitResult {
	if len(stmt.Modules) == 0 {
		return ast.VisitRecurse
	}

	resolveDecl := func(decl ast.Declaration) {
		if existed := r.CurrentTable.InsertDecl(decl.Name(), decl); existed {
			r.err(ddperror.NAME_ALREADY_USED_IN_MODULE, stmt.FileName.Range, decl.Name(), decl.Module().GetIncludeFilename())
			return
		}
	}

	// add imported symbols
	ast.IterateImportedDecls(stmt, func(name string, decl ast.Declaration, tok token.Token) bool {
		if decl == nil {
			r.err(ddperror.NAME_NOT_PUBLIC_IN_MODULE, tok.Range, name, ast.TrimStringLit(&stmt.FileName))
		} else {
			resolveDecl(decl)
		}
		return true
	})
	return ast.VisitRecurse
}

func (r *Resolver) VisitAssignStmt(stmt *ast.AssignStmt) ast.VisitResult {
	switch assign := stmt.Var.(type) {
	case *ast.Ident:
		// check if the variable exists
		if varDecl, exists, isVar := r.CurrentTable.LookupDecl(assign.Literal.Literal); !exists {
			r.err(ddperror.VAR_NOT_DECLARED, assign.Literal.Range, assign.Literal.Literal)
		} else if !isVar {
			r.err(ddperror.NAME_NOT_A_VAR, assign.Token().Range, assign.Literal.Literal)
		} else if _, isConst := varDecl.(*ast.ConstDecl); isConst {
			r.err(ddperror.CONST_NOT_ASSIGNABLE, assign.Token().Range, assign.Literal.Literal)
		} else { // set the reference to the declaration
			assign.Declaration = varDecl
		}
	case *ast.Indexing:
		r.visit(assign.Lhs)
		r.visit(assign.Index)
	case *ast.FieldAccess:
		r.visit(assign.Rhs)
	}
	r.visit(stmt.Rhs)
	return ast.VisitRecurse
}

func (r *Resolver) VisitBlockStmt(stmt *ast.BlockStmt) ast.VisitResult {
	if stmt.Symbols == nil {
		r.setScope(stmt.Symbols) // set the current scope to the block
		// visit every statement in the block
		for _, stmt := range stmt.Statements {
			r.visit(stmt)
		}
		r.exitScope() // restore the enclosing scope
	}
	return ast.VisitRecurse
}

func (r *Resolver) VisitIfStmt(stmt *ast.IfStmt) ast.VisitResult {
	r.visit(stmt.Condition)
	if _, ok := stmt.Then.(*ast.BlockStmt); !ok {
		r.visit(stmt.Then)
	}
	if stmt.Else != nil {
		if _, ok := stmt.Else.(*ast.BlockStmt); !ok {
			r.visit(stmt.Else)
		}
	}
	return ast.VisitRecurse
}

func (r *Resolver) VisitWhileStmt(stmt *ast.WhileStmt) ast.VisitResult {
	r.visit(stmt.Condition)
	// r.visit(stmt.Body)
	return ast.VisitRecurse
}

func (r *Resolver) VisitForStmt(stmt *ast.ForStmt) ast.VisitResult {
	r.setScope(stmt.Body.Symbols)
	// only visit the InitVal because the variable is already in the scope
	r.visit(stmt.Initializer.InitVal)
	r.visit(stmt.To)
	if stmt.StepSize != nil {
		r.visit(stmt.StepSize)
	}
	// r.visit(stmt.Body) // created by calling checkedDeclration in the parser so already resolved
	r.exitScope()
	return ast.VisitRecurse
}

func (r *Resolver) VisitForRangeStmt(stmt *ast.ForRangeStmt) ast.VisitResult {
	r.setScope(stmt.Body.Symbols)
	// only visit the InitVal because the variable is already in the scope
	r.visit(stmt.Initializer.InitVal)
	r.visit(stmt.In)
	// r.visit(stmt.Body) // created by calling checkedDeclration in the parser so already resolved
	r.exitScope()
	return ast.VisitRecurse
}

func (r *Resolver) VisitBreakContinueStmt(stmt *ast.BreakContinueStmt) ast.VisitResult {
	if r.LoopDepth == 0 {
		r.err(ddperror.BREAK_OR_CONTINUE_ONLY_IN_LOOP, stmt.Tok.Range)
	}
	return ast.VisitRecurse
}

func (r *Resolver) VisitReturnStmt(stmt *ast.ReturnStmt) ast.VisitResult {
	if stmt.Value == nil {
		return ast.VisitRecurse
	}
	r.visit(stmt.Value)
	return ast.VisitRecurse
}

func (*Resolver) VisitTodoStmt(*ast.TodoStmt) ast.VisitResult {
	return ast.VisitRecurse
}
