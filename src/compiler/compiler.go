package compiler

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ast/annotators"
	"github.com/DDP-Projekt/Kompilierer/src/compiler/llvm"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/ddppath"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
	"github.com/DDP-Projekt/Kompilierer/src/token"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"

	"github.com/llir/irutil"
)

// compiles a mainModule and all it's imports
// every module is written to a io.Writer created
// by calling destCreator with the given module
// returns:
//   - a set of all external dependendcies
//   - an error
func compileWithImports(mod *ast.Module, destCreator func(*ast.Module) io.Writer,
	errHndl ddperror.Handler, optimizationLevel uint,
) (map[string]struct{}, error) {
	compiledMods := map[string]*ast.Module{}
	dependencies := map[string]struct{}{}
	return compileWithImportsRec(mod, destCreator, compiledMods, dependencies, true, errHndl, optimizationLevel)
}

func compileWithImportsRec(mod *ast.Module, destCreator func(*ast.Module) io.Writer,
	compiledMods map[string]*ast.Module, dependencies map[string]struct{},
	isMainModule bool, errHndl ddperror.Handler, optimizationLevel uint,
) (map[string]struct{}, error) {
	// the ast must be valid (and should have been resolved and typechecked beforehand)
	if mod.Ast.Faulty {
		return nil, fmt.Errorf("Fehlerhafter Quellcode im Modul '%s', Kompilierung abgebrochen", mod.GetIncludeFilename())
	}

	// check if the module was already compiled
	if _, alreadyCompiled := compiledMods[mod.FileName]; !alreadyCompiled {
		compiledMods[mod.FileName] = mod // add the module to the set
	} else {
		return dependencies, nil // break the recursion if the module was already compiled
	}

	// add the external dependencies
	for path := range mod.ExternalDependencies {
		if abspath, err := filepath.Abs(filepath.Join(filepath.Dir(mod.FileName), path)); err != nil {
			errHndl(ddperror.New(ddperror.MISC_INCLUDE_ERROR, ddperror.LEVEL_ERROR, token.Range{},
				fmt.Sprintf("Es konnte kein Absoluter Dateipfad für die Datei '%s' gefunden werden: %s", path, err), mod.FileName))
		} else {
			path = abspath
		}
		dependencies[path] = struct{}{}
	}

	// compile this module
	if _, err := newCompiler(mod, errHndl, optimizationLevel).compile(destCreator(mod), isMainModule); err != nil {
		return nil, fmt.Errorf("Fehler beim Kompilieren des Moduls '%s': %w", mod.GetIncludeFilename(), err)
	}

	// recursively compile the other dependencies
	for _, imprt := range mod.Imports {
		for _, imprtMod := range imprt.Modules {
			if _, err := compileWithImportsRec(imprtMod, destCreator, compiledMods, dependencies, false, errHndl, optimizationLevel); err != nil {
				return nil, err
			}
		}
	}

	return dependencies, nil
}

// small wrapper for a ast.FuncDecl and the corresponding ir function
type funcWrapper struct {
	irFunc   *ir.Func      // the function in the llvm ir
	funcDecl *ast.FuncDecl // the ast.FuncDecl
}

// holds state to compile a DDP AST into llvm ir
type compiler struct {
	ddpModule         *ast.Module      // the module to be compiled
	mod               *ir.Module       // the ir module (basically the ir file)
	errorHandler      ddperror.Handler // errors are passed to this function
	optimizationLevel uint             // level of optimization
	result            *Result          // result of the compilation
	llTarget          llvmTarget       // information about the target machine

	cbb              *ir.Block                                 // current basic block in the ir
	cf               *ir.Func                                  // current function
	scp              *scope                                    // current scope in the ast (not in the ir)
	cfscp            *scope                                    // out-most scope of the current function
	functions        map[string]*funcWrapper                   // all the global functions
	typeMap          map[ddptypes.Type]*ast.Module             // maps ddpTypes to the module they originate from
	structTypes      map[*ddptypes.StructType]*ddpIrStructType // struct names mapped to their IR type
	latestReturn     value.Value                               // return of the latest evaluated expression (in the ir)
	latestReturnType ddpIrType                                 // the type of latestReturn
	latestIsTemp     bool                                      // ewther the latestReturn is a temporary or not
	importedModules  map[*ast.Module]struct{}                  // all the modules that have already been imported
	currentNode      ast.Node                                  // used for error reporting
	typeDefVTables   map[string]constant.Constant

	moduleInitFunc             *ir.Func  // the module_init func of this module
	moduleInitCbb              *ir.Block // cbb but for module_init
	moduleDisposeFunc          *ir.Func
	out_of_bounds_error_string *ir.Global
	slice_error_string         *ir.Global
	todo_error_string          *ir.Global
	bad_cast_error_string      *ir.Global
	invalid_utf8_error_string  *ir.Global

	curLeaveBlock    *ir.Block // leave block of the current loop
	curContinueBlock *ir.Block // block where a continue should jump to
	curLoopScope     *scope    // scope of the current loop for break/continue to free to

	// all the type definitions of inbuilt types used by the compiler
	void                                                                          *ddpIrVoidType
	ddpinttyp, ddpfloattyp, ddpbooltyp, ddpchartyp                                *ddpIrPrimitiveType
	ddpstring                                                                     *ddpIrStringType
	ddpany                                                                        *ddpIrAnyType
	ddpintlist, ddpfloatlist, ddpboollist, ddpcharlist, ddpstringlist, ddpanylist *ddpIrListType
}

// create a new Compiler to compile the passed AST
func newCompiler(module *ast.Module, errorHandler ddperror.Handler, optimizationLevel uint) *compiler {
	if errorHandler == nil { // default error handler does nothing
		errorHandler = ddperror.EmptyHandler
	}
	return &compiler{
		ddpModule:         module,
		mod:               ir.NewModule(),
		errorHandler:      errorHandler,
		optimizationLevel: optimizationLevel,
		result: &Result{
			Dependencies: make(map[string]struct{}),
		},
		cbb:              nil,
		cf:               nil,
		scp:              newScope(nil), // global scope
		cfscp:            nil,
		functions:        make(map[string]*funcWrapper),
		typeMap:          createTypeMap(module),
		structTypes:      make(map[*ddptypes.StructType]*ddpIrStructType),
		latestReturn:     nil,
		latestReturnType: nil,
		latestIsTemp:     false,
		importedModules:  make(map[*ast.Module]struct{}),
		typeDefVTables:   make(map[string]constant.Constant),
		curLeaveBlock:    nil,
		curContinueBlock: nil,
		curLoopScope:     nil,
	}
}

// compile the AST contained in c
// if w is not nil, the resulting llir is written to w
// otherwise a string representation is returned in result
// if isMainModule is false, no ddp_main function will be generated
func (c *compiler) compile(w io.Writer, isMainModule bool) (result *Result, rerr error) {
	defer compiler_panic_wrapper(c)

	llTarget, err := newllvmTarget()
	if err != nil {
		return nil, err
	}
	c.llTarget = *llTarget

	c.mod.SourceFilename = c.ddpModule.FileName // set the module filename (optional metadata)
	c.addExternalDependencies()

	c.setup()

	if isMainModule {
		// called from the ddp-c-runtime after initialization
		ddpmain := c.insertFunction(
			"ddp_ddpmain",
			nil,
			c.mod.NewFunc("ddp_ddpmain", ddpint),
		)
		c.cf = ddpmain               // first function is ddpmain
		c.cbb = ddpmain.NewBlock("") // first block
	}

	// visit every statement in the modules AST and compile it
	for _, stmt := range c.ddpModule.Ast.Statements {
		if isMainModule {
			c.visitNode(stmt)
		} else {
			switch stmt.(type) {
			case *ast.DeclStmt, *ast.ImportStmt, *ast.FuncDef:
				c.visitNode(stmt)
			default:
				// in imports we only visit declarations and ignore other top-level statements
			}
		}
	}

	if isMainModule {
		c.scp = c.exitScope(c.scp) // exit the main scope
		// call all the module_dispose functions
		for mod := range c.importedModules {
			_, dispose_name := getModuleInitDisposeName(mod)
			dispose_fun := c.functions[dispose_name]
			c.cbb.NewCall(dispose_fun.irFunc)
		}
		// on success ddpmain returns 0
		c.cbb.NewRet(zero)
	}

	c.moduleInitCbb.NewRet(nil) // terminate the module_init func

	_, err = c.mod.WriteTo(w)
	return c.result, err
}

// dumps only the definitions for inbuilt list types to w
func (c *compiler) dumpListDefinitions(w io.Writer) error {
	defer compiler_panic_wrapper(c)

	c.mod.SourceFilename = ddppath.LIST_DEFS_NAME

	c.setupErrorStrings()
	// the order of these function calls is important
	// because the primitive types need to be setup
	// before the list types
	// and the void type before everything else
	c.void = &ddpIrVoidType{}
	c.initRuntimeFunctions()
	c.setupPrimitiveTypes(false)
	c.ddpstring = c.defineStringType(false)
	c.ddpany = c.defineAnyType()
	c.setupListTypes(false) // we want definitions

	_, err := c.mod.WriteTo(w)
	return err
}

func (c *compiler) addExternalDependencies() {
	// add the external dependencies
	for path := range c.ddpModule.ExternalDependencies {
		if abspath, err := filepath.Abs(filepath.Join(filepath.Dir(c.ddpModule.FileName), path)); err != nil {
			c.errorHandler(ddperror.New(ddperror.MISC_INCLUDE_ERROR, ddperror.LEVEL_ERROR, token.Range{},
				fmt.Sprintf("Es konnte kein Absoluter Dateipfad für die Datei '%s' gefunden werden: %s", path, err), c.ddpModule.FileName))
		} else {
			path = abspath
		}
		c.result.Dependencies[path] = struct{}{}
	}
}

// if the llvm-ir should be commented
// increases the intermediate file size
var Comments_Enabled = true

func (c *compiler) commentNode(block *ir.Block, node ast.Node, details string) {
	if Comments_Enabled {
		comment := fmt.Sprintf("F %s, %d:%d: %s", c.ddpModule.FileName, node.Token().Range.Start.Line, node.Token().Range.Start.Column, node)
		if details != "" {
			comment += " (" + details + ")"
		}
		c.comment(comment, block)
	}
}

func (c *compiler) comment(comment string, block *ir.Block) {
	if Comments_Enabled {
		block.Insts = append(block.Insts, irutil.NewComment(comment))
	}
}

// helper to visit a single node
func (c *compiler) visitNode(node ast.Node) {
	c.currentNode = node
	node.Accept(c)
}

// helper to evaluate an expression and return its ir value and type
// the  bool signals wether the returned value is a temporary value that can be claimed
// or if it is a 'reference' to a variable that must be copied
func (c *compiler) evaluate(expr ast.Expression) (value.Value, ddpIrType, bool) {
	c.visitNode(expr)
	return c.latestReturn, c.latestReturnType, c.latestIsTemp
}

// helper to insert a function into the global function map
// returns the ir function
func (c *compiler) insertFunction(name string, funcDecl *ast.FuncDecl, irFunc *ir.Func) *ir.Func {
	c.functions[name] = &funcWrapper{
		funcDecl: funcDecl,
		irFunc:   irFunc,
	}
	return irFunc
}

func (c *compiler) setup() {
	c.setupErrorStrings()

	// the order of these function calls is important
	// because the primitive types need to be setup
	// before the list types
	c.void = &ddpIrVoidType{}
	c.initRuntimeFunctions()
	c.setupPrimitiveTypes(true)
	c.ddpstring = c.defineStringType(true)
	c.ddpany = c.defineAnyType()
	c.setupListTypes(true)

	c.setupModuleInitDispose()

	c.setupOperators()
}

// used in setup()
func (c *compiler) setupErrorStrings() {
	createErrorString := func(msg string) *ir.Global {
		error_string := c.mod.NewGlobalDef("", constant.NewCharArrayFromString(msg))
		error_string.Linkage = enum.LinkageInternal
		error_string.Visibility = enum.VisibilityDefault
		error_string.Immutable = true
		return error_string
	}

	c.out_of_bounds_error_string = createErrorString("Zeile %lld, Spalte %lld: Index außerhalb der Listen Länge (Index war %ld, Listen Länge war %ld)\n")
	c.slice_error_string = createErrorString("Invalide Indexe (Index 1 war %ld, Index 2 war %ld)\n")
	c.todo_error_string = createErrorString("Zeile %lld, Spalte %lld: Dieser Teil des Programms wurde noch nicht implementiert\n")
	c.bad_cast_error_string = createErrorString("Zeile %lld, Spalte %lld: Falsche Typumwandlung")
	c.invalid_utf8_error_string = createErrorString("Zeile %lld, Spalte %lld: Invalider UTF8 Wert im Text")
}

// used in setup()
func (c *compiler) setupPrimitiveTypes(declarationOnly bool) {
	c.ddpinttyp = c.definePrimitiveType(ddpint, zero, llvm.Int64Type(), "ddpint", declarationOnly)
	c.ddpfloattyp = c.definePrimitiveType(ddpfloat, zerof, llvm.DoubleType(), "ddpfloat", declarationOnly)
	c.ddpbooltyp = c.definePrimitiveType(ddpbool, constant.False, llvm.Int1Type(), "ddpbool", declarationOnly)
	c.ddpchartyp = c.definePrimitiveType(ddpchar, newIntT(ddpchar, 0), llvm.Int32Type(), "ddpchar", declarationOnly)
}

// used in setup()
func (c *compiler) setupListTypes(declarationOnly bool) {
	c.ddpintlist = c.createListType("ddpintlist", c.ddpinttyp, declarationOnly)
	c.ddpfloatlist = c.createListType("ddpfloatlist", c.ddpfloattyp, declarationOnly)
	c.ddpboollist = c.createListType("ddpboollist", c.ddpbooltyp, declarationOnly)
	c.ddpcharlist = c.createListType("ddpcharlist", c.ddpchartyp, declarationOnly)
	c.ddpstringlist = c.createListType("ddpstringlist", c.ddpstring, declarationOnly)
	c.ddpanylist = c.createListType("ddpanylist", c.ddpany, declarationOnly)
}

// used in setup()
// creates a function that can be called to initialize the global state of this module
func (c *compiler) setupModuleInitDispose() {
	init_name, dispose_name := getModuleInitDisposeName(c.ddpModule)
	c.moduleInitFunc = c.mod.NewFunc(init_name, c.void.IrType())
	c.moduleInitFunc.Visibility = enum.VisibilityDefault
	c.moduleInitCbb = c.moduleInitFunc.NewBlock("")
	c.insertFunction(init_name, nil, c.moduleInitFunc)

	c.moduleDisposeFunc = c.mod.NewFunc(dispose_name, c.void.IrType())
	c.moduleDisposeFunc.Visibility = enum.VisibilityDefault
	c.moduleDisposeFunc.NewBlock("").NewRet(nil)
	c.insertFunction(dispose_name, nil, c.moduleDisposeFunc)
}

// used in setup()
func (c *compiler) setupOperators() {
	// hoch operator for different type combinations
	c.declareExternalRuntimeFunction("pow", ddpfloat, ir.NewParam("f1", ddpfloat), ir.NewParam("f2", ddpfloat))

	// logarithm
	c.declareExternalRuntimeFunction("log10", ddpfloat, ir.NewParam("f", ddpfloat))

	// ddpstring to type cast
	c.declareExternalRuntimeFunction("ddp_string_to_int", ddpint, ir.NewParam("str", c.ddpstring.ptr))
	c.declareExternalRuntimeFunction("ddp_string_to_float", ddpfloat, ir.NewParam("str", c.ddpstring.ptr))
}

// deep copies the value pointed to by src into dest
// and returns dest
func (c *compiler) deepCopyInto(dest, src value.Value, typ ddpIrType) value.Value {
	c.cbb.NewCall(typ.DeepCopyFunc(), dest, src)
	return dest
}

// calls the corresponding free function on val
// if typ.IsPrimitive() == false
func (c *compiler) freeNonPrimitive(val value.Value, typ ddpIrType) {
	if !typ.IsPrimitive() {
		c.cbb.NewCall(typ.FreeFunc(), val)
	}
}

// claims the given value if possible, copies it otherwise
// dest should be a value that is definetly freed at some point (meaning a variable or list-element etc.)
func (c *compiler) claimOrCopy(dest, val value.Value, valTyp ddpIrType, isTemp bool) {
	if !valTyp.IsPrimitive() {
		if isTemp { // temporaries can be claimed
			val = c.cbb.NewLoad(valTyp.IrType(), c.scp.claimTemporary(val))
			c.cbb.NewStore(val, dest)
		} else { // non-temporaries need to be copied
			c.deepCopyInto(dest, val, valTyp)
		}
	} else { // primitives are trivially copied
		c.cbb.NewStore(val, dest) // store the value
	}
}

func (c *compiler) freeTemporaries(scp *scope, force bool) {
	for _, v := range scp.temporaries {
		if !v.protected || force {
			c.freeNonPrimitive(v.val, v.typ)
		}
	}
}

// helper to exit a scope
// frees all local variables
// returns the enclosing scope
func (c *compiler) exitScope(scp *scope) *scope {
	for _, v := range scp.variables {
		if !v.isRef && !v.protected {
			c.freeNonPrimitive(v.val, v.typ)
		}
	}
	c.freeTemporaries(scp, false)
	return scp.enclosing
}

func (c *compiler) exitFuncScope(fun *ast.FuncDecl) *scope {
	meta := annotators.ConstFuncParamMeta{}
	if attachement, ok := fun.Module().Ast.GetMetadataByKind(fun, annotators.ConstFuncParamMetaKind); ok {
		meta = attachement.(annotators.ConstFuncParamMeta)
	}

	for paramDecl, v := range c.cfscp.variables {
		if !v.isRef && (!meta.IsConst[paramDecl.Name()] || c.optimizationLevel < 2) {
			c.freeNonPrimitive(v.val, v.typ)
		}
	}
	c.freeTemporaries(c.cfscp, true)
	return c.cfscp.enclosing
}

func (*compiler) Visitor() {}

// should have been filtered by the resolver/typechecker, so err
func (c *compiler) VisitBadDecl(d *ast.BadDecl) ast.VisitResult {
	c.err("Es wurde eine invalide Deklaration gefunden")
	return ast.VisitRecurse
}

func (c *compiler) VisitConstDecl(d *ast.ConstDecl) ast.VisitResult {
	return ast.VisitRecurse
}

func (c *compiler) VisitVarDecl(d *ast.VarDecl) ast.VisitResult {
	// allocate the variable on the function call frame
	// all local variables are allocated in the first basic block of the function they are within
	// in the ir a local variable is a alloca instruction (a stack allocation)

	Typ := c.toIrType(d.Type) // get the llvm type
	var varLocation value.Value
	if c.scp.isGlobalScope() { // global scope
		// globals are first assigned in ddp_main or module_init
		// so we assign them a default value here
		//
		// names are mangled only in the actual ir-definitions, not in the compiler data-structures
		globalDef := c.mod.NewGlobalDef(c.mangledNameDecl(d), Typ.DefaultValue())
		// make private variables static like in C
		// commented out because of generics where private variables might be used
		// from a different module
		// if !d.IsPublic && !d.IsExternVisible {
		// 	globalDef.Linkage = enum.LinkageInternal
		// }
		globalDef.Visibility = enum.VisibilityDefault
		varLocation = globalDef
	} else {
		c.commentNode(c.cbb, d, d.Name())
		varLocation = c.NewAlloca(Typ.IrType())
	}

	// adds the variable initializer to the function fun
	addInitializer := func() {
		initVal, initTyp, isTemp := c.evaluate(d.InitVal) // evaluate the initial value

		// implicit cast to any if required
		if ddptypes.DeepEqual(d.Type, ddptypes.VARIABLE) && initTyp != c.ddpany {
			vtable := initTyp.VTable()
			if typeDef, isTypeDef := ddptypes.CastTypeDef(d.InitType); isTypeDef {
				vtable = c.typeDefVTables[c.mangledNameType(typeDef)]
			}

			initVal, _, isTemp = c.castNonAnyToAny(initVal, initTyp, isTemp, vtable)
		}

		c.claimOrCopy(varLocation, initVal, Typ, isTemp)
	}

	if c.scp.isGlobalScope() { // module_init
		cf, cbb := c.cf, c.cbb
		c.cf, c.cbb = c.moduleInitFunc, c.moduleInitCbb
		current_temporaries_end := len(c.scp.temporaries)
		addInitializer() // initialize the variable in module_init
		// free all temporaries that were created in the initializer
		for _, v := range c.scp.temporaries[current_temporaries_end:] {
			c.freeNonPrimitive(v.val, v.typ)
		}
		c.scp.temporaries = c.scp.temporaries[:current_temporaries_end]
		c.moduleInitFunc, c.moduleInitCbb = c.cf, c.cbb

		c.cf, c.cbb = c.moduleDisposeFunc, c.moduleDisposeFunc.Blocks[0]
		c.freeNonPrimitive(varLocation, Typ) // free the variable in module_dispose

		c.cf, c.cbb = cf, cbb
	}

	// if those are nil, we are at the global scope but there is no ddp_main func
	// meaning this module is being compiled as a non-main module
	if c.cf != nil && c.cbb != nil { // ddp_main
		addInitializer()
	}

	c.scp.addVar(d, varLocation, Typ, false)
	return ast.VisitRecurse
}

func (c *compiler) VisitFuncDecl(decl *ast.FuncDecl) ast.VisitResult {
	if ast.IsGeneric(decl) {
		return ast.VisitRecurse
	}

	retType := c.toIrType(decl.ReturnType) // get the llvm type
	retTypeIr := retType.IrType()
	params := make([]*ir.Param, 0, len(decl.Parameters)) // list of the ir parameters

	hasReturnParam := !retType.IsPrimitive()
	// non-primitives are returned by passing a pointer to the struct as first parameter
	if hasReturnParam {
		params = append(params, ir.NewParam("", retType.PtrType()))
		retTypeIr = c.void.IrType()
	}

	// append all the other parameters
	for _, param := range decl.Parameters {
		ty := c.toIrParamType(param.Type)                            // convert the type of the parameter
		params = append(params, ir.NewParam(param.Name.Literal, ty)) // add it to the list
	}

	irFunc := c.mod.NewFunc(c.mangledNameDecl(decl), retTypeIr, params...) // create the ir function
	irFunc.CallingConv = enum.CallingConvC                                 // every function is called with the c calling convention to make interaction with inbuilt stuff easier
	// make private functions static like in C
	// commented out because of generics where private functions might be called
	// from a different module
	// if !decl.IsPublic && !decl.IsExternVisible {
	// 	irFunc.Linkage = enum.LinkageInternal
	// 	irFunc.Visibility = enum.VisibilityDefault
	// }

	c.insertFunction(irFunc.Name(), decl, irFunc)

	// inbuilt or external functions are defined in c
	if ast.IsExternFunc(decl) {
		irFunc.Linkage = enum.LinkageExternal
	} else if !ast.IsForwardDecl(decl) {
		c.defineFuncBody(irFunc, hasReturnParam, params, decl)
	}
	return ast.VisitRecurse
}

func (c *compiler) VisitFuncDef(def *ast.FuncDef) ast.VisitResult {
	fun := c.functions[c.mangledNameDecl(def.Func)] // retreive the function (the resolver took care that it is present)
	retType := c.toIrType(def.Func.ReturnType)      // get the llvm type

	c.defineFuncBody(fun.irFunc, !retType.IsPrimitive(), fun.irFunc.Params, def.Func)
	return ast.VisitRecurse
}

// helper function for VisitFuncDef and VisitFuncDecl to compile the  body of a ir function
func (c *compiler) defineFuncBody(irFunc *ir.Func, hasReturnParam bool, params []*ir.Param, decl *ast.FuncDecl) {
	fun, block, cfscp := c.cf, c.cbb, c.cfscp // safe the state before the function body
	c.cf, c.cbb, c.scp = irFunc, irFunc.NewBlock(""), newScope(c.scp)
	c.cfscp = c.scp

	// we want to skip the possible return-parameter
	if hasReturnParam {
		params = params[1:]
	}

	body := decl.Body
	if ast.IsForwardDecl(decl) {
		body = decl.Def.Body
	}

	// passed arguments are immutable (llvm uses ssa registers) so we declare them as local variables
	// the caller has to take care of possible deep-copies
	for i := range params {
		irType := c.toIrType(decl.Parameters[i].Type.Type)
		varDecl, _, _ := body.Symbols.LookupDecl(params[i].Name())
		paramDecl := varDecl.(*ast.VarDecl)
		if decl.Parameters[i].Type.IsReference {
			// references are implemented similar to name-shadowing
			// they basically just get another name in the function scope, which
			// refers to the same variable allocation
			c.scp.addVar(paramDecl, params[i], irType, true)
		} else if !irType.IsPrimitive() { // strings and lists need special handling
			// add the local variable for the parameter
			v := c.scp.addVar(paramDecl, c.NewAlloca(irType.IrType()), irType, false)
			c.cbb.NewStore(c.cbb.NewLoad(irType.IrType(), params[i]), v) // store the copy in the local variable
		} else { // primitive types don't need any special handling
			v := c.scp.addVar(paramDecl, c.NewAlloca(irType.IrType()), irType, false)
			c.cbb.NewStore(params[i], v)
		}
	}

	// modified VisitBlockStmt
	c.scp = newScope(c.scp) // a block gets its own scope
	toplevelReturn := false
	for _, stmt := range body.Statements {
		c.visitNode(stmt)
		// on toplevel return statements, ignore anything that follows
		if _, ok := stmt.(*ast.ReturnStmt); ok {
			toplevelReturn = true
			break
		}
	}
	// free the local variables of the function
	if toplevelReturn {
		c.scp = c.scp.enclosing
	} else {
		c.scp = c.exitScope(c.scp)
	}

	if c.cbb.Term == nil {
		c.cbb.NewRet(nil) // every block needs a terminator, and every function a return
	}

	// free the parameters of the function
	if toplevelReturn {
		c.scp = c.scp.enclosing
	} else {
		c.scp = c.exitFuncScope(decl)
	}
	c.cf, c.cbb, c.cfscp = fun, block, cfscp // restore state before the function (to main)
}

func (c *compiler) VisitStructDecl(decl *ast.StructDecl) ast.VisitResult {
	c.defineOrDeclareAllDeclTypes(decl)
	return ast.VisitRecurse
}

func (c *compiler) VisitTypeAliasDecl(decl *ast.TypeAliasDecl) ast.VisitResult {
	return ast.VisitRecurse
}

func (c *compiler) VisitTypeDefDecl(decl *ast.TypeDefDecl) ast.VisitResult {
	c.addTypdefVTable(decl, false)
	return ast.VisitRecurse
}

// should have been filtered by the resolver/typechecker, so err
func (c *compiler) VisitBadExpr(e *ast.BadExpr) ast.VisitResult {
	c.err("Es wurde ein invalider Ausdruck gefunden")
	return ast.VisitRecurse
}

func (c *compiler) VisitIdent(e *ast.Ident) ast.VisitResult {
	if decl, isConst := e.Declaration.(*ast.ConstDecl); isConst {
		c.evaluate(decl.Val)
		return ast.VisitRecurse
	}

	if e.Declaration.(*ast.VarDecl).IsGlobal && e.Declaration.Module() != c.ddpModule {
		c.declareImportedVarDecl(e.Declaration.(*ast.VarDecl))
	}

	Var := c.scp.lookupVar(e.Declaration.(*ast.VarDecl)) // get the alloca in the ir
	c.commentNode(c.cbb, e, e.Literal.Literal)

	if Var.typ.IsPrimitive() { // primitives are simply loaded
		c.latestReturn = c.cbb.NewLoad(Var.typ.IrType(), Var.val)
	} else { // non-primitives are used by pointer
		c.latestReturn = Var.val
	}
	c.latestReturnType = Var.typ
	c.latestIsTemp = false
	return ast.VisitRecurse
}

func (c *compiler) VisitIndexing(e *ast.Indexing) ast.VisitResult {
	elementPtr, elementType, stringIndexing := c.evaluateAssignableOrReference(e, false)

	if stringIndexing != nil {
		lhs, _, _ := c.evaluate(stringIndexing.Lhs)
		index, _, _ := c.evaluate(stringIndexing.Index)
		c.latestReturn = c.cbb.NewCall(c.ddpstring.indexIrFun, lhs, index)
		c.latestReturnType = c.ddpchartyp
		// c.latestIsTemp = false // it is a primitive typ, so we don't care
		return ast.VisitRecurse
	} else {
		if elementType.IsPrimitive() {
			c.latestReturn = c.cbb.NewLoad(elementType.IrType(), elementPtr)
		} else {
			c.latestReturn = elementPtr
			c.latestIsTemp = false
		}
	}
	c.latestReturnType = elementType
	return ast.VisitRecurse
}

func (c *compiler) VisitFieldAccess(expr *ast.FieldAccess) ast.VisitResult {
	fieldPtr, fieldType, _ := c.evaluateAssignableOrReference(expr, false)

	if fieldType.IsPrimitive() {
		c.latestReturn = c.cbb.NewLoad(fieldType.IrType(), fieldPtr)
	} else {
		dest := c.NewAlloca(fieldType.IrType())
		c.deepCopyInto(dest, fieldPtr, fieldType)
		c.latestReturn, c.latestReturnType = c.scp.addTemporary(dest, fieldType)
		c.latestIsTemp = true
	}
	c.latestReturnType = fieldType
	return ast.VisitRecurse
}

// literals are simple ir constants
func (c *compiler) VisitIntLit(e *ast.IntLit) ast.VisitResult {
	c.commentNode(c.cbb, e, "")
	c.latestReturn = newInt(e.Value)
	c.latestReturnType = c.ddpinttyp
	return ast.VisitRecurse
}

func (c *compiler) VisitFloatLit(e *ast.FloatLit) ast.VisitResult {
	c.commentNode(c.cbb, e, "")
	c.latestReturn = constant.NewFloat(ddpfloat, e.Value)
	c.latestReturnType = c.ddpfloattyp
	return ast.VisitRecurse
}

func (c *compiler) VisitBoolLit(e *ast.BoolLit) ast.VisitResult {
	c.commentNode(c.cbb, e, "")
	c.latestReturn = constant.NewBool(e.Value)
	c.latestReturnType = c.ddpbooltyp
	return ast.VisitRecurse
}

func (c *compiler) VisitCharLit(e *ast.CharLit) ast.VisitResult {
	c.commentNode(c.cbb, e, "")
	c.latestReturn = newIntT(ddpchar, int64(e.Value))
	c.latestReturnType = c.ddpchartyp
	return ast.VisitRecurse
}

// string literals are created by the runtime
// so we need to do some work here
func (c *compiler) VisitStringLit(e *ast.StringLit) ast.VisitResult {
	constStr := c.mod.NewGlobalDef("", irutil.NewCString(e.Value))
	// call the ddp-runtime function to create the ddpstring
	c.commentNode(c.cbb, e, constStr.Name())
	dest := c.NewAlloca(c.ddpstring.typ)
	if e.Value == "" {
		c.cbb.NewStore(c.ddpstring.DefaultValue(), dest)
	} else {
		c.cbb.NewCall(c.ddpstring.fromConstantsIrFun, dest, c.cbb.NewBitCast(constStr, i8ptr))
	}
	c.latestReturn, c.latestReturnType = c.scp.addTemporary(dest, c.ddpstring) // so that it is freed later
	c.latestIsTemp = true
	return ast.VisitRecurse
}

func (c *compiler) VisitListLit(e *ast.ListLit) ast.VisitResult {
	listType := c.toIrType(e.Type).(*ddpIrListType)
	list := c.NewAlloca(listType.IrType())

	// get the listLen as irValue
	var listLen value.Value = zero
	if e.Values != nil {
		listLen = newInt(int64(len(e.Values)))
	} else if e.Count != nil && e.Value != nil {
		listLen, _, _ = c.evaluate(e.Count)
	} else { // empty list
		c.cbb.NewStore(listType.DefaultValue(), list)
		c.latestReturn, c.latestReturnType = c.scp.addTemporary(list, listType)
		c.latestIsTemp = true
		return ast.VisitRecurse
	}

	// create a empty list of the correct length
	c.cbb.NewCall(listType.fromConstantsIrFun, list, listLen)

	listArr := c.loadStructField(list, list_arr_field_index) // load the array

	if e.Values != nil { // we got some values to copy
		// evaluate every value and copy it into the array
		for i, v := range e.Values {
			val, valTyp, isTemp := c.evaluate(v)
			elementPtr := c.indexArray(listArr, newInt(int64(i)))
			c.claimOrCopy(elementPtr, val, valTyp, isTemp)
		}
	} else if e.Count != nil && e.Value != nil { // single Value multiple times
		val, _, _ := c.evaluate(e.Value) // if val is a temporary, it is freed automatically

		c.createFor(zero, c.forDefaultCond(listLen), func(index value.Value) {
			elementPtr := c.indexArray(listArr, index)
			if listType.elementType.IsPrimitive() {
				c.cbb.NewStore(val, elementPtr)
			} else {
				c.deepCopyInto(elementPtr, val, listType.elementType)
			}
		})
	}
	c.latestReturn, c.latestReturnType = c.scp.addTemporary(list, listType)
	c.latestIsTemp = true
	return ast.VisitRecurse
}

func (c *compiler) VisitUnaryExpr(e *ast.UnaryExpr) ast.VisitResult {
	if e.OverloadedBy != nil {
		return c.VisitFuncCall(&ast.FuncCall{
			Range: e.GetRange(),
			Tok:   e.Tok,
			Name:  e.OverloadedBy.Decl.Name(),
			Func:  e.OverloadedBy.Decl,
			Args:  e.OverloadedBy.Args,
		})
	}

	rhs, typ, _ := c.evaluate(e.Rhs) // compile the expression onto which the operator is applied

	// big switches for the different type combinations
	c.commentNode(c.cbb, e, e.Operator.String())
	switch e.Operator {
	case ast.UN_ABS:
		switch typ {
		case c.ddpfloattyp:
			// c.latestReturn = rhs < 0 ? 0 - rhs : rhs;
			c.latestReturn = c.createTernary(c.cbb.NewFCmp(enum.FPredOLT, rhs, zerof),
				func() value.Value { return c.cbb.NewFSub(zerof, rhs) },
				func() value.Value { return rhs },
			)
			c.latestReturnType = c.ddpfloattyp
		case c.ddpinttyp:
			// c.latestReturn = rhs < 0 ? 0 - rhs : rhs;
			c.latestReturn = c.createTernary(c.cbb.NewICmp(enum.IPredSLT, rhs, zero),
				func() value.Value { return c.cbb.NewSub(zero, rhs) },
				func() value.Value { return rhs },
			)
			c.latestReturnType = c.ddpinttyp
		default:
			c.err("invalid Parameter Type for BETRAG: %s", typ.Name())
		}
	case ast.UN_NEGATE:
		switch typ {
		case c.ddpfloattyp:
			c.latestReturn = c.cbb.NewFNeg(rhs)
			c.latestReturnType = c.ddpfloattyp
		case c.ddpinttyp:
			c.latestReturn = c.cbb.NewSub(zero, rhs)
			c.latestReturnType = c.ddpinttyp
		default:
			c.err("invalid Parameter Type for NEGATE: %s", typ.Name())
		}
	case ast.UN_NOT:
		c.latestReturn = c.cbb.NewXor(rhs, newInt(1))
		c.latestReturnType = c.ddpbooltyp
	case ast.UN_LOGIC_NOT:
		c.latestReturn = c.cbb.NewXor(rhs, all_ones)
		c.latestReturnType = c.ddpinttyp
	case ast.UN_LEN:
		switch typ {
		case c.ddpstring:
			c.latestReturn = c.cbb.NewCall(c.ddpstring.lengthIrFun, rhs)
		default:
			if _, isList := typ.(*ddpIrListType); isList {
				c.latestReturn = c.loadStructField(rhs, list_len_field_index)
			} else {
				c.err("invalid Parameter Type for LÄNGE: %s", typ.Name())
			}
		}
		c.latestReturnType = c.ddpinttyp
	default:
		c.err("Unbekannter Operator '%s'", e.Operator)
	}
	return ast.VisitRecurse
}

func (c *compiler) VisitBinaryExpr(e *ast.BinaryExpr) ast.VisitResult {
	c.commentNode(c.cbb, e, e.Operator.String())

	if e.OverloadedBy != nil {
		return c.VisitFuncCall(&ast.FuncCall{
			Range: e.GetRange(),
			Tok:   e.Tok,
			Name:  e.OverloadedBy.Decl.Name(),
			Func:  e.OverloadedBy.Decl,
			Args:  e.OverloadedBy.Args,
		})
	}

	// for UND and ODER both operands are booleans, so we don't need to worry about memory management
	// for BIN_FIELD_ACCESS we don't want to evaluate Lhs, as it is just the field name
	switch e.Operator {
	case ast.BIN_AND:
		lhs, _, _ := c.evaluate(e.Lhs)
		startBlock, trueBlock, leaveBlock := c.cbb, c.cf.NewBlock(""), c.cf.NewBlock("")
		c.commentNode(c.cbb, e, e.Operator.String())
		c.cbb.NewCondBr(lhs, trueBlock, leaveBlock)

		c.cbb = trueBlock
		// collect temporaries because of possible short-circuiting
		c.scp = newScope(c.scp)
		rhs, _, _ := c.evaluate(e.Rhs)
		// free temporaries
		c.scp = c.exitScope(c.scp)
		c.commentNode(c.cbb, e, e.Operator.String())
		c.cbb.NewBr(leaveBlock)
		trueBlock = c.cbb

		c.cbb = leaveBlock
		c.commentNode(c.cbb, e, e.Operator.String())
		c.latestReturn = c.cbb.NewPhi(ir.NewIncoming(rhs, trueBlock), ir.NewIncoming(lhs, startBlock))
		c.latestReturnType = c.ddpbooltyp
		return ast.VisitRecurse
	case ast.BIN_OR:
		lhs, _, _ := c.evaluate(e.Lhs)
		startBlock, falseBlock, leaveBlock := c.cbb, c.cf.NewBlock(""), c.cf.NewBlock("")
		c.commentNode(c.cbb, e, e.Operator.String())
		c.cbb.NewCondBr(lhs, leaveBlock, falseBlock)

		c.cbb = falseBlock
		// collect temporaries because of possible short-circuiting
		c.scp = newScope(c.scp)
		rhs, _, _ := c.evaluate(e.Rhs)
		// free temporaries
		c.scp = c.exitScope(c.scp)
		c.commentNode(c.cbb, e, e.Operator.String())
		c.cbb.NewBr(leaveBlock)
		falseBlock = c.cbb // in case c.evaluate has multiple blocks

		c.cbb = leaveBlock
		c.commentNode(c.cbb, e, e.Operator.String())
		c.latestReturn = c.cbb.NewPhi(ir.NewIncoming(lhs, startBlock), ir.NewIncoming(rhs, falseBlock))
		c.latestReturnType = c.ddpbooltyp
		return ast.VisitRecurse
	case ast.BIN_FIELD_ACCESS:
		rhs, rhsTyp, rhsIsTemp := c.evaluate(e.Rhs)
		if structType, isStruct := rhsTyp.(*ddpIrStructType); isStruct {
			fieldIndex := getFieldIndex(e.Lhs.Token().Literal, structType)
			fieldType := structType.fieldIrTypes[fieldIndex]
			fieldPtr := c.indexStruct(rhs, fieldIndex)
			if fieldType.IsPrimitive() {
				c.latestReturn = c.cbb.NewLoad(fieldType.IrType(), fieldPtr)
			} else if !rhsIsTemp {
				c.latestReturn, c.latestIsTemp = fieldPtr, false
			} else {
				dest := c.NewAlloca(fieldType.IrType())
				c.cbb.NewStore(c.cbb.NewLoad(fieldType.IrType(), fieldPtr), dest)
				c.cbb.NewStore(fieldType.DefaultValue(), fieldPtr)
				c.latestReturn, c.latestReturnType = c.scp.addTemporary(dest, fieldType)
				c.latestIsTemp = true
			}
			c.latestReturnType = fieldType
		} else {
			c.err("invalid Parameter Types for VON (%s)", rhsTyp.Name())
		}
		return ast.VisitRecurse
	}

	// compile the two expressions onto which the operator is applied
	lhs, lhsTyp, isTempLhs := c.evaluate(e.Lhs)
	rhs, rhsTyp, isTempRhs := c.evaluate(e.Rhs)
	// big switches on the different type combinations
	switch e.Operator {
	case ast.BIN_XOR:
		c.latestReturn = c.cbb.NewXor(lhs, rhs)
		c.latestReturnType = c.ddpbooltyp
	case ast.BIN_CONCAT:
		var (
			result    value.Value
			resultTyp ddpIrType
			claimsLhs bool
			claimsRhs bool
		)

		lhsListTyp, lhsIsList := lhsTyp.(*ddpIrListType)
		rhsListTyp, rhsIsList := rhsTyp.(*ddpIrListType)

		if lhsIsList {
			resultTyp = lhsListTyp
		} else if rhsIsList {
			resultTyp = rhsListTyp
		} else {
			if lhsTyp == c.ddpstring && !rhsIsList ||
				rhsTyp == c.ddpstring && !lhsIsList {
				resultTyp = c.ddpstring
			} else {
				resultTyp = c.getListType(lhsTyp)
			}
		}
		result = c.NewAlloca(resultTyp.IrType())

		// string concatenations
		var concat_func *ir.Func = nil
		if lhsTyp == c.ddpstring && rhsTyp == c.ddpstring {
			concat_func = c.ddpstring.str_str_concat_IrFunc
			claimsLhs, claimsRhs = true, false
		} else if lhsTyp == c.ddpstring && rhsTyp == c.ddpchartyp {
			concat_func = c.ddpstring.str_char_concat_IrFunc
			claimsLhs, claimsRhs = true, false
		} else if lhsTyp == c.ddpchartyp && rhsTyp == c.ddpstring {
			concat_func = c.ddpstring.char_str_concat_IrFunc
			claimsLhs, claimsRhs = false, true
		}

		// list concatenations
		if concat_func == nil {
			if lhsIsList && rhsIsList {
				concat_func = lhsListTyp.list_list_concat_IrFunc
				claimsLhs, claimsRhs = true, false
			} else if lhsIsList && !rhsIsList {
				concat_func = lhsListTyp.list_scalar_concat_IrFunc
				claimsLhs, claimsRhs = true, false
			} else if !lhsIsList && !rhsIsList {
				concat_func = c.getListType(lhsTyp).scalar_scalar_concat_IrFunc
				claimsLhs, claimsRhs = false, false
			} else if !lhsIsList && rhsIsList {
				concat_func = rhsListTyp.scalar_list_concat_IrFunc
				claimsLhs, claimsRhs = false, true
			}
		}

		// the concat functions use the buffer of some of their arguments
		// if those arguments aren't temporaries, we copy them
		//
		// the concat function is also required to free the memory of the claimed
		// arguments or claim their memory for the result, so we do not have to free them
		if claimsLhs && !isTempLhs {
			dest := c.NewAlloca(lhsTyp.IrType())
			lhs = c.deepCopyInto(dest, lhs, lhsTyp)
		}
		if claimsRhs && !isTempRhs {
			dest := c.NewAlloca(rhsTyp.IrType())
			rhs = c.deepCopyInto(dest, rhs, rhsTyp)
		}

		c.cbb.NewCall(concat_func, result, lhs, rhs)
		c.latestReturn, c.latestReturnType = c.scp.addTemporary(result, resultTyp)
		c.latestIsTemp = true
	case ast.BIN_PLUS:
		switch lhsTyp {
		case c.ddpinttyp:
			switch rhsTyp {
			case c.ddpinttyp:
				c.latestReturn = c.cbb.NewAdd(lhs, rhs)
				c.latestReturnType = c.ddpinttyp
			case c.ddpfloattyp:
				fp := c.cbb.NewSIToFP(lhs, ddpfloat)
				c.latestReturn = c.cbb.NewFAdd(fp, rhs)
				c.latestReturnType = c.ddpfloattyp
			default:
				c.err("invalid Parameter Types for PLUS (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		case c.ddpfloattyp:
			switch rhsTyp {
			case c.ddpinttyp:
				fp := c.cbb.NewSIToFP(rhs, ddpfloat)
				c.latestReturn = c.cbb.NewFAdd(lhs, fp)
			case c.ddpfloattyp:
				c.latestReturn = c.cbb.NewFAdd(lhs, rhs)
			default:
				c.err("invalid Parameter Types for PLUS (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
			c.latestReturnType = c.ddpfloattyp
		default:
			c.err("invalid Parameter Types for PLUS (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
		}
	case ast.BIN_MINUS:
		switch lhsTyp {
		case c.ddpinttyp:
			switch rhsTyp {
			case c.ddpinttyp:
				c.latestReturn = c.cbb.NewSub(lhs, rhs)
				c.latestReturnType = c.ddpinttyp
			case c.ddpfloattyp:
				fp := c.cbb.NewSIToFP(lhs, ddpfloat)
				c.latestReturn = c.cbb.NewFSub(fp, rhs)
				c.latestReturnType = c.ddpfloattyp
			default:
				c.err("invalid Parameter Types for MINUS (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		case c.ddpfloattyp:
			switch rhsTyp {
			case c.ddpinttyp:
				fp := c.cbb.NewSIToFP(rhs, ddpfloat)
				c.latestReturn = c.cbb.NewFSub(lhs, fp)
			case c.ddpfloattyp:
				c.latestReturn = c.cbb.NewFSub(lhs, rhs)
			default:
				c.err("invalid Parameter Types for MINUS (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
			c.latestReturnType = c.ddpfloattyp
		default:
			c.err("invalid Parameter Types for MINUS (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
		}
	case ast.BIN_MULT:
		switch lhsTyp {
		case c.ddpinttyp:
			switch rhsTyp {
			case c.ddpinttyp:
				c.latestReturn = c.cbb.NewMul(lhs, rhs)
				c.latestReturnType = c.ddpinttyp
			case c.ddpfloattyp:
				fp := c.cbb.NewSIToFP(lhs, ddpfloat)
				c.latestReturn = c.cbb.NewFMul(fp, rhs)
				c.latestReturnType = c.ddpfloattyp
			default:
				c.err("invalid Parameter Types for MAL (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		case c.ddpfloattyp:
			switch rhsTyp {
			case c.ddpinttyp:
				fp := c.cbb.NewSIToFP(rhs, ddpfloat)
				c.latestReturn = c.cbb.NewFMul(lhs, fp)
			case c.ddpfloattyp:
				c.latestReturn = c.cbb.NewFMul(lhs, rhs)
			default:
				c.err("invalid Parameter Types for MAL (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
			c.latestReturnType = c.ddpfloattyp
		default:
			c.err("invalid Parameter Types for MAL (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
		}
	case ast.BIN_DIV:
		switch lhsTyp {
		case c.ddpinttyp:
			switch rhsTyp {
			case c.ddpinttyp:
				lhs = c.cbb.NewSIToFP(lhs, ddpfloat)
				rhs = c.cbb.NewSIToFP(rhs, ddpfloat)
				c.latestReturn = c.cbb.NewFDiv(lhs, rhs)
			case c.ddpfloattyp:
				fp := c.cbb.NewSIToFP(lhs, ddpfloat)
				c.latestReturn = c.cbb.NewFDiv(fp, rhs)
			default:
				c.err("invalid Parameter Types for DURCH (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		case c.ddpfloattyp:
			switch rhsTyp {
			case c.ddpinttyp:
				fp := c.cbb.NewSIToFP(rhs, ddpfloat)
				c.latestReturn = c.cbb.NewFDiv(lhs, fp)
			case c.ddpfloattyp:
				c.latestReturn = c.cbb.NewFDiv(lhs, rhs)
			default:
				c.err("invalid Parameter Types for DURCH (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		default:
			c.err("invalid Parameter Types for DURCH (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
		}
		c.latestReturnType = c.ddpfloattyp
	case ast.BIN_INDEX:
		switch lhsTyp {
		case c.ddpstring:
			c.latestReturn = c.cbb.NewCall(c.ddpstring.indexIrFun, lhs, rhs)
			c.latestReturnType = c.ddpchartyp
		default:
			if listType, isList := lhsTyp.(*ddpIrListType); isList {
				listLen := c.loadStructField(lhs, list_len_field_index)
				index := c.cbb.NewSub(rhs, newInt(1)) // ddp indices start at 1, so subtract 1
				// index bounds check
				cond := c.cbb.NewAnd(c.cbb.NewICmp(enum.IPredSLT, index, listLen), c.cbb.NewICmp(enum.IPredSGE, index, zero))
				c.createIfElse(cond, func() {
					listArr := c.loadStructField(lhs, list_arr_field_index)
					elementPtr := c.indexArray(listArr, index)
					// if the list is a temporary, we need to copy the element
					if isTempLhs {
						if listType.elementType.IsPrimitive() { // primitives are simply loaded
							c.latestReturn = c.cbb.NewLoad(listType.elementType.IrType(), elementPtr)
						} else {
							dest := c.NewAlloca(listType.elementType.IrType())
							c.latestReturn, c.latestReturnType = c.scp.addTemporary(
								c.deepCopyInto(dest, elementPtr, listType.elementType),
								listType.elementType,
							)
							c.latestIsTemp = true // the element is now also a temporary
						}
					} else {
						if listType.elementType.IsPrimitive() {
							c.latestReturn = c.cbb.NewLoad(listType.elementType.IrType(), elementPtr)
						} else { // the list is not temporary, so a reference to the element is enough
							c.latestReturn = elementPtr
							c.latestIsTemp = false
						}
					}
				}, func() { // runtime error
					line, column := int64(e.Token().Range.Start.Line), int64(e.Token().Range.Start.Column)
					c.out_of_bounds_error(newInt(line), newInt(column), rhs, listLen)
				})
				c.latestReturnType = listType.elementType
			} else {
				c.err("invalid Parameter Types for STELLE (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		}
	case ast.BIN_SLICE_FROM, ast.BIN_SLICE_TO:
		dest := c.NewAlloca(lhsTyp.IrType())

		switch lhsTyp {
		case c.ddpstring:
			if e.Operator == ast.BIN_SLICE_FROM {
				str_len := c.cbb.NewCall(c.ddpstring.lengthIrFun, lhs)
				c.cbb.NewCall(c.ddpstring.sliceIrFun, dest, lhs, rhs, str_len)
			} else {
				c.cbb.NewCall(c.ddpstring.sliceIrFun, dest, lhs, newInt(1), rhs)
			}
		default:
			if listTyp, isList := lhsTyp.(*ddpIrListType); isList {
				if e.Operator == ast.BIN_SLICE_FROM {
					lst_len := c.loadStructField(lhs, list_len_field_index)
					c.cbb.NewCall(listTyp.sliceIrFun, dest, lhs, rhs, lst_len)
				} else {
					c.cbb.NewCall(listTyp.sliceIrFun, dest, lhs, newInt(1), rhs)
				}
			} else {
				c.err("invalid Parameter Types for %s (%s, %s)", e.Operator.String(), lhsTyp.Name(), rhsTyp.Name())
			}
		}
		c.latestReturn, c.latestReturnType = c.scp.addTemporary(dest, lhsTyp)
		c.latestIsTemp = true
	case ast.BIN_POW:
		switch lhsTyp {
		case c.ddpinttyp:
			switch rhsTyp {
			case c.ddpinttyp:
				lhs = c.cbb.NewSIToFP(lhs, ddpfloat)
				rhs = c.cbb.NewSIToFP(rhs, ddpfloat)
			case c.ddpfloattyp:
				lhs = c.cbb.NewSIToFP(lhs, ddpfloat)
			default:
				c.err("invalid Parameter Types for HOCH (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		case c.ddpfloattyp:
			switch rhsTyp {
			case c.ddpinttyp:
				rhs = c.cbb.NewSIToFP(rhs, ddpfloat)
			}
		default:
			c.err("invalid Parameter Types for HOCH (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
		}
		c.latestReturn = c.cbb.NewCall(c.functions["pow"].irFunc, lhs, rhs)
		c.latestReturnType = c.ddpfloattyp
	case ast.BIN_LOG:
		switch lhsTyp {
		case c.ddpinttyp:
			switch rhsTyp {
			case c.ddpinttyp:
				lhs = c.cbb.NewSIToFP(lhs, ddpfloat)
				rhs = c.cbb.NewSIToFP(rhs, ddpfloat)
			case c.ddpfloattyp:
				lhs = c.cbb.NewSIToFP(lhs, ddpfloat)
			default:
				c.err("invalid Parameter Types for LOGARITHMUS (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		case c.ddpfloattyp:
			switch rhsTyp {
			case c.ddpinttyp:
				rhs = c.cbb.NewSIToFP(rhs, ddpfloat)
			}
		default:
			c.err("invalid Parameter Types for LOGARITHMUS (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
		}
		log10_num := c.cbb.NewCall(c.functions["log10"].irFunc, lhs)
		log10_base := c.cbb.NewCall(c.functions["log10"].irFunc, rhs)
		c.latestReturn = c.cbb.NewFDiv(log10_num, log10_base)
		c.latestReturnType = c.ddpfloattyp
	case ast.BIN_LOGIC_AND:
		c.latestReturn = c.cbb.NewAnd(lhs, rhs)
		c.latestReturnType = c.ddpinttyp
	case ast.BIN_LOGIC_OR:
		c.latestReturn = c.cbb.NewOr(lhs, rhs)
		c.latestReturnType = c.ddpinttyp
	case ast.BIN_LOGIC_XOR:
		c.latestReturn = c.cbb.NewXor(lhs, rhs)
		c.latestReturnType = c.ddpinttyp
	case ast.BIN_MOD:
		c.latestReturn = c.cbb.NewSRem(lhs, rhs)
		c.latestReturnType = c.ddpinttyp
	case ast.BIN_LEFT_SHIFT:
		c.latestReturn = c.cbb.NewShl(lhs, rhs)
		c.latestReturnType = c.ddpinttyp
	case ast.BIN_RIGHT_SHIFT:
		c.latestReturn = c.cbb.NewLShr(lhs, rhs)
		c.latestReturnType = c.ddpinttyp
	case ast.BIN_EQUAL:
		c.compare_values(lhs, rhs, lhsTyp)
	case ast.BIN_UNEQUAL:
		equal := c.compare_values(lhs, rhs, lhsTyp)
		c.latestReturn = c.cbb.NewXor(equal, newInt(1))
	case ast.BIN_LESS:
		switch lhsTyp {
		case c.ddpinttyp:
			switch rhsTyp {
			case c.ddpinttyp:
				c.latestReturn = c.cbb.NewICmp(enum.IPredSLT, lhs, rhs)
			case c.ddpfloattyp:
				fp := c.cbb.NewSIToFP(lhs, ddpfloat)
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOLT, fp, rhs)
			default:
				c.err("invalid Parameter Types for KLEINER (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		case c.ddpfloattyp:
			switch rhsTyp {
			case c.ddpinttyp:
				fp := c.cbb.NewSIToFP(rhs, ddpfloat)
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOLT, lhs, fp)
			case c.ddpfloattyp:
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOLT, lhs, rhs)
			default:
				c.err("invalid Parameter Types for KLEINER (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		}
		c.latestReturnType = c.ddpbooltyp
	case ast.BIN_LESS_EQ:
		switch lhsTyp {
		case c.ddpinttyp:
			switch rhsTyp {
			case c.ddpinttyp:
				c.latestReturn = c.cbb.NewICmp(enum.IPredSLE, lhs, rhs)
			case c.ddpfloattyp:
				fp := c.cbb.NewSIToFP(lhs, ddpfloat)
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOLE, fp, rhs)
			default:
				c.err("invalid Parameter Types for KLEINERODER (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		case c.ddpfloattyp:
			switch rhsTyp {
			case c.ddpinttyp:
				fp := c.cbb.NewSIToFP(rhs, ddpfloat)
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOLE, lhs, fp)
			case c.ddpfloattyp:
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOLE, lhs, rhs)
			default:
				c.err("invalid Parameter Types for KLEINERODER (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		default:
			c.err("invalid Parameter Types for KLEINERODER (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
		}
		c.latestReturnType = c.ddpbooltyp
	case ast.BIN_GREATER:
		switch lhsTyp {
		case c.ddpinttyp:
			switch rhsTyp {
			case c.ddpinttyp:
				c.latestReturn = c.cbb.NewICmp(enum.IPredSGT, lhs, rhs)
			case c.ddpfloattyp:
				fp := c.cbb.NewSIToFP(lhs, ddpfloat)
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOGT, fp, rhs)
			default:
				c.err("invalid Parameter Types for GRÖßER (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		case c.ddpfloattyp:
			switch rhsTyp {
			case c.ddpinttyp:
				fp := c.cbb.NewSIToFP(rhs, ddpfloat)
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOGT, lhs, fp)
			case c.ddpfloattyp:
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOGT, lhs, rhs)
			default:
				c.err("invalid Parameter Types for GRÖßER (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		default:
			c.err("invalid Parameter Types for GRÖßER (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
		}
		c.latestReturnType = c.ddpbooltyp
	case ast.BIN_GREATER_EQ:
		switch lhsTyp {
		case c.ddpinttyp:
			switch rhsTyp {
			case c.ddpinttyp:
				c.latestReturn = c.cbb.NewICmp(enum.IPredSGE, lhs, rhs)
			case c.ddpfloattyp:
				fp := c.cbb.NewSIToFP(lhs, ddpfloat)
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOGE, fp, rhs)
			default:
				c.err("invalid Parameter Types for GRÖßERODER (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		case c.ddpfloattyp:
			switch rhsTyp {
			case c.ddpinttyp:
				fp := c.cbb.NewSIToFP(rhs, ddpfloat)
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOGE, lhs, fp)
			case c.ddpfloattyp:
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOGE, lhs, rhs)
			default:
				c.err("invalid Parameter Types for GRÖßERODER (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		default:
			c.err("invalid Parameter Types for GRÖßERODER (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
		}
		c.latestReturnType = c.ddpbooltyp
	}
	return ast.VisitRecurse
}

func (c *compiler) VisitTernaryExpr(e *ast.TernaryExpr) ast.VisitResult {
	if e.OverloadedBy != nil {
		return c.VisitFuncCall(&ast.FuncCall{
			Range: e.GetRange(),
			Tok:   e.Tok,
			Name:  e.OverloadedBy.Decl.Name(),
			Func:  e.OverloadedBy.Decl,
			Args:  e.OverloadedBy.Args,
		})
	}

	// if due to short circuiting
	if e.Operator == ast.TER_FALLS {
		mid, _, _ := c.evaluate(e.Mid)
		trueBlock, falseBlock, leaveBlock := c.cf.NewBlock(""), c.cf.NewBlock(""), c.cf.NewBlock("")
		c.commentNode(c.cbb, e, e.Operator.String())
		c.cbb.NewCondBr(mid, trueBlock, falseBlock)

		c.cbb = trueBlock
		// collect temporaries because of possible short-circuiting
		c.scp = newScope(c.scp)
		lhs, lhsTyp, lhsIsTemp := c.evaluate(e.Lhs)
		// claim the temporary, as the phi instruction will become the actual temporary
		if lhsIsTemp && !lhsTyp.IsPrimitive() {
			lhs = c.scp.claimTemporary(lhs)
		}
		// free temporaries
		c.scp = c.exitScope(c.scp)
		c.commentNode(c.cbb, e, e.Operator.String())
		c.cbb.NewBr(leaveBlock)
		trueBlock = c.cbb

		c.cbb = falseBlock
		// collect temporaries because of possible short-circuiting
		c.scp = newScope(c.scp)
		rhs, rhsTyp, rhsIsTemp := c.evaluate(e.Rhs)
		// claim the temporary, as the phi instruction will become the actual temporary
		if rhsIsTemp && !rhsTyp.IsPrimitive() {
			rhs = c.scp.claimTemporary(rhs)
		}
		// free temporaries
		c.scp = c.exitScope(c.scp)
		c.commentNode(c.cbb, e, e.Operator.String())
		c.cbb.NewBr(leaveBlock)
		falseBlock = c.cbb

		// simple case, where both can be treated the same way
		if lhsIsTemp == rhsIsTemp {
			c.latestIsTemp = lhsIsTemp
		} else {
			c.latestIsTemp = true

			// we need to copy the non-temp value to be sure
			c.cbb = trueBlock
			if lhsIsTemp {
				c.cbb = falseBlock
			}

			// turn the non-temp into a temporary and claim the temporary,
			// as the phi instruction will become the actual temporary
			dest := c.NewAlloca(lhsTyp.IrType())
			if lhsIsTemp {
				rhs = c.deepCopyInto(dest, rhs, lhsTyp)
			} else {
				lhs = c.deepCopyInto(dest, lhs, lhsTyp)
			}
		}

		c.cbb = leaveBlock
		c.commentNode(c.cbb, e, e.Operator.String())
		c.latestReturn = c.cbb.NewPhi(ir.NewIncoming(lhs, trueBlock), ir.NewIncoming(rhs, falseBlock))
		if c.latestIsTemp {
			c.scp.addTemporary(c.latestReturn, lhsTyp)
		}
		c.latestReturnType = lhsTyp
		return ast.VisitRecurse
	}

	lhs, lhsTyp, _ := c.evaluate(e.Lhs)
	mid, midTyp, _ := c.evaluate(e.Mid)
	rhs, rhsTyp, _ := c.evaluate(e.Rhs)

	switch e.Operator {
	case ast.TER_SLICE:
		dest := c.NewAlloca(lhsTyp.IrType())
		switch lhsTyp {
		case c.ddpstring:
			c.cbb.NewCall(c.ddpstring.sliceIrFun, dest, lhs, mid, rhs)
		default:
			if listTyp, isList := lhsTyp.(*ddpIrListType); isList {
				c.cbb.NewCall(listTyp.sliceIrFun, dest, lhs, mid, rhs)
			} else {
				c.err("invalid Parameter Types for VONBIS (%s, %s, %s)", lhsTyp.Name(), midTyp.Name(), rhsTyp.Name())
			}
		}
		c.latestReturn, c.latestReturnType = c.scp.addTemporary(dest, lhsTyp)
		c.latestIsTemp = true
	case ast.TER_BETWEEN:
		switch lhsTyp {
		case c.ddpinttyp:
			switch rhsTyp {
			case c.ddpinttyp:
				switch midTyp {
				case c.ddpinttyp:
					c.latestReturn = c.cbb.NewOr(
						c.cbb.NewAnd(c.cbb.NewICmp(enum.IPredSGT, lhs, rhs), c.cbb.NewICmp(enum.IPredSLT, lhs, mid)),
						c.cbb.NewAnd(c.cbb.NewICmp(enum.IPredSGT, lhs, mid), c.cbb.NewICmp(enum.IPredSLT, lhs, rhs)),
					)
				case c.ddpfloattyp:
					fLhs := c.cbb.NewSIToFP(lhs, ddpfloat)
					fRhs := c.cbb.NewSIToFP(rhs, ddpfloat)
					c.latestReturn = c.cbb.NewOr(
						c.cbb.NewAnd(c.cbb.NewFCmp(enum.FPredOGT, fLhs, fRhs), c.cbb.NewFCmp(enum.FPredOLT, fLhs, mid)),
						c.cbb.NewAnd(c.cbb.NewFCmp(enum.FPredOGT, fLhs, mid), c.cbb.NewFCmp(enum.FPredOLT, fLhs, fRhs)),
					)
				default:
					c.err("invalid Parameter Types for ZWISCHEN (%s, %s, %s)", lhsTyp.Name(), midTyp.Name(), rhsTyp.Name())
				}
			case c.ddpfloattyp:
				switch midTyp {
				case c.ddpinttyp:
					fLhs := c.cbb.NewSIToFP(lhs, ddpfloat)
					fMid := c.cbb.NewSIToFP(mid, ddpfloat)
					c.latestReturn = c.cbb.NewOr(
						c.cbb.NewAnd(c.cbb.NewFCmp(enum.FPredOGT, fLhs, rhs), c.cbb.NewFCmp(enum.FPredOLT, fLhs, fMid)),
						c.cbb.NewAnd(c.cbb.NewFCmp(enum.FPredOGT, fLhs, fMid), c.cbb.NewFCmp(enum.FPredOLT, fLhs, rhs)),
					)
				case c.ddpfloattyp:
					fLhs := c.cbb.NewSIToFP(lhs, ddpfloat)
					fRhs := c.cbb.NewSIToFP(rhs, ddpfloat)
					c.latestReturn = c.cbb.NewOr(
						c.cbb.NewAnd(c.cbb.NewFCmp(enum.FPredOGT, fLhs, fRhs), c.cbb.NewFCmp(enum.FPredOLT, fLhs, mid)),
						c.cbb.NewAnd(c.cbb.NewFCmp(enum.FPredOGT, fLhs, mid), c.cbb.NewFCmp(enum.FPredOLT, fLhs, fRhs)),
					)
				default:
					c.err("invalid Parameter Types for ZWISCHEN (%s, %s, %s)", lhsTyp.Name(), midTyp.Name(), rhsTyp.Name())
				}
			default:
				c.err("invalid Parameter Types for ZWISCHEN (%s, %s, %s)", lhsTyp.Name(), midTyp.Name(), rhsTyp.Name())
			}
		case c.ddpfloattyp:
			switch rhsTyp {
			case c.ddpinttyp:
				switch midTyp {
				case c.ddpinttyp:
					fMid := c.cbb.NewSIToFP(mid, ddpfloat)
					fRhs := c.cbb.NewSIToFP(rhs, ddpfloat)
					c.latestReturn = c.cbb.NewOr(
						c.cbb.NewAnd(c.cbb.NewFCmp(enum.FPredOGT, lhs, fRhs), c.cbb.NewFCmp(enum.FPredOLT, lhs, fMid)),
						c.cbb.NewAnd(c.cbb.NewFCmp(enum.FPredOGT, lhs, fMid), c.cbb.NewFCmp(enum.FPredOLT, lhs, fRhs)),
					)
				case c.ddpfloattyp:
					fRhs := c.cbb.NewSIToFP(rhs, ddpfloat)
					c.latestReturn = c.cbb.NewOr(
						c.cbb.NewAnd(c.cbb.NewFCmp(enum.FPredOGT, lhs, fRhs), c.cbb.NewFCmp(enum.FPredOLT, lhs, mid)),
						c.cbb.NewAnd(c.cbb.NewFCmp(enum.FPredOGT, lhs, mid), c.cbb.NewFCmp(enum.FPredOLT, lhs, fRhs)),
					)
				default:
					c.err("invalid Parameter Types for ZWISCHEN (%s, %s, %s)", lhsTyp.Name(), midTyp.Name(), rhsTyp.Name())
				}
			case c.ddpfloattyp:
				switch midTyp {
				case c.ddpinttyp:
					fMid := c.cbb.NewSIToFP(mid, ddpfloat)
					c.latestReturn = c.cbb.NewOr(
						c.cbb.NewAnd(c.cbb.NewFCmp(enum.FPredOGT, lhs, rhs), c.cbb.NewFCmp(enum.FPredOLT, lhs, fMid)),
						c.cbb.NewAnd(c.cbb.NewFCmp(enum.FPredOGT, lhs, fMid), c.cbb.NewFCmp(enum.FPredOLT, lhs, rhs)),
					)
				case c.ddpfloattyp:
					c.latestReturn = c.cbb.NewOr(
						c.cbb.NewAnd(c.cbb.NewFCmp(enum.FPredOGT, lhs, rhs), c.cbb.NewFCmp(enum.FPredOLT, lhs, mid)),
						c.cbb.NewAnd(c.cbb.NewFCmp(enum.FPredOGT, lhs, mid), c.cbb.NewFCmp(enum.FPredOLT, lhs, rhs)),
					)
				default:
					c.err("invalid Parameter Types for ZWISCHEN (%s, %s, %s)", lhsTyp.Name(), midTyp.Name(), rhsTyp.Name())
				}
			default:
				c.err("invalid Parameter Types for ZWISCHEN (%s, %s, %s)", lhsTyp.Name(), midTyp.Name(), rhsTyp.Name())
			}
		default:
			c.err("invalid Parameter Types for ZWISCHEN (%s, %s, %s)", lhsTyp.Name(), midTyp.Name(), rhsTyp.Name())
		}
		c.latestReturnType = c.ddpbooltyp
	default:
		c.err("invalid Parameter Types for VONBIS (%s, %s, %s)", lhsTyp.Name(), midTyp.Name(), rhsTyp.Name())
	}
	return ast.VisitRecurse
}

func (c *compiler) VisitCastExpr(e *ast.CastExpr) ast.VisitResult {
	if e.OverloadedBy != nil {
		return c.VisitFuncCall(&ast.FuncCall{
			Range: e.GetRange(),
			Tok:   e.Token(),
			Name:  e.OverloadedBy.Decl.Name(),
			Func:  e.OverloadedBy.Decl,
			Args:  e.OverloadedBy.Args,
		})
	}

	targetType := ddptypes.TrueUnderlying(e.TargetType)
	lhs, lhsTyp, isTempLhs := c.evaluate(e.Lhs)

	vtable := c.toIrType(targetType).VTable()
	if typeDef, isTypeDef := ddptypes.CastTypeDef(e.TargetType); isTypeDef {
		vtable = c.typeDefVTables[c.mangledNameType(typeDef)]
	}

	// helper function to cast non-primitive from any to their concrete type
	nonPrimitiveAnyCast := func() {
		nonPrimTyp := c.toIrType(targetType)

		dest := c.NewAlloca(nonPrimTyp.IrType())
		c.createIfElse(c.compareAnyType(lhs, vtable), func() {
			// temporary values can be claimed
			if isTempLhs {
				val_ptr := c.loadAnyValuePtr(lhs, nonPrimTyp.IrType())
				val := c.cbb.NewLoad(nonPrimTyp.IrType(), val_ptr)
				c.cbb.NewStore(val, dest)
				c.createIfElse(c.isSmallAny(lhs), func() {}, func() {
					c.ddp_reallocate(val_ptr, newInt(int64(c.getTypeSize(nonPrimTyp))), zero)
				})
				c.scp.claimTemporary(lhs) // don't call free func on the now invalid any
			} else {
				// non-temporaries are simply deep copied
				c.deepCopyInto(dest, c.loadAnyValuePtr(lhs, nonPrimTyp.IrType()), nonPrimTyp)
			}
			c.latestReturn, c.latestReturnType = c.scp.addTemporary(dest, nonPrimTyp)
			c.latestIsTemp = true
		}, func() {
			line, column := int64(e.Token().Range.Start.Line), int64(e.Token().Range.Start.Column)
			c.runtime_error(1, c.bad_cast_error_string, newInt(line), newInt(column))
		})
	}

	if ddptypes.IsList(targetType) {
		if lhsTyp == c.ddpany {
			nonPrimitiveAnyCast()
			return ast.VisitRecurse
		}

		listType := c.getListType(lhsTyp)
		list := c.NewAlloca(listType.typ)
		c.cbb.NewCall(listType.fromConstantsIrFun, list, newInt(1))
		elementPtr := c.indexArray(c.loadStructField(list, list_arr_field_index), zero)
		c.claimOrCopy(elementPtr, lhs, lhsTyp, isTempLhs)
		c.latestReturn, c.latestReturnType = c.scp.addTemporary(list, listType)
		c.latestIsTemp = true
	} else {
		// helper function to cast primitive from any to their concrete type
		primitiveAnyCast := func(primTyp ddpIrType) {
			c.createIfElse(c.compareAnyType(lhs, vtable), func() {
				c.latestReturn = c.loadSmallAnyValue(lhs, primTyp.IrType())
				c.latestReturnType, c.latestIsTemp = primTyp, true
			}, func() {
				line, column := int64(e.Token().Range.Start.Line), int64(e.Token().Range.Start.Column)
				c.runtime_error(1, c.bad_cast_error_string, newInt(line), newInt(column))
			})
		}

		switch targetType {
		case ddptypes.ZAHL:
			switch lhsTyp {
			case c.ddpinttyp:
				c.latestReturn = lhs
			case c.ddpfloattyp:
				c.latestReturn = c.cbb.NewFPToSI(lhs, ddpint)
			case c.ddpbooltyp:
				cond := c.cbb.NewICmp(enum.IPredNE, lhs, zero)
				c.latestReturn = c.cbb.NewZExt(cond, ddpint)
			case c.ddpchartyp:
				c.latestReturn = c.cbb.NewSExt(lhs, ddpint)
			case c.ddpstring:
				c.latestReturn = c.cbb.NewCall(c.functions["ddp_string_to_int"].irFunc, lhs)
			case c.ddpany:
				primitiveAnyCast(c.ddpinttyp)
			default:
				c.err("invalid Parameter Type for ZAHL: %s", lhsTyp.Name())
			}
		case ddptypes.KOMMAZAHL:
			switch lhsTyp {
			case c.ddpinttyp:
				c.latestReturn = c.cbb.NewSIToFP(lhs, ddpfloat)
			case c.ddpfloattyp:
				c.latestReturn = lhs
			case c.ddpstring:
				c.latestReturn = c.cbb.NewCall(c.functions["ddp_string_to_float"].irFunc, lhs)
			case c.ddpany:
				primitiveAnyCast(c.ddpfloattyp)
			default:
				c.err("invalid Parameter Type for KOMMAZAHL: %s", lhsTyp.Name())
			}
		case ddptypes.WAHRHEITSWERT:
			switch lhsTyp {
			case c.ddpinttyp:
				c.latestReturn = c.cbb.NewICmp(enum.IPredNE, lhs, zero)
			case c.ddpbooltyp:
				c.latestReturn = lhs
			case c.ddpany:
				primitiveAnyCast(c.ddpbooltyp)
			default:
				c.err("invalid Parameter Type for WAHRHEITSWERT: %s", lhsTyp.Name())
			}
		case ddptypes.BUCHSTABE:
			switch lhsTyp {
			case c.ddpinttyp:
				c.latestReturn = c.cbb.NewTrunc(lhs, ddpchar)
			case c.ddpchartyp:
				c.latestReturn = lhs
			case c.ddpany:
				primitiveAnyCast(c.ddpchartyp)
			default:
				c.err("invalid Parameter Type for BUCHSTABE: %s", lhsTyp.Name())
			}
		case ddptypes.TEXT:
			if lhsTyp == c.ddpany {
				nonPrimitiveAnyCast()
				return ast.VisitRecurse
			}

			if lhsTyp == c.ddpstring {
				c.latestReturn = lhs
				c.latestReturnType = c.ddpstring
				c.latestIsTemp = isTempLhs
				return ast.VisitRecurse // don't free lhs
			}

			var to_string_func *ir.Func
			switch lhsTyp {
			case c.ddpinttyp:
				to_string_func = c.ddpstring.int_to_string_IrFun
			case c.ddpfloattyp:
				to_string_func = c.ddpstring.float_to_string_IrFun
			case c.ddpbooltyp:
				to_string_func = c.ddpstring.bool_to_string_IrFun
			case c.ddpchartyp:
				to_string_func = c.ddpstring.char_to_string_IrFun
			default:
				c.err("invalid Parameter Type for TEXT: %s", lhsTyp.Name())
			}
			dest := c.NewAlloca(c.ddpstring.typ)
			c.cbb.NewCall(to_string_func, dest, lhs)
			c.latestReturn, c.latestReturnType = c.scp.addTemporary(dest, c.ddpstring)
			c.latestIsTemp = true
		case ddptypes.VARIABLE:
			if lhsTyp == c.ddpany {
				break
			}

			c.latestReturn, c.latestReturnType, c.latestIsTemp = c.castNonAnyToAny(lhs, lhsTyp, isTempLhs, lhsTyp.VTable())
		default:
			if lhsTyp == c.ddpany {
				nonPrimitiveAnyCast()
				return ast.VisitRecurse
			}
			// this is now valid because of typedefs/typealiases
			// c.err("Invalide Typumwandlung zu %s (%s)", e.TargetType, targetType)
		}
	}
	c.latestReturnType = c.toIrType(targetType)
	return ast.VisitRecurse
}

func (c *compiler) VisitTypeOpExpr(e *ast.TypeOpExpr) ast.VisitResult {
	switch e.Operator {
	case ast.TYPE_SIZE:
		c.latestReturn = c.sizeof(c.toIrType(e.Rhs).IrType())
		c.latestReturnType = c.ddpinttyp
	case ast.TYPE_DEFAULT:
		switch t := ddptypes.TrueUnderlying(e.Rhs).(type) {
		case *ddptypes.StructType:
			result, resultType := c.evaluateStructLiteral(t, nil)
			c.latestReturn, c.latestReturnType = c.scp.addTemporary(result, resultType)
		default:
			irType := c.toIrType(e.Rhs)
			var defaultValue value.Value = irType.DefaultValue()
			if !irType.IsPrimitive() {
				dest := c.NewAlloca(irType.IrType())
				c.cbb.NewStore(defaultValue, dest)
				defaultValue = dest
			}
			c.latestReturn, c.latestReturnType = c.scp.addTemporary(defaultValue, irType)
		}
	default:
		c.err("invalid TypeOpExpr Operator: %d", e.Operator)
	}
	c.latestIsTemp = true
	return ast.VisitRecurse
}

func (c *compiler) VisitTypeCheck(e *ast.TypeCheck) ast.VisitResult {
	lhs, _, _ := c.evaluate(e.Lhs)

	vtable := c.toIrType(e.CheckType).VTable()
	if typeDef, isTypeDef := ddptypes.CastTypeDef(e.CheckType); isTypeDef {
		vtable = c.typeDefVTables[c.mangledNameType(typeDef)]
	}

	c.latestReturn = c.compareAnyType(lhs, vtable)
	c.latestReturnType = c.ddpbooltyp
	return ast.VisitRecurse
}

func (c *compiler) VisitGrouping(e *ast.Grouping) ast.VisitResult {
	e.Expr.Accept(c) // visit like a normal expression, grouping is just precedence stuff which has already been parsed
	return ast.VisitRecurse
}

// helper for VisitAssignStmt and VisitFuncCall
// if as_ref is true, the assignable is treated as a reference parameter and the third return value can be ignored
// if as_ref is false, the assignable is treated as the lhs in an AssignStmt and might be a string indexing
func (c *compiler) evaluateAssignableOrReference(ass ast.Assigneable, as_ref bool) (value.Value, ddpIrType, *ast.Indexing) {
	switch assign := ass.(type) {
	case *ast.Ident:
		Var := c.scp.lookupVar(assign.Declaration.(*ast.VarDecl))
		return Var.val, Var.typ, nil
	case *ast.Indexing:
		lhs, lhsTyp, _ := c.evaluateAssignableOrReference(assign.Lhs, as_ref) // get the (possibly nested) assignable
		if listTyp, isList := lhsTyp.(*ddpIrListType); isList {
			index, _, _ := c.evaluate(assign.Index)
			index = c.cbb.NewSub(index, newInt(1)) // ddpindices start at 1
			listLen := c.loadStructField(lhs, list_len_field_index)
			var elementPtr value.Value

			cond := c.cbb.NewAnd(c.cbb.NewICmp(enum.IPredSLT, index, listLen), c.cbb.NewICmp(enum.IPredSGE, index, zero))
			c.createIfElse(cond, func() {
				listArr := c.loadStructField(lhs, list_arr_field_index)
				elementPtr = c.indexArray(listArr, index)
			}, func() { // runtime error
				line, column := int64(assign.Token().Range.Start.Line), int64(assign.Token().Range.Start.Column)
				c.out_of_bounds_error(newInt(line), newInt(column), c.cbb.NewAdd(index, newInt(1)), listLen)
			})
			return elementPtr, listTyp.elementType, nil
		} else if !as_ref && lhsTyp == c.ddpstring {
			return lhs, lhsTyp, assign
		} else {
			c.err("non-list/string/struct type passed as assignable/reference")
		}
	case *ast.FieldAccess:
		rhs, rhsTyp, _ := c.evaluateAssignableOrReference(assign.Rhs, as_ref)
		if structTyp, isStruct := rhsTyp.(*ddpIrStructType); isStruct {
			fieldIndex := getFieldIndex(assign.Field.Literal.Literal, structTyp)
			fieldPtr := c.indexStruct(rhs, fieldIndex)
			return fieldPtr, structTyp.fieldIrTypes[fieldIndex], nil
		} else {
			c.err("non-struct type passed to FieldAccess")
		}
	}
	c.err("Invalid types in evaluateAssignableOrReference %s", ass)
	return nil, nil, nil
}

func (c *compiler) VisitFuncCall(e *ast.FuncCall) ast.VisitResult {
	mangledName := c.mangledNameDecl(e.Func)
	_, ok := c.functions[mangledName] // retreive the function (the resolver took care that it is present)
	needsInstantiation := !ok && ast.IsGenericInstantiation(e.Func)

	if needsInstantiation {
		c.VisitFuncDecl(e.Func)
	}

	// declare the function if it is imported
	// this is needed so that plain expressions from other modules
	// (i.e. struct literals) work
	if !needsInstantiation && e.Func.Module() != c.ddpModule {
		c.declareImportedFuncDecl(e.Func)
	}

	fun := c.functions[mangledName]

	args := make([]value.Value, 0, len(fun.funcDecl.Parameters)+1)

	meta := annotators.ConstFuncParamMeta{}
	if attachement, ok := e.Func.Module().Ast.GetMetadataByKind(fun.funcDecl, annotators.ConstFuncParamMetaKind); ok {
		meta = attachement.(annotators.ConstFuncParamMeta)
	}

	irReturnType := c.toIrType(fun.funcDecl.ReturnType)
	var ret value.Value
	if !irReturnType.IsPrimitive() {
		ret = c.NewAlloca(irReturnType.IrType())
		args = append(args, ret)
	}

	for _, param := range fun.funcDecl.Parameters {
		var val value.Value

		// differentiate between references and normal parameters
		if param.Type.IsReference {
			if assign, ok := e.Args[param.Name.Literal].(ast.Assigneable); ok {
				val, _, _ = c.evaluateAssignableOrReference(assign, true)
			} else {
				c.err("non-assignable passed as reference to %s", fun.funcDecl.Name())
			}
		} else {
			eval, valTyp, isTemp := c.evaluate(e.Args[param.Name.Literal]) // compile each argument for the function
			if valTyp.IsPrimitive() ||
				(!ast.IsExternFunc(fun.funcDecl) && c.optimizationLevel >= 2 && meta.IsConst[param.Name.Literal]) {
				val = eval
			} else { // function parameters need to be copied by the caller
				dest := c.NewAlloca(valTyp.IrType())
				c.claimOrCopy(dest, eval, valTyp, isTemp)
				val = dest // do not add it to the temporaries, as the callee will free it
			}
		}

		args = append(args, val) // add the value to the arguments
	}

	c.commentNode(c.cbb, e, "")
	// compile the actual function call
	if irReturnType.IsPrimitive() {
		c.latestReturn = c.cbb.NewCall(fun.irFunc, args...)
	} else {
		c.cbb.NewCall(fun.irFunc, args...)
		c.latestReturn, c.latestReturnType = c.scp.addTemporary(ret, irReturnType)
		c.latestIsTemp = true
	}
	c.latestReturnType = irReturnType

	// the arguments of external functions must be freed by the caller
	// normal functions free their parameters in their body
	if ast.IsExternFunc(fun.funcDecl) {
		for i, param := range fun.funcDecl.Parameters {
			if !param.Type.IsReference {
				if irReturnType.IsPrimitive() {
					c.freeNonPrimitive(args[i], c.toIrType(param.Type.Type))
				} else {
					c.freeNonPrimitive(args[i+1], c.toIrType(param.Type.Type))
				}
			}
		}
	}
	return ast.VisitRecurse
}

func (c *compiler) evaluateStructLiteral(structType *ddptypes.StructType, args map[string]ast.Expression) (value.Value, ddpIrType) {
	// search in the types module for the decl, as it might not be present in this module due to transitive dependencies
	structDeclInterface, _, _ := c.typeMap[structType].Ast.Symbols.LookupDecl(structType.Name)
	structDecl := structDeclInterface.(*ast.StructDecl)
	resultType := c.toIrType(structType)
	result := c.NewAlloca(resultType.IrType())
	for i, field := range structType.Fields {
		fieldDecl := structDecl.Fields[i].(*ast.VarDecl)
		initType := fieldDecl.InitType
		argExpr := fieldDecl.InitVal
		if fieldArg, hasArg := args[field.Name]; hasArg {
			// the arg was passed so use that instead
			argExpr = fieldArg
		}

		// if no default value was given
		if argExpr == nil {
			argExpr = &ast.TypeOpExpr{Operator: ast.TYPE_DEFAULT, Rhs: fieldDecl.Type}
		}

		argVal, argType, isTempArg := c.evaluate(argExpr)

		// implicit cast to any if required
		if ddptypes.DeepEqual(field.Type, ddptypes.VARIABLE) && argType != c.ddpany {
			vtable := argType.VTable()
			if typeDef, isTypeDef := ddptypes.CastTypeDef(initType); isTypeDef {
				vtable = c.typeDefVTables[c.mangledNameType(typeDef)]
			}

			argVal, argType, isTempArg = c.castNonAnyToAny(argVal, argType, isTempArg, vtable)
		}

		c.claimOrCopy(c.indexStruct(result, int64(i)), argVal, argType, isTempArg)
	}
	return result, resultType
}

func (c *compiler) VisitStructLiteral(expr *ast.StructLiteral) ast.VisitResult {
	result, resultType := c.evaluateStructLiteral(expr.Struct.Type.(*ddptypes.StructType), expr.Args)
	c.latestReturn, c.latestReturnType = c.scp.addTemporary(result, resultType)
	c.latestIsTemp = true
	return ast.VisitRecurse
}

// should have been filtered by the resolver/typechecker, so err
func (c *compiler) VisitBadStmt(s *ast.BadStmt) ast.VisitResult {
	c.err("Es wurde eine invalide Aussage gefunden")
	return ast.VisitRecurse
}

func (c *compiler) VisitDeclStmt(s *ast.DeclStmt) ast.VisitResult {
	s.Decl.Accept(c)
	return ast.VisitRecurse
}

func (c *compiler) VisitExprStmt(s *ast.ExprStmt) ast.VisitResult {
	c.visitNode(s.Expr)
	return ast.VisitRecurse
}

// if t is a struct types it is declared in this compilation unit
func (c *compiler) declareIfStruct(t ddptypes.Type) {
	underlying := ddptypes.TrueUnderlying(t)
	if structType, isStruct := ddptypes.CastStruct(underlying); isStruct {
		c.defineOrDeclareStructType(structType)
	}
}

func (c *compiler) declareImportedFuncDecl(decl *ast.FuncDecl) {
	if ast.IsGeneric(decl) {
		return
	}

	mangledName := c.mangledNameDecl(decl)
	// already declared
	if _, ok := c.functions[mangledName]; ok {
		return
	}

	// declare all types this function depends on
	c.declareIfStruct(decl.ReturnType)
	for _, param := range decl.Parameters {
		c.declareIfStruct(param.Type.Type)
	}
	retType := c.toIrType(decl.ReturnType) // get the llvm type
	retTypeIr := retType.IrType()
	params := make([]*ir.Param, 0, len(decl.Parameters)) // list of the ir parameters

	hasReturnParam := !retType.IsPrimitive()
	// non-primitives are returned by passing a pointer to the struct as first parameter
	if hasReturnParam {
		params = append(params, ir.NewParam("", retType.PtrType()))
		retTypeIr = c.void.IrType()
	}

	// append all the other parameters
	for _, param := range decl.Parameters {
		ty := c.toIrParamType(param.Type)                            // convert the type of the parameter
		params = append(params, ir.NewParam(param.Name.Literal, ty)) // add it to the list
	}

	irFunc := c.mod.NewFunc(mangledName, retTypeIr, params...) // create the ir function
	irFunc.CallingConv = enum.CallingConvC                     // every function is called with the c calling convention to make interaction with inbuilt stuff easier
	// declare it as extern function
	irFunc.Linkage = enum.LinkageExternal
	irFunc.Visibility = enum.VisibilityDefault

	c.insertFunction(irFunc.Name(), decl, irFunc)
}

func (c *compiler) declareImportedVarDecl(decl *ast.VarDecl) {
	// imported decls are always in the global scope
	// even in generic instantiations
	scp := c.scp
	for !scp.isGlobalScope() {
		scp = scp.enclosing
	}

	if scp.lookupVar(decl).val != nil {
		return
	}

	c.declareIfStruct(decl.Type)
	Typ := c.toIrType(decl.Type)
	globalDecl := c.mod.NewGlobal(c.mangledNameDecl(decl), Typ.IrType())
	globalDecl.Linkage = enum.LinkageExternal
	globalDecl.Visibility = enum.VisibilityDefault

	scp.addProtected(decl, globalDecl, Typ, false) // freed by module_dispose
}

func (c *compiler) VisitImportStmt(s *ast.ImportStmt) ast.VisitResult {
	if len(s.Modules) == 0 {
		c.err("importStmt.Module == nil")
	}

	ast.IterateImportedDecls(s, func(name string, decl ast.Declaration, _ token.Token) bool {
		switch decl := decl.(type) {
		case *ast.ConstDecl:
			c.declareIfStruct(decl.Type) // not needed yet
		case *ast.VarDecl: // declare the variable as external
			c.declareImportedVarDecl(decl)
		case *ast.FuncDecl:
			c.declareImportedFuncDecl(decl)
		case *ast.TypeAliasDecl:
			c.declareIfStruct(decl.Type)
		case *ast.TypeDefDecl:
			c.declareIfStruct(decl.Type)
			c.addTypdefVTable(decl, true)
		case *ast.StructDecl:
			c.defineOrDeclareAllDeclTypes(decl)
		case *ast.BadDecl:
			c.err("BadDecl in import")
		default:
			c.err("invalid decl type")
		}
		return true
	})
	// only call the module init func once per module
	// and also initialize the modules that this module imports
	for _, mod := range s.Modules {
		ast.IterateModuleImports(mod, func(module *ast.Module) {
			if _, alreadyImported := c.importedModules[module]; alreadyImported {
				return
			}

			init_name, dispose_name := getModuleInitDisposeName(module)
			module_init := c.mod.NewFunc(init_name, c.void.IrType())
			module_init.Linkage = enum.LinkageExternal
			module_init.Visibility = enum.VisibilityDefault
			c.insertFunction(init_name, nil, module_init)
			if c.cf != nil && c.cbb != nil { // ddp_main
				c.cbb.NewCall(module_init) // only call this in main modules
			}

			module_dispose := c.mod.NewFunc(dispose_name, c.void.IrType())
			module_dispose.Linkage = enum.LinkageExternal
			module_dispose.Visibility = enum.VisibilityDefault
			c.insertFunction(dispose_name, nil, module_dispose)

			c.importedModules[module] = struct{}{}
		})
	}
	return ast.VisitRecurse
}

func (c *compiler) VisitAssignStmt(s *ast.AssignStmt) ast.VisitResult {
	rhs, rhsTyp, isTempRhs := c.evaluate(s.Rhs) // compile the expression

	lhs, lhsTyp, lhsStringIndexing := c.evaluateAssignableOrReference(s.Var, false)

	if lhsStringIndexing != nil {
		index, _, _ := c.evaluate(lhsStringIndexing.Index)
		c.cbb.NewCall(c.ddpstring.replaceCharIrFun, lhs, rhs, index)
	} else {
		c.freeNonPrimitive(lhs, lhsTyp) // free the old value in the variable/list

		// implicit cast to any if required
		if lhsTyp == c.ddpany && rhsTyp != c.ddpany {
			vtable := rhsTyp.VTable()
			if typeDef, isTypeDef := ddptypes.CastTypeDef(s.RhsType); isTypeDef {
				vtable = c.typeDefVTables[c.mangledNameType(typeDef)]
			}
			rhs, rhsTyp, isTempRhs = c.castNonAnyToAny(rhs, rhsTyp, isTempRhs, vtable)
		}

		c.claimOrCopy(lhs, rhs, rhsTyp, isTempRhs) // copy/claim the new value
	}
	return ast.VisitRecurse
}

func (c *compiler) VisitBlockStmt(s *ast.BlockStmt) ast.VisitResult {
	c.scp = newScope(c.scp) // a block gets its own scope
	wasReturn := false
	for _, stmt := range s.Statements {
		c.visitNode(stmt)
		if _, ok := stmt.(*ast.ReturnStmt); ok {
			wasReturn = true
			break
		}
	}
	if wasReturn {
		c.scp = c.scp.enclosing
	} else {
		c.scp = c.exitScope(c.scp) // free local variables and return to the previous scope
	}
	return ast.VisitRecurse
}

// for info on how the generated ir works you might want to see https://llir.github.io/document/user-guide/control/#If
func (c *compiler) VisitIfStmt(s *ast.IfStmt) ast.VisitResult {
	cond, _, _ := c.evaluate(s.Condition)
	thenBlock, elseBlock, leaveBlock := c.cf.NewBlock(""), c.cf.NewBlock(""), c.cf.NewBlock("")
	c.commentNode(c.cbb, s, "")
	if s.Else != nil {
		c.cbb.NewCondBr(cond, thenBlock, elseBlock)
	} else {
		c.cbb.NewCondBr(cond, thenBlock, leaveBlock)
	}

	c.cbb, c.scp = thenBlock, newScope(c.scp)
	c.visitNode(s.Then)
	if c.cbb.Term == nil {
		c.commentNode(c.cbb, s, "")
		c.cbb.NewBr(leaveBlock)
	}
	c.scp = c.exitScope(c.scp)

	if s.Else != nil {
		c.cbb, c.scp = elseBlock, newScope(c.scp)
		c.visitNode(s.Else)
		if c.cbb.Term == nil {
			c.commentNode(c.cbb, s, "")
			c.cbb.NewBr(leaveBlock)
		}
		c.scp = c.exitScope(c.scp)
	} else {
		elseBlock.NewUnreachable()
	}

	c.cbb = leaveBlock
	return ast.VisitRecurse
}

// for info on how the generated ir works you might want to see https://llir.github.io/document/user-guide/control/#Loop
func (c *compiler) VisitWhileStmt(s *ast.WhileStmt) ast.VisitResult {
	loopScopeBack, leaveBlockBack, continueBlockBack := c.curLoopScope, c.curLeaveBlock, c.curContinueBlock
	switch op := s.While.Type; op {
	case token.SOLANGE, token.MACHE:
		condBlock, body, bodyScope := c.cf.NewBlock(""), c.cf.NewBlock(""), newScope(c.scp)
		breakLeave := c.cf.NewBlock("")
		c.curLoopScope, c.curLeaveBlock, c.curContinueBlock = bodyScope, breakLeave, condBlock

		c.commentNode(c.cbb, s, "")
		if op == token.SOLANGE {
			c.cbb.NewBr(condBlock)
		} else {
			c.cbb.NewBr(body)
		}

		c.cbb, c.scp = body, bodyScope
		c.visitNode(s.Body)
		if c.cbb.Term == nil {
			c.cbb.NewBr(condBlock)
		}

		c.cbb, c.scp = condBlock, c.exitScope(c.scp) // the condition is not in scope
		cond, _, _ := c.evaluate(s.Condition)
		leaveBlock := c.cf.NewBlock("")
		c.commentNode(c.cbb, s, "")
		c.cbb.NewCondBr(cond, body, leaveBlock)

		trueLeave := c.cf.NewBlock("")
		leaveBlock.NewBr(trueLeave)
		breakLeave.NewBr(trueLeave)
		c.cbb = trueLeave
	case token.WIEDERHOLE:
		counter := c.NewAlloca(ddpint)
		cond, _, _ := c.evaluate(s.Condition)
		c.cbb.NewStore(cond, counter)
		condBlock, body, bodyScope := c.cf.NewBlock(""), c.cf.NewBlock(""), newScope(c.scp)
		breakLeave := c.cf.NewBlock("")
		c.curLoopScope, c.curLeaveBlock, c.curContinueBlock = bodyScope, breakLeave, condBlock

		c.commentNode(c.cbb, s, "")
		c.cbb.NewBr(condBlock)

		c.cbb, c.scp = body, bodyScope
		c.cbb.NewStore(c.cbb.NewSub(c.cbb.NewLoad(ddpint, counter), newInt(1)), counter)
		c.visitNode(s.Body)
		if c.cbb.Term == nil {
			c.commentNode(c.cbb, s, "")
			c.cbb.NewBr(condBlock)
		}

		leaveBlock := c.cf.NewBlock("")
		c.cbb, c.scp = condBlock, c.exitScope(c.scp) // the condition is not in scope
		c.commentNode(c.cbb, s, "")
		c.cbb.NewCondBr( // while counter != 0, execute body
			c.cbb.NewICmp(enum.IPredNE, c.cbb.NewLoad(ddpint, counter), zero),
			body,
			leaveBlock,
		)

		trueLeave := c.cf.NewBlock("")
		leaveBlock.NewBr(trueLeave)
		breakLeave.NewBr(trueLeave)
		c.cbb = trueLeave
	}
	c.curLoopScope, c.curLeaveBlock, c.curContinueBlock = loopScopeBack, leaveBlockBack, continueBlockBack
	return ast.VisitRecurse
}

// for info on how the generated ir works you might want to see https://llir.github.io/document/user-guide/control/#Loop
func (c *compiler) VisitForStmt(s *ast.ForStmt) ast.VisitResult {
	new_IorF_comp := func(ipred enum.IPred, fpred enum.FPred, x value.Value, yi, yf value.Value) value.Value {
		if ddptypes.DeepEqual(s.Initializer.Type, ddptypes.ZAHL) {
			return c.cbb.NewICmp(ipred, x, yi)
		} else {
			return c.cbb.NewFCmp(fpred, x, yf)
		}
	}

	loopScopeBack, leaveBlockBack, continueBlockBack := c.curLoopScope, c.curLeaveBlock, c.curContinueBlock

	c.scp = newScope(c.scp)     // scope for the for body
	c.visitNode(s.Initializer)  // compile the counter variable declaration
	var incrementer value.Value // Schrittgröße
	// if no stepsize was present it is 1
	if s.StepSize == nil {
		if ddptypes.DeepEqual(s.Initializer.Type, ddptypes.ZAHL) {
			incrementer = newInt(1)
		} else {
			incrementer = constant.NewFloat(ddpfloat, 1.0)
		}
	} else { // stepsize was present, so compile it
		incrementer, _, _ = c.evaluate(s.StepSize)
	}

	condBlock := c.cf.NewBlock("")
	incrementBlock := c.cf.NewBlock("")
	forBody := c.cf.NewBlock("")

	breakLeave := c.cf.NewBlock("")
	c.curLoopScope, c.curLeaveBlock, c.curContinueBlock = c.scp, breakLeave, incrementBlock

	c.commentNode(c.cbb, s, "")
	c.cbb.NewBr(condBlock) // we begin by evaluating the condition (not compiled yet, but the ir starts here)
	// compile the for-body
	c.cbb = forBody
	c.visitNode(s.Body)
	if c.cbb.Term == nil { // if there is no return at the end we jump to the incrementBlock
		c.commentNode(c.cbb, s, "")
		c.cbb.NewBr(incrementBlock)
	}

	// compile the incrementBlock
	c.cbb = incrementBlock
	Var := c.scp.lookupVar(s.Initializer)
	indexVar := c.cbb.NewLoad(Var.typ.IrType(), Var.val)

	// add the incrementer to the counter variable
	var add value.Value
	if ddptypes.DeepEqual(s.Initializer.Type, ddptypes.ZAHL) {
		add = c.cbb.NewAdd(indexVar, incrementer)
	} else {
		add = c.cbb.NewFAdd(indexVar, incrementer)
	}
	c.cbb.NewStore(add, c.scp.lookupVar(s.Initializer).val)
	c.commentNode(c.cbb, s, "")
	c.cbb.NewBr(condBlock) // check the condition (loop)

	// finally compile the condition block(s)
	loopDown := c.cf.NewBlock("")
	loopUp := c.cf.NewBlock("")
	leaveBlock := c.cf.NewBlock("") // after the condition is false we jump to the leaveBlock

	c.cbb = condBlock
	// we check the counter differently depending on wether or not we are looping up or down (positive vs negative stepsize)
	cond := new_IorF_comp(enum.IPredSLT, enum.FPredOLT, incrementer, newInt(0), constant.NewFloat(ddpfloat, 0.0))
	c.commentNode(c.cbb, s, "")
	c.cbb.NewCondBr(cond, loopDown, loopUp)

	c.cbb = loopUp
	// we are counting up, so compare less-or-equal
	to, _, _ := c.evaluate(s.To)
	cond = new_IorF_comp(enum.IPredSLE, enum.FPredOLE, c.cbb.NewLoad(Var.typ.IrType(), Var.val), to, to)
	c.commentNode(c.cbb, s, "")
	c.cbb.NewCondBr(cond, forBody, leaveBlock)

	c.cbb = loopDown
	// we are counting down, so compare greater-or-equal
	to, _, _ = c.evaluate(s.To)
	cond = new_IorF_comp(enum.IPredSGE, enum.FPredOGE, c.cbb.NewLoad(Var.typ.IrType(), Var.val), to, to)
	c.commentNode(c.cbb, s, "")
	c.cbb.NewCondBr(cond, forBody, leaveBlock)

	c.cbb = leaveBlock
	c.scp = c.exitScope(c.scp) // leave the scope

	trueLeave := c.cf.NewBlock("")
	leaveBlock.NewBr(trueLeave)
	breakLeave.NewBr(trueLeave)
	c.cbb = trueLeave

	c.curLoopScope, c.curLeaveBlock, c.curContinueBlock = loopScopeBack, leaveBlockBack, continueBlockBack
	return ast.VisitRecurse
}

func (c *compiler) VisitForRangeStmt(s *ast.ForRangeStmt) ast.VisitResult {
	loopScopeBack, leaveBlockBack, continueBlockBack := c.curLoopScope, c.curLeaveBlock, c.curContinueBlock

	c.scp = newScope(c.scp)
	in, inTyp, isTempIn := c.evaluate(s.In)

	temp := c.NewAlloca(inTyp.IrType())
	c.claimOrCopy(temp, in, inTyp, isTempIn)
	in, _ = c.scp.addTemporary(temp, inTyp)
	c.scp.protectTemporary(in)

	var (
		iter_ptr      value.Value // pointer used for iteration
		iter_ptr_type types.Type
		end_ptr       value.Value // points to the one-after-last element
		length        value.Value
		index         value.Value = c.NewAlloca(c.ddpinttyp.IrType())
	)

	if inTyp == c.ddpstring {
		iter_ptr_type = i8ptr
		iter_ptr = c.NewAlloca(iter_ptr_type)
		iter_ptr_val := c.loadStructField(in, string_str_field_index)
		c.cbb.NewStore(iter_ptr_val, iter_ptr)
		length = c.loadStructField(in, string_cap_field_index)
		end_ptr = c.indexArray(iter_ptr_val, c.cbb.NewSub(length, newInt(1)))
	} else {
		iter_ptr_type = inTyp.(*ddpIrListType).elementType.PtrType()
		iter_ptr = c.NewAlloca(iter_ptr_type)
		iter_ptr_val := c.loadStructField(in, list_arr_field_index)
		c.cbb.NewStore(iter_ptr_val, iter_ptr)
		length = c.loadStructField(in, list_len_field_index)
		end_ptr = c.indexArray(iter_ptr_val, length)
	}

	loopStart, condBlock, bodyBlock, incrementBlock, leaveBlock := c.cf.NewBlock(""), c.cf.NewBlock(""), c.cf.NewBlock(""), c.cf.NewBlock(""), c.cf.NewBlock("")
	c.cbb.NewCondBr(c.cbb.NewICmp(enum.IPredEQ, length, zero), leaveBlock, loopStart)

	c.cbb = loopStart
	irType := c.toIrType(s.Initializer.Type)
	c.scp.addProtected(s.Initializer, c.NewAlloca(irType.IrType()), irType, false)
	if s.Index != nil {
		c.scp.addVar(s.Index, index, c.ddpinttyp, false)
		c.cbb.NewStore(newInt(1), index)
	}
	c.cbb.NewBr(condBlock)

	c.cbb = condBlock
	c.cbb.NewCondBr(c.cbb.NewICmp(enum.IPredNE, c.cbb.NewPtrToInt(c.cbb.NewLoad(iter_ptr_type, iter_ptr), ddpint), c.cbb.NewPtrToInt(end_ptr, ddpint)), bodyBlock, leaveBlock)

	loopVar := c.scp.lookupVar(s.Initializer)

	continueBlock := c.cf.NewBlock("")
	c.cbb = continueBlock
	c.freeNonPrimitive(loopVar.val, loopVar.typ)
	c.cbb.NewBr(incrementBlock)

	c.cbb = bodyBlock
	var num_bytes value.Value
	if inTyp == c.ddpstring {
		num_bytes = c.cbb.NewCall(utf8_string_to_char_irfun,
			c.cbb.NewLoad(iter_ptr_type, iter_ptr),
			loopVar.val,
		)
		c.createIfElse(c.cbb.NewICmp(enum.IPredEQ, num_bytes, all_ones), func() {
			line, column := int64(s.In.Token().Range.Start.Line), int64(s.In.Token().Range.Start.Column)
			c.runtime_error(1, c.invalid_utf8_error_string, newInt(line), newInt(column))
		}, func() {})
	} else {
		elementPtr := c.cbb.NewLoad(iter_ptr_type, iter_ptr)
		inListTyp := inTyp.(*ddpIrListType)
		if inListTyp.elementType.IsPrimitive() {
			element := c.cbb.NewLoad(inListTyp.elementType.IrType(), elementPtr)
			c.cbb.NewStore(element, loopVar.val)
		} else {
			c.deepCopyInto(loopVar.val, elementPtr, inListTyp.elementType)
		}
	}
	breakLeave := c.cf.NewBlock("")
	breakLeave.NewBr(leaveBlock)
	c.curLoopScope, c.curLeaveBlock, c.curContinueBlock = c.scp, breakLeave, continueBlock
	c.visitNode(s.Body)
	c.freeNonPrimitive(loopVar.val, loopVar.typ)
	if c.cbb.Term == nil {
		c.cbb.NewBr(incrementBlock)
	}

	c.cbb = incrementBlock
	if inTyp == c.ddpstring {
		c.cbb.NewStore(
			c.cbb.NewIntToPtr(
				c.cbb.NewAdd(
					c.cbb.NewPtrToInt(c.cbb.NewLoad(iter_ptr_type, iter_ptr), ddpint),
					num_bytes,
				),
				i8ptr,
			),
			iter_ptr,
		)
	} else {
		inListTyp := inTyp.(*ddpIrListType)
		c.cbb.NewStore(
			c.cbb.NewIntToPtr(
				c.cbb.NewAdd(
					c.cbb.NewPtrToInt(c.cbb.NewLoad(iter_ptr_type, iter_ptr), ddpint),
					newInt(int64(c.getTypeSize(inListTyp.elementType))),
				),
				iter_ptr_type,
			),
			iter_ptr,
		)
	}
	if s.Index != nil {
		c.cbb.NewStore(c.cbb.NewAdd(c.cbb.NewLoad(c.ddpinttyp.IrType(), index), newInt(1)), index) // index += 1
	}
	c.cbb.NewBr(condBlock)

	c.cbb = leaveBlock
	c.scp.unprotectTemporary(in)
	// delete(c.scp.variables, s.Initializer.Name()) // the loopvar was already freed
	c.scp = c.exitScope(c.scp)

	c.cbb = breakLeave
	c.freeNonPrimitive(in, inTyp)
	c.freeNonPrimitive(loopVar.val, loopVar.typ)

	trueLeave := c.cf.NewBlock("")
	leaveBlock.NewBr(trueLeave)
	breakLeave.NewBr(trueLeave)
	c.cbb = trueLeave

	c.curLoopScope, c.curLeaveBlock, c.curContinueBlock = loopScopeBack, leaveBlockBack, continueBlockBack
	return ast.VisitRecurse
}

func (c *compiler) VisitBreakContinueStmt(s *ast.BreakContinueStmt) ast.VisitResult {
	c.exitNestedScopes(c.curLoopScope)
	c.commentNode(c.cbb, s, "")
	if s.Tok.Type == token.VERLASSE {
		c.cbb.NewBr(c.curLeaveBlock)
		c.cbb = c.cf.NewBlock("")
		return ast.VisitRecurse
	}
	c.cbb.NewBr(c.curContinueBlock)
	c.cbb = c.cf.NewBlock("")
	return ast.VisitRecurse
}

func (c *compiler) VisitReturnStmt(s *ast.ReturnStmt) ast.VisitResult {
	exitScopeReturn := func() {
		for scp := c.scp; scp != c.cfscp; scp = scp.enclosing {
			for _, Var := range scp.variables {
				if !Var.isRef {
					c.freeNonPrimitive(Var.val, Var.typ)
				}
			}
			c.freeTemporaries(scp, true)
		}
		c.exitFuncScope(s.Func)
	}

	if s.Value == nil {
		exitScopeReturn()
		c.commentNode(c.cbb, s, "")
		c.cbb.NewRet(nil)
		return ast.VisitRecurse
	}
	val, valTyp, isTemp := c.evaluate(s.Value)
	vtable := valTyp.VTable()
	if typeDef, isTypeDef := ddptypes.CastTypeDef(s.Func.ReturnType); isTypeDef {
		vtable = c.typeDefVTables[c.mangledNameType(typeDef)]
	}
	if valTyp.IsPrimitive() {
		// implicit cast to any if required
		if ddptypes.DeepEqual(s.Func.ReturnType, ddptypes.VARIABLE) && valTyp != c.ddpany {

			val, valTyp, isTemp = c.castNonAnyToAny(val, valTyp, isTemp, vtable)
			c.cbb.NewStore(c.cbb.NewLoad(valTyp.IrType(), val), c.cf.Params[0])
			c.claimOrCopy(c.cf.Params[0], val, valTyp, isTemp)
			c.cbb.NewRet(nil)
		} else {
			// normal return
			c.cbb.NewRet(val)
		}
	} else {
		// implicit cast to any if required
		if ddptypes.DeepEqual(s.Func.ReturnType, ddptypes.VARIABLE) && valTyp != c.ddpany {
			val, valTyp, isTemp = c.castNonAnyToAny(val, valTyp, isTemp, vtable)
		}

		c.cbb.NewStore(c.cbb.NewLoad(valTyp.IrType(), val), c.cf.Params[0])
		c.claimOrCopy(c.cf.Params[0], val, valTyp, isTemp)
		c.cbb.NewRet(nil)
	}
	exitScopeReturn()
	c.commentNode(c.cbb, s, "")
	return ast.VisitRecurse
}

func (c *compiler) VisitTodoStmt(stmt *ast.TodoStmt) ast.VisitResult {
	line, column := int64(stmt.Token().Range.Start.Line), int64(stmt.Token().Range.Start.Column)
	c.runtime_error(1, c.todo_error_string, newInt(line), newInt(column))
	return ast.VisitRecurse
}

// exits all scopes until the current function scope
// frees all scp.non_primitives
func (c *compiler) exitNestedScopes(targetScope *scope) {
	for scp := c.scp; scp != targetScope.enclosing; scp = c.exitScope(scp) {
	}
}

func (c *compiler) addTypdefVTable(d *ast.TypeDefDecl, declarationOnly bool) {
	name := c.mangledNameType(d.Type)
	if _, ok := c.typeDefVTables[name]; ok {
		return
	}

	ir_type := c.toIrType(d.Type)

	// see equivalent in runtime/include/ddptypes.h
	vtable_type := c.mod.NewTypeDef(name+"_vtable_type", types.NewStruct(
		ddpint, // ddpint type_size
		ptr(types.NewFunc(types.Void, ir_type.PtrType())),                    // free_func_ptr free_func
		ptr(types.NewFunc(types.Void, ir_type.PtrType(), ir_type.PtrType())), // deep_copy_func_ptr deep_copy_func
		ptr(types.NewFunc(ddpbool, ir_type.PtrType(), ir_type.PtrType())),    // equal_func_ptr equal_func
	))

	var vtable *ir.Global
	if declarationOnly {
		vtable = c.mod.NewGlobal(name+"_vtable", ptr(vtable_type))
		vtable.Linkage = enum.LinkageExternal
		vtable.Visibility = enum.VisibilityDefault
	} else {
		if ir_type.IsPrimitive() {
			vtable = c.mod.NewGlobalDef(name+"_vtable", constant.NewStruct(vtable_type.(*types.StructType),
				newInt(int64(c.getTypeSize(ir_type))),
				constant.NewNull(vtable_type.(*types.StructType).Fields[1].(*types.PointerType)),
				constant.NewNull(vtable_type.(*types.StructType).Fields[2].(*types.PointerType)),
				constant.NewNull(vtable_type.(*types.StructType).Fields[3].(*types.PointerType)),
			))
		} else {
			vtable = c.mod.NewGlobalDef(name+"_vtable", constant.NewStruct(vtable_type.(*types.StructType),
				newInt(int64(c.getTypeSize(ir_type))),
				ir_type.FreeFunc(),
				ir_type.DeepCopyFunc(),
				ir_type.EqualsFunc(),
			))
		}
	}
	c.typeDefVTables[name] = vtable
}
