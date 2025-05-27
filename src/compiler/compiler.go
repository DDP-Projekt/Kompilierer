package compiler

import (
	"fmt"
	"path/filepath"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ast/annotators"
	"github.com/DDP-Projekt/Kompilierer/src/compiler/llvm"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
	"github.com/DDP-Projekt/Kompilierer/src/token"
)

// compiles a mainModule and all it's imports
// every module is written to a io.Writer created
// by calling destCreator with the given module
// returns:
//   - a set of all external dependendcies
//   - an error
func compileWithImports(mod *ast.Module, resultConsumer func(*ast.Module, *Result),
	errHndl ddperror.Handler, optimizationLevel uint,
) (map[string]struct{}, error) {
	compiledMods := map[string]*ast.Module{}
	dependencies := map[string]struct{}{}
	return compileWithImportsRec(mod, resultConsumer, compiledMods, dependencies, true, errHndl, optimizationLevel)
}

func compileWithImportsRec(mod *ast.Module, resultConsumer func(*ast.Module, *Result),
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
	compiler, err := newCompiler(mod.FileName, mod, errHndl, optimizationLevel)
	if err != nil {
		return nil, err
	}
	result := compiler.compile(isMainModule)
	resultConsumer(mod, result)

	// recursively compile the other dependencies
	for _, imprt := range mod.Imports {
		for _, imprtMod := range imprt.Modules {
			if _, err := compileWithImportsRec(imprtMod, resultConsumer, compiledMods, dependencies, false, errHndl, optimizationLevel); err != nil {
				return nil, err
			}
		}
	}

	return dependencies, nil
}

// small wrapper for a ast.FuncDecl and the corresponding ir function
type funcWrapper struct {
	irFunc        llvm.Value    // the function in the llvm ir
	llFuncBuilder *llBuilder    // nil for inbuilt or runtime functions
	funcDecl      *ast.FuncDecl // the ast.FuncDecl
}

type llTypes struct {
	ptr, void, i8, i32, i64                     llvm.Type
	ddpint, ddpfloat, ddpbyte, ddpbool, ddpchar llvm.Type
	vtable_type                                 llvm.Type
}

func newLLTypes(llctx llvm.Context) llTypes {
	ptr := llctx.PointerType(0)
	i8 := llctx.Int8Type()
	i32 := llctx.Int32Type()
	i64 := llctx.Int64Type()
	return llTypes{
		ptr:      ptr,
		void:     llctx.VoidType(),
		i8:       i8,
		i32:      i32,
		i64:      i64,
		ddpint:   i64,
		ddpfloat: llctx.DoubleType(),
		ddpbyte:  llctx.Int8Type(),
		ddpbool:  llctx.Int1Type(),
		ddpchar:  llctx.Int32Type(),
		vtable_type: llctx.StructType([]llvm.Type{
			i64,
			ptr,
			ptr,
			ptr,
		}, false,
		),
	}
}

type llConstants struct {
	zero, zerof, zero8, all_ones, all_ones8, False, True, Null llvm.Value
}

func newLLConstants(types llTypes) llConstants {
	return llConstants{
		zero:      llvm.ConstInt(types.i64, 0, false),
		zerof:     llvm.ConstFloat(types.ddpfloat, 0),
		zero8:     llvm.ConstInt(types.i8, 0, false),
		all_ones:  llvm.ConstAllOnes(types.i64),
		all_ones8: llvm.ConstAllOnes(types.i8),
		False:     llvm.ConstInt(types.ddpbool, 0, false),
		True:      llvm.ConstInt(types.ddpbool, 1, false),
		Null:      llvm.ConstNull(types.ptr),
	}
}

// holds state to compile a DDP AST into llvm ir
type compiler struct {
	llvmModuleContext
	ddpModule         *ast.Module      // the module to be compiled
	errorHandler      ddperror.Handler // errors are passed to this function
	optimizationLevel uint             // level of optimization
	result            *Result          // result of the compilation

	builderStack    []*llBuilder
	functions       map[string]*funcWrapper                   // all the global functions
	typeMap         map[ddptypes.Type]*ast.Module             // maps ddpTypes to the module they originate from
	structTypes     map[*ddptypes.StructType]*ddpIrStructType // struct names mapped to their IR type
	importedModules map[*ast.Module]struct{}                  // all the modules that have already been imported
	typeDefVTables  map[string]llvm.Value

	moduleInitBuilder          *llBuilder // the module_init func of this module
	moduleDisposeBuilder       *llBuilder
	out_of_bounds_error_string llvm.Value
	slice_error_string         llvm.Value
	todo_error_string          llvm.Value
	bad_cast_error_string      llvm.Value
	invalid_utf8_error_string  llvm.Value

	// raw llvm types and constants
	llTypes
	llConstants
	// all the type definitions of inbuilt types used by the compiler
	voidtyp                                                                                    *ddpIrVoidType
	ddpinttyp, ddpfloattyp, ddpbytetyp, ddpbooltyp, ddpchartyp                                 *ddpIrPrimitiveType
	ddpstring                                                                                  *ddpIrStringType
	ddpany                                                                                     *ddpIrAnyType
	ddpintlist, ddpfloatlist, ddpbytelist, ddpboollist, ddpcharlist, ddpstringlist, ddpanylist *ddpIrListType
	ddpgenericlist                                                                             *ddpIrGenericListType
}

// create a new Compiler to compile the passed AST
func newCompiler(name string, module *ast.Module, errorHandler ddperror.Handler, optimizationLevel uint) (*compiler, error) {
	if errorHandler == nil { // default error handler does nothing
		errorHandler = ddperror.EmptyHandler
	}

	context, err := newllvmModuleContext(name)
	if err != nil {
		return nil, fmt.Errorf("Error creating llvmModuleContext: %w", context)
	}

	types := newLLTypes(context.llctx)
	constants := newLLConstants(types)

	return &compiler{
		llvmModuleContext: context,
		ddpModule:         module,
		errorHandler:      errorHandler,
		optimizationLevel: optimizationLevel,
		result: &Result{
			Dependencies: make(map[string]struct{}),
		},

		functions:       make(map[string]*funcWrapper),
		typeMap:         createTypeMap(module),
		structTypes:     make(map[*ddptypes.StructType]*ddpIrStructType),
		importedModules: make(map[*ast.Module]struct{}),
		typeDefVTables:  make(map[string]llvm.Value),

		llTypes:     types,
		llConstants: constants,
	}, nil
}

// compile the AST contained in c
// if w is not nil, the resulting llir is written to w
// otherwise a string representation is returned in result
// if isMainModule is false, no ddp_main function will be generated
func (c *compiler) compile(isMainModule bool) (result *Result) {
	defer compiler_panic_wrapper(c)

	c.addExternalDependencies()

	c.pushBuilder(&llBuilder{
		c:       c,
		Builder: c.llctx.NewBuilder(),
		scp:     newScope(nil),
	})

	c.setup()

	if isMainModule {
		c.disposeAndPop()
		c.newBuilder("ddp_ddpmain", llvm.FunctionType(c.llctx.Int64Type(), nil, false), nil)
		// called from the ddp-c-runtime after initialization
		c.insertFunction(
			"ddp_ddpmain",
			nil,
			c.builder().llFn,
			c.builder(),
		)
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
		c.builder().scp = c.exitScope(c.builder().scp) // exit the main scope
		// call all the module_dispose functions
		for mod := range c.importedModules {
			_, dispose_name := getModuleInitDisposeName(mod)
			dispose_fun := c.functions[dispose_name]
			c.builder().CreateCall(c.void, dispose_fun.irFunc, nil, "")
		}
		// on success ddpmain returns 0
		c.builder().CreateRet(c.zero)
	}

	c.moduleInitBuilder.CreateRet(llvm.Value{}) // terminate the module_init func

	for _, wrapper := range c.functions {
		wrapper.llFuncBuilder.Dispose()
	}

	return c.result
}

// dumps only the definitions for inbuilt list types to w
func (c *compiler) dumpListDefinitions() llvmModuleContext {
	defer compiler_panic_wrapper(c)

	c.pushBuilder(&llBuilder{
		c:       c,
		Builder: c.llctx.NewBuilder(),
		scp:     newScope(nil),
	})

	c.setupErrorStrings()
	// the order of these function calls is important
	// because the primitive types need to be setup
	// before the list types
	// and the void type before everything else
	c.voidtyp = &ddpIrVoidType{}
	c.initRuntimeFunctions()
	c.setupPrimitiveTypes(false)
	c.ddpstring = c.defineStringType(false)
	c.ddpany = c.defineAnyType()
	c.setupListTypes(false) // we want definitions

	return c.llvmModuleContext
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

// helper to visit a single node
func (c *compiler) visitNode(node ast.Node) {
	c.builder().currentNode = node
	node.Accept(c)
}

// helper to evaluate an expression and return its ir value and type
// the  bool signals wether the returned value is a temporary value that can be claimed
// or if it is a 'reference' to a variable that must be copied
func (c *compiler) evaluate(expr ast.Expression) (llvm.Value, ddpIrType, bool) {
	c.visitNode(expr)
	return c.builder().latestReturn, c.builder().latestReturnType, c.builder().latestIsTemp
}

// helper to insert a function into the global function map
// returns the ir function
func (c *compiler) insertFunction(name string, funcDecl *ast.FuncDecl, llFunc llvm.Value, builder *llBuilder) llvm.Value {
	c.functions[name] = &funcWrapper{
		funcDecl:      funcDecl,
		irFunc:        llFunc,
		llFuncBuilder: builder,
	}
	return llFunc
}

func (c *compiler) setup() {
	c.setupErrorStrings()

	// the order of these function calls is important
	// because the primitive types need to be setup
	// before the list types
	c.voidtyp = c.defineVoidType()
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
	createErrorString := func(msg string) llvm.Value {
		error_string := c.builder().CreateGlobalString(msg, "")
		// error_string.SetLinkage(llvm.InternalLinkage)
		// error_string.SetVisibility(llvm.DefaultVisibility)
		error_string.SetGlobalConstant(true)
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
	c.ddpinttyp = c.definePrimitiveType(c.ddpint, c.zero, "ddpint", declarationOnly)
	c.ddpfloattyp = c.definePrimitiveType(c.ddpfloat, c.zerof, "ddpfloat", declarationOnly)
	c.ddpbytetyp = c.definePrimitiveType(c.ddpbyte, c.zero8, "ddpbyte", declarationOnly)
	c.ddpbooltyp = c.definePrimitiveType(c.ddpbool, c.False, "ddpbool", declarationOnly)
	c.ddpchartyp = c.definePrimitiveType(c.ddpchar, llvm.ConstInt(c.ddpchar, 0, false), "ddpchar", declarationOnly)
}

// used in setup()
func (c *compiler) setupListTypes(declarationOnly bool) {
	c.ddpintlist = c.createListType("ddpintlist", c.ddpinttyp, declarationOnly)
	c.ddpfloatlist = c.createListType("ddpfloatlist", c.ddpfloattyp, declarationOnly)
	c.ddpbytelist = c.createListType("ddpbytelist", c.ddpbytetyp, declarationOnly)
	c.ddpboollist = c.createListType("ddpboollist", c.ddpbooltyp, declarationOnly)
	c.ddpcharlist = c.createListType("ddpcharlist", c.ddpchartyp, declarationOnly)
	c.ddpstringlist = c.createListType("ddpstringlist", c.ddpstring, declarationOnly)
	c.ddpanylist = c.createListType("ddpanylist", c.ddpany, declarationOnly)
	c.ddpgenericlist = c.createGenericListType()
}

// used in setup()
// creates a function that can be called to initialize the global state of this module
func (c *compiler) setupModuleInitDispose() {
	init_name, dispose_name := getModuleInitDisposeName(c.ddpModule)
	c.moduleInitBuilder = c.createBuilder(init_name, llvm.FunctionType(c.void, nil, false), nil)
	c.moduleInitBuilder.llFn.SetVisibility(llvm.DefaultVisibility)
	c.insertFunction(init_name, nil, c.moduleInitBuilder.llFn, c.moduleInitBuilder)

	c.moduleDisposeBuilder = c.createBuilder(dispose_name, llvm.FunctionType(c.void, nil, false), nil)
	c.moduleInitBuilder.llFn.SetVisibility(llvm.DefaultVisibility)
	c.insertFunction(dispose_name, nil, c.moduleDisposeBuilder.llFn, c.moduleDisposeBuilder)
}

// used in setup()
func (c *compiler) setupOperators() {
	// hoch operator for different type combinations
	c.declareExternalRuntimeFunction("pow", false, c.ddpfloat, c.ddpfloat, c.ddpfloat)

	// logarithm
	c.declareExternalRuntimeFunction("log10", false, c.ddpfloat, c.ddpfloat)

	// ddpstring to type cast
	c.declareExternalRuntimeFunction("ddp_string_to_int", false, c.ddpint, c.ptr)
	c.declareExternalRuntimeFunction("ddp_string_to_float", false, c.ddpfloat, c.ptr)
}

// deep copies the value pointed to by src into dest
// and returns dest
func (c *compiler) deepCopyInto(dest, src llvm.Value, typ ddpIrType) llvm.Value {
	dest = c.builder().CreateCall(c.void, typ.DeepCopyFunc(), []llvm.Value{dest, src}, "")
	return dest
}

// calls the corresponding free function on val
// if typ.IsPrimitive() == false
func (c *compiler) freeNonPrimitive(val llvm.Value, typ ddpIrType) {
	if !typ.IsPrimitive() {
		c.builder().CreateCall(c.void, typ.FreeFunc(), []llvm.Value{val}, "")
	}
}

// claims the given value if possible, copies it otherwise
// dest should be a value that is definetly freed at some point (meaning a variable or list-element etc.)
func (c *compiler) claimOrCopy(dest, val llvm.Value, valTyp ddpIrType, isTemp bool) {
	if !valTyp.IsPrimitive() {
		if isTemp { // temporaries can be claimed
			val = c.builder().CreateLoad(valTyp.LLType(), c.builder().scp.claimTemporary(val), "")
			c.builder().CreateStore(val, dest)
		} else { // non-temporaries need to be copied
			c.deepCopyInto(dest, val, valTyp)
		}
	} else { // primitives are trivially copied
		c.builder().CreateStore(val, dest) // store the value
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

	for paramDecl, v := range c.builder().fnScope.variables {
		if !v.isRef && (!meta.IsConst[paramDecl.Name()] || c.optimizationLevel < 2) {
			c.freeNonPrimitive(v.val, v.typ)
		}
	}
	c.freeTemporaries(c.builder().fnScope, true)
	return c.builder().fnScope.enclosing
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
	var varLocation llvm.Value
	if c.builder().scp.isGlobalScope() { // global scope
		// globals are first assigned in ddp_main or module_init
		// so we assign them a default value here
		//
		// names are mangled only in the actual ir-definitions, not in the compiler data-structures
		globalDef := llvm.AddGlobal(c.llmod, Typ.LLType(), c.mangledNameDecl(d))
		globalDef.SetInitializer(Typ.DefaultValue())
		// make private variables static like in C
		// commented out because of generics where private variables might be used
		// from a different module
		// if !d.IsPublic && !d.IsExternVisible {
		// 	globalDef.Linkage = enum.LinkageInternal
		// }
		globalDef.SetVisibility(llvm.DefaultVisibility)
		varLocation = globalDef
	} else {
		varLocation = c.NewAlloca(Typ.LLType())
	}

	// adds the variable initializer to the function fun
	addInitializer := func() {
		initVal, initTyp, isTemp := c.evaluate(d.InitVal) // evaluate the initial value

		// implicit numeric casts
		if ddptypes.IsNumeric(d.Type) && ddptypes.IsNumeric(d.InitType) {
			initVal, initTyp = c.numericCast(initVal, initTyp, Typ), Typ
		}

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

	if c.builder().scp.isGlobalScope() { // module_init
		c.pushBuilder(c.moduleInitBuilder)
		current_temporaries_end := len(c.builder().scp.temporaries)
		addInitializer() // initialize the variable in module_init
		// free all temporaries that were created in the initializer
		for _, v := range c.builder().scp.temporaries[current_temporaries_end:] {
			c.freeNonPrimitive(v.val, v.typ)
		}
		c.builder().scp.temporaries = c.builder().scp.temporaries[:current_temporaries_end]
		c.popBuilder()

		c.pushBuilder(c.moduleDisposeBuilder)
		c.freeNonPrimitive(varLocation, Typ) // free the variable in module_dispose
		c.popBuilder()
	}

	// if those are nil, we are at the global scope but there is no ddp_main func
	// meaning this module is being compiled as a non-main module
	if c.builder().isDDPMain() {
		addInitializer()
	}

	c.builder().scp.addVar(d, varLocation, Typ, false)
	return ast.VisitRecurse
}

func (c *compiler) getPossiblyGenericReturnType(decl *ast.FuncDecl) ddpIrType {
	// if the return type is generic it can only be a list
	if _, isGeneric := ddptypes.CastDeeplyNestedGenerics(decl.ReturnType); isGeneric {
		return c.ddpgenericlist
	} else {
		return c.toIrType(decl.ReturnType) // get the llvm type
	}
}

func (c *compiler) getPossiblyGenericParamType(param *ast.ParameterInfo) llvm.Type {
	if _, isGeneric := ddptypes.CastDeeplyNestedGenerics(param.Type.Type); isGeneric && param.Type.IsReference {
		return c.ptr
	} else if isGeneric {
		return c.ptr
	} else {
		return c.toIrParamType(param.Type) // convert the type of the parameter
	}
}

func (c *compiler) VisitFuncDecl(decl *ast.FuncDecl) ast.VisitResult {
	if ast.IsGeneric(decl) && !ast.IsExternFunc(decl) {
		return ast.VisitRecurse
	}

	// extern functions are instantiated once
	if ast.IsGenericInstantiation(decl) && ast.IsExternFunc(decl) {
		decl = decl.GenericInstantiation.GenericDecl
	}

	retType := c.getPossiblyGenericReturnType(decl)

	retTypeIr := retType.LLType()
	params := make([]llvm.Type, 0, len(decl.Parameters)+1) // list of the ir parameters
	paramNames := make([]string, 0, len(decl.Parameters)+1)

	hasReturnParam := !retType.IsPrimitive()
	// non-primitives are returned by passing a pointer to the struct as first parameter
	if hasReturnParam {
		params = append(params, c.ptr)
		paramNames = append(paramNames, "")
		retTypeIr = c.voidtyp.LLType()
	}

	// append all the other parameters
	for _, param := range decl.Parameters {
		paramIrType := c.getPossiblyGenericParamType(&param)

		params = append(params, paramIrType) // add it to the list
		paramNames = append(paramNames, param.Name.Literal)
	}

	llFuncBuilder := c.newBuilder(c.mangledNameDecl(decl), llvm.FunctionType(retTypeIr, params, false), paramNames)
	// make private functions static like in C
	// commented out because of generics where private functions might be called
	// from a different module
	// if !decl.IsPublic && !decl.IsExternVisible {
	// 	irFunc.Linkage = enum.LinkageInternal
	// 	irFunc.Visibility = enum.VisibilityDefault
	// }

	c.insertFunction(llFuncBuilder.fnName, decl, llFuncBuilder.llFn, llFuncBuilder)

	// inbuilt or external functions are defined in c
	if ast.IsExternFunc(decl) {
		llFuncBuilder.llFn.SetLinkage(llvm.ExternalLinkage)
	} else if !ast.IsForwardDecl(decl) {
		c.defineFuncBody(llFuncBuilder, hasReturnParam, decl)
	}
	return ast.VisitRecurse
}

func (c *compiler) VisitFuncDef(def *ast.FuncDef) ast.VisitResult {
	fun := c.functions[c.mangledNameDecl(def.Func)] // retreive the function (the resolver took care that it is present)
	retType := c.toIrType(def.Func.ReturnType)      // get the llvm type

	c.defineFuncBody(fun.llFuncBuilder, !retType.IsPrimitive(), def.Func)
	return ast.VisitRecurse
}

// helper function for VisitFuncDef and VisitFuncDecl to compile the  body of a ir function
func (c *compiler) defineFuncBody(llFuncBuilder *llBuilder, hasReturnParam bool, decl *ast.FuncDecl) {
	llFuncBuilder.scp = newScope(c.builder().scp)
	c.pushBuilder(llFuncBuilder)
	defer c.popBuilder()

	params := llFuncBuilder.params
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
		varDecl, _, _ := body.Symbols.LookupDecl(params[i].name)
		paramDecl := varDecl.(*ast.VarDecl)
		if decl.Parameters[i].Type.IsReference {
			// references are implemented similar to name-shadowing
			// they basically just get another name in the function scope, which
			// refers to the same variable allocation
			c.builder().scp.addVar(paramDecl, params[i].val, irType, true)
		} else if !irType.IsPrimitive() { // strings and lists need special handling
			// add the local variable for the parameter
			v := c.builder().scp.addVar(paramDecl, c.NewAlloca(irType.LLType()), irType, false)
			c.builder().CreateStore(c.builder().CreateLoad(irType.LLType(), params[i].val, ""), v) // store the copy in the local variable
		} else { // primitive types don't need any special handling
			v := c.builder().scp.addVar(paramDecl, c.NewAlloca(irType.LLType()), irType, false)
			c.builder().CreateStore(params[i].val, v)
		}
	}

	// modified VisitBlockStmt
	c.builder().scp = newScope(c.builder().scp) // a block gets its own scope
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
		c.builder().scp = c.builder().scp.enclosing
	} else {
		c.builder().scp = c.exitScope(c.builder().scp)
	}

	if c.builder().cb.Terminator().IsNil() {
		c.builder().CreateRet(llvm.Value{}) // every block needs a terminator, and every function a return
	}

	// free the parameters of the function
	if toplevelReturn {
		c.builder().scp = c.builder().scp.enclosing
	} else {
		c.builder().scp = c.exitFuncScope(decl)
	}
	c.popBuilder()
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

	Var := c.builder().scp.lookupVar(e.Declaration.(*ast.VarDecl)) // get the alloca in the ir

	if Var.typ.IsPrimitive() { // primitives are simply loaded
		c.builder().latestReturn = c.builder().CreateLoad(Var.typ.LLType(), Var.val, "")
	} else { // non-primitives are used by pointer
		c.builder().latestReturn = Var.val
	}
	c.builder().latestReturnType = Var.typ
	c.builder().latestIsTemp = false
	return ast.VisitRecurse
}

func (c *compiler) VisitIndexing(e *ast.Indexing) ast.VisitResult {
	elementPtr, elementType, stringIndexing := c.evaluateAssignableOrReference(e, false)

	if stringIndexing != nil {
		lhs, lhsTyp, _ := c.evaluate(stringIndexing.Lhs)
		index, _, _ := c.evaluate(stringIndexing.Index)
		lhs = c.floatOrByteAsInt(lhs, lhsTyp)
		c.builder().latestReturn = c.builder().CreateCall(c.ddpchar, c.ddpstring.indexIrFun, []llvm.Value{lhs, index}, "")
		c.builder().latestReturnType = c.ddpchartyp
		// c.builder().latestIsTemp = false // it is a primitive typ, so we don't care
		return ast.VisitRecurse
	} else {
		if elementType.IsPrimitive() {
			c.builder().latestReturn = c.builder().CreateLoad(elementType.LLType(), elementPtr, "")
		} else {
			c.builder().latestReturn = elementPtr
			c.builder().latestIsTemp = false
		}
	}
	c.builder().latestReturnType = elementType
	return ast.VisitRecurse
}

func (c *compiler) VisitFieldAccess(expr *ast.FieldAccess) ast.VisitResult {
	fieldPtr, fieldType, _ := c.evaluateAssignableOrReference(expr, false)

	if fieldType.IsPrimitive() {
		c.builder().latestReturn = c.builder().CreateLoad(fieldType.LLType(), fieldPtr, "")
	} else {
		dest := c.NewAlloca(fieldType.LLType())
		c.deepCopyInto(dest, fieldPtr, fieldType)
		c.builder().latestReturn, c.builder().latestReturnType = c.builder().scp.addTemporary(dest, fieldType)
		c.builder().latestIsTemp = true
	}
	c.builder().latestReturnType = fieldType
	return ast.VisitRecurse
}

// literals are simple ir constants
func (c *compiler) VisitIntLit(e *ast.IntLit) ast.VisitResult {
	c.builder().latestReturn = c.newInt(e.Value)
	c.builder().latestReturnType = c.ddpinttyp
	return ast.VisitRecurse
}

func (c *compiler) VisitFloatLit(e *ast.FloatLit) ast.VisitResult {
	c.builder().latestReturn = llvm.ConstFloat(c.ddpfloat, e.Value)
	c.builder().latestReturnType = c.ddpfloattyp
	return ast.VisitRecurse
}

func (c *compiler) VisitBoolLit(e *ast.BoolLit) ast.VisitResult {
	c.builder().latestReturn = c.newIntT(c.ddpbool, int64(boolToInt(e.Value)))
	c.builder().latestReturnType = c.ddpbooltyp
	return ast.VisitRecurse
}

func (c *compiler) VisitCharLit(e *ast.CharLit) ast.VisitResult {
	c.builder().latestReturn = c.newIntT(c.ddpchar, int64(e.Value))
	c.builder().latestReturnType = c.ddpchartyp
	return ast.VisitRecurse
}

// string literals are created by the runtime
// so we need to do some work here
func (c *compiler) VisitStringLit(e *ast.StringLit) ast.VisitResult {
	// call the ddp-runtime function to create the ddpstring
	dest := c.NewAlloca(c.ddpstring.typ)
	if e.Value == "" {
		c.builder().CreateStore(c.ddpstring.DefaultValue(), dest)
	} else {
		constStr := c.builder().CreateGlobalString(e.Value, "")
		c.builder().CreateCall(c.void, c.ddpstring.fromConstantsIrFun, []llvm.Value{dest, constStr}, "")
	}
	c.builder().latestReturn, c.builder().latestReturnType = c.builder().scp.addTemporary(dest, c.ddpstring) // so that it is freed later
	c.builder().latestIsTemp = true
	return ast.VisitRecurse
}

func (c *compiler) VisitListLit(e *ast.ListLit) ast.VisitResult {
	listType := c.toIrType(e.Type).(*ddpIrListType)
	list := c.NewAlloca(listType.LLType())

	// get the listLen as irValue
	listLen := c.zero
	if e.Values != nil {
		listLen = c.newInt(int64(len(e.Values)))
	} else if e.Count != nil && e.Value != nil {
		var lenType ddpIrType
		listLen, lenType, _ = c.evaluate(e.Count)
		listLen = c.floatOrByteAsInt(listLen, lenType)
	} else { // empty list
		c.builder().CreateStore(listType.DefaultValue(), list)
		c.builder().latestReturn, c.builder().latestReturnType = c.builder().scp.addTemporary(list, listType)
		c.builder().latestIsTemp = true
		return ast.VisitRecurse
	}

	// create a empty list of the correct length
	c.builder().CreateCall(c.void, listType.fromConstantsIrFun, []llvm.Value{list, listLen}, "")

	listArr := c.loadStructField(listType.typ, list, list_arr_field_index) // load the array

	if e.Values != nil { // we got some values to copy
		// evaluate every value and copy it into the array
		for i, v := range e.Values {
			val, valTyp, isTemp := c.evaluate(v)
			elementPtr := c.indexArray(listType.elementType.LLType(), listArr, c.newInt(int64(i)))
			c.claimOrCopy(elementPtr, val, valTyp, isTemp)
		}
	} else if e.Count != nil && e.Value != nil { // single Value multiple times
		val, _, _ := c.evaluate(e.Value) // if val is a temporary, it is freed automatically

		c.createFor(c.zero, c.forDefaultCond(listLen), func(index llvm.Value) {
			elementPtr := c.indexArray(listType.elementType.LLType(), listArr, index)
			if listType.elementType.IsPrimitive() {
				c.builder().CreateStore(val, elementPtr)
			} else {
				c.deepCopyInto(elementPtr, val, listType.elementType)
			}
		})
	}
	c.builder().latestReturn, c.builder().latestReturnType = c.builder().scp.addTemporary(list, listType)
	c.builder().latestIsTemp = true
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
	switch e.Operator {
	case ast.UN_ABS:
		switch typ {
		case c.ddpfloattyp:
			// c.builder().latestReturn = rhs < 0 ? 0 - rhs : rhs;
			c.builder().latestReturn = c.createTernary(c.ddpfloat, c.builder().CreateFCmp(llvm.FloatOLT, rhs, c.zerof, ""),
				func() llvm.Value { return c.builder().CreateFSub(c.zerof, rhs, "") },
				func() llvm.Value { return rhs },
			)
			c.builder().latestReturnType = c.ddpfloattyp
		case c.ddpinttyp:
			// c.builder().latestReturn = rhs < 0 ? 0 - rhs : rhs;
			c.builder().latestReturn = c.createTernary(c.ddpint, c.builder().CreateICmp(llvm.IntSLT, rhs, c.zero, ""),
				func() llvm.Value { return c.builder().CreateSub(c.zero, rhs, "") },
				func() llvm.Value { return rhs },
			)
			c.builder().latestReturnType = c.ddpinttyp
		case c.ddpbytetyp:
			// a byte is unsigned and therefore does not need to be changed
		default:
			c.err("invalid Parameter Type for BETRAG: %s", typ.Name())
		}
	case ast.UN_NEGATE:
		switch typ {
		case c.ddpfloattyp:
			c.builder().latestReturn = c.builder().CreateFNeg(rhs, "")
			c.builder().latestReturnType = c.ddpfloattyp
		case c.ddpinttyp:
			c.builder().latestReturn = c.builder().CreateSub(c.zero, rhs, "")
			c.builder().latestReturnType = c.ddpinttyp
		case c.ddpinttyp:
			c.builder().latestReturn = c.builder().CreateSub(c.zero, c.floatOrByteAsInt(rhs, c.ddpbytetyp), "")
			c.builder().latestReturnType = c.ddpinttyp
		default:
			c.err("invalid Parameter Type for NEGATE: %s", typ.Name())
		}
	case ast.UN_NOT:
		c.builder().latestReturn = c.builder().CreateXor(rhs, c.newInt(1), "")
		c.builder().latestReturnType = c.ddpbooltyp
	case ast.UN_LOGIC_NOT:
		switch typ {
		case c.ddpinttyp:
			c.builder().latestReturn = c.builder().CreateXor(rhs, c.all_ones, "")
			c.builder().latestReturnType = c.ddpinttyp
		case c.ddpbytetyp:
			c.builder().latestReturn = c.builder().CreateXor(rhs, c.all_ones8, "")
			c.builder().latestReturnType = c.ddpbytetyp
		}
	case ast.UN_LEN:
		switch typ {
		case c.ddpstring:
			c.builder().latestReturn = c.builder().CreateCall(c.ddpint, c.ddpstring.lengthIrFun, []llvm.Value{rhs}, "")
		default:
			if listTyp, isList := typ.(*ddpIrListType); isList {
				c.builder().latestReturn = c.loadStructField(listTyp.typ, rhs, list_len_field_index)
			} else {
				c.err("invalid Parameter Type for LÄNGE: %s", typ.Name())
			}
		}
		c.builder().latestReturnType = c.ddpinttyp
	default:
		c.err("Unbekannter Operator '%s'", e.Operator)
	}
	return ast.VisitRecurse
}

func (c *compiler) VisitBinaryExpr(e *ast.BinaryExpr) ast.VisitResult {
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
		startBlock, trueBlock, leaveBlock := c.builder().cb, c.builder().newBlock(), c.builder().newBlock()
		c.builder().CreateCondBr(lhs, trueBlock, leaveBlock)

		c.builder().setBlock(trueBlock)
		// collect temporaries because of possible short-circuiting
		c.builder().scp = newScope(c.builder().scp)
		rhs, _, _ := c.evaluate(e.Rhs)
		// free temporaries
		c.builder().scp = c.exitScope(c.builder().scp)
		c.builder().CreateBr(leaveBlock)
		trueBlock = c.builder().cb

		c.builder().setBlock(leaveBlock)
		phi := c.builder().CreatePHI(c.ddpbool, "")
		phi.AddIncoming([]llvm.Value{rhs, lhs}, []llvm.BasicBlock{trueBlock, startBlock})
		c.builder().latestReturn = phi
		c.builder().latestReturnType = c.ddpbooltyp
		return ast.VisitRecurse
	case ast.BIN_OR:
		lhs, _, _ := c.evaluate(e.Lhs)
		startBlock, falseBlock, leaveBlock := c.builder().cb, c.builder().newBlock(), c.builder().newBlock()
		c.builder().CreateCondBr(lhs, leaveBlock, falseBlock)

		c.builder().setBlock(falseBlock)
		// collect temporaries because of possible short-circuiting
		c.builder().scp = newScope(c.builder().scp)
		rhs, _, _ := c.evaluate(e.Rhs)
		// free temporaries
		c.builder().scp = c.exitScope(c.builder().scp)
		c.builder().CreateBr(leaveBlock)
		falseBlock = c.builder().cb // in case c.evaluate has multiple blocks

		c.builder().setBlock(leaveBlock)
		phi := c.builder().CreatePHI(c.ddpbool, "")
		phi.AddIncoming([]llvm.Value{lhs, rhs}, []llvm.BasicBlock{startBlock, falseBlock})
		c.builder().latestReturn = phi
		c.builder().latestReturnType = c.ddpbooltyp
		return ast.VisitRecurse
	case ast.BIN_FIELD_ACCESS:
		rhs, rhsTyp, rhsIsTemp := c.evaluate(e.Rhs)
		if structType, isStruct := rhsTyp.(*ddpIrStructType); isStruct {
			fieldIndex := getFieldIndex(e.Lhs.Token().Literal, structType)
			fieldType := structType.fieldIrTypes[fieldIndex]
			fieldPtr := c.indexStruct(structType.typ, rhs, fieldIndex)
			if fieldType.IsPrimitive() {
				c.builder().latestReturn = c.builder().CreateLoad(fieldType.LLType(), fieldPtr, "")
			} else if !rhsIsTemp {
				c.builder().latestReturn, c.builder().latestIsTemp = fieldPtr, false
			} else {
				dest := c.NewAlloca(fieldType.LLType())
				c.builder().CreateStore(c.builder().CreateLoad(fieldType.LLType(), fieldPtr, ""), dest)
				c.builder().CreateStore(fieldType.DefaultValue(), fieldPtr)
				c.builder().latestReturn, c.builder().latestReturnType = c.builder().scp.addTemporary(dest, fieldType)
				c.builder().latestIsTemp = true
			}
			c.builder().latestReturnType = fieldType
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
		c.builder().latestReturn = c.builder().CreateXor(lhs, rhs, "")
		c.builder().latestReturnType = c.ddpbooltyp
	case ast.BIN_CONCAT:
		var (
			result    llvm.Value
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
		result = c.NewAlloca(resultTyp.LLType())

		// string concatenations
		var concat_func llvm.Value
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
		if concat_func.IsNil() {
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
			dest := c.NewAlloca(lhsTyp.LLType())
			lhs = c.deepCopyInto(dest, lhs, lhsTyp)
		}
		if claimsRhs && !isTempRhs {
			dest := c.NewAlloca(rhsTyp.LLType())
			rhs = c.deepCopyInto(dest, rhs, rhsTyp)
		}

		c.builder().CreateCall(c.void, concat_func, []llvm.Value{result, lhs, rhs}, "")
		c.builder().latestReturn, c.builder().latestReturnType = c.builder().scp.addTemporary(result, resultTyp)
		c.builder().latestIsTemp = true
	case ast.BIN_PLUS:
		switch lhsTyp {
		case c.ddpinttyp:
			switch rhsTyp {
			case c.ddpinttyp:
				c.builder().latestReturn = c.builder().CreateAdd(lhs, rhs, "")
				c.builder().latestReturnType = c.ddpinttyp
			case c.ddpfloattyp:
				fp := c.builder().CreateSIToFP(lhs, c.ddpfloat, "")
				c.builder().latestReturn = c.builder().CreateFAdd(fp, rhs, "")
				c.builder().latestReturnType = c.ddpfloattyp
			case c.ddpbytetyp:
				c.builder().latestReturn = c.builder().CreateAdd(lhs, c.floatOrByteAsInt(rhs, c.ddpbytetyp), "")
				c.builder().latestReturnType = c.ddpinttyp
			default:
				c.err("invalid Parameter Types for PLUS (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		case c.ddpfloattyp:
			switch rhsTyp {
			case c.ddpinttyp:
				fp := c.builder().CreateSIToFP(rhs, c.ddpfloat, "")
				c.builder().latestReturn = c.builder().CreateFAdd(lhs, fp, "")
			case c.ddpfloattyp:
				c.builder().latestReturn = c.builder().CreateFAdd(lhs, rhs, "")
			case c.ddpbytetyp:
				fp := c.builder().CreateUIToFP(rhs, c.ddpfloat, "")
				c.builder().latestReturn = c.builder().CreateFAdd(lhs, fp, "")
			default:
				c.err("invalid Parameter Types for PLUS (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
			c.builder().latestReturnType = c.ddpfloattyp
		case c.ddpbytetyp:
			switch rhsTyp {
			case c.ddpinttyp:
				c.builder().latestReturn = c.builder().CreateAdd(c.floatOrByteAsInt(lhs, c.ddpbytetyp), rhs, "")
				c.builder().latestReturnType = c.ddpinttyp
			case c.ddpfloattyp:
				fp := c.builder().CreateUIToFP(lhs, c.ddpfloat, "")
				c.builder().latestReturn = c.builder().CreateFAdd(fp, rhs, "")
			case c.ddpbytetyp:
				c.builder().latestReturn = c.builder().CreateAdd(lhs, rhs, "")
				c.builder().latestReturnType = c.ddpbytetyp
			default:
				c.err("invalid Parameter Types for PLUS (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		default:
			c.err("invalid Parameter Types for PLUS (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
		}
	case ast.BIN_MINUS:
		switch lhsTyp {
		case c.ddpinttyp:
			switch rhsTyp {
			case c.ddpinttyp:
				c.builder().latestReturn = c.builder().CreateSub(lhs, rhs, "")
				c.builder().latestReturnType = c.ddpinttyp
			case c.ddpfloattyp:
				fp := c.builder().CreateSIToFP(lhs, c.ddpfloat, "")
				c.builder().latestReturn = c.builder().CreateFSub(fp, rhs, "")
				c.builder().latestReturnType = c.ddpfloattyp
			case c.ddpbytetyp:
				c.builder().latestReturn = c.builder().CreateSub(lhs, c.floatOrByteAsInt(rhs, c.ddpbytetyp), "")
				c.builder().latestReturnType = c.ddpinttyp
			default:
				c.err("invalid Parameter Types for MINUS (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		case c.ddpfloattyp:
			switch rhsTyp {
			case c.ddpinttyp:
				fp := c.builder().CreateSIToFP(rhs, c.ddpfloat, "")
				c.builder().latestReturn = c.builder().CreateFSub(lhs, fp, "")
			case c.ddpfloattyp:
				c.builder().latestReturn = c.builder().CreateFSub(lhs, rhs, "")
			case c.ddpbytetyp:
				fp := c.builder().CreateUIToFP(rhs, c.ddpfloat, "")
				c.builder().latestReturn = c.builder().CreateFSub(lhs, fp, "")
			default:
				c.err("invalid Parameter Types for MINUS (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
			c.builder().latestReturnType = c.ddpfloattyp
		case c.ddpbytetyp:
			switch rhsTyp {
			case c.ddpinttyp:
				c.builder().latestReturn = c.builder().CreateSub(c.floatOrByteAsInt(lhs, c.ddpbytetyp), rhs, "")
				c.builder().latestReturnType = c.ddpinttyp
			case c.ddpfloattyp:
				fp := c.builder().CreateUIToFP(lhs, c.ddpfloat, "")
				c.builder().latestReturn = c.builder().CreateFSub(fp, rhs, "")
				c.builder().latestReturnType = c.ddpfloattyp
			case c.ddpbytetyp:
				c.builder().latestReturn = c.builder().CreateSub(lhs, rhs, "")
				c.builder().latestReturnType = c.ddpbytetyp
			default:
				c.err("invalid Parameter Types for MINUS (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		default:
			c.err("invalid Parameter Types for MINUS (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
		}
	case ast.BIN_MULT:
		switch lhsTyp {
		case c.ddpinttyp:
			switch rhsTyp {
			case c.ddpinttyp:
				c.builder().latestReturn = c.builder().CreateMul(lhs, rhs, "")
			case c.ddpfloattyp:
				fp := c.builder().CreateSIToFP(lhs, c.ddpfloat, "")
				c.builder().latestReturn = c.builder().CreateFMul(fp, rhs, "")
				c.builder().latestReturnType = c.ddpfloattyp
			case c.ddpbytetyp:
				c.builder().latestReturn = c.builder().CreateMul(lhs, c.floatOrByteAsInt(rhs, c.ddpbytetyp), "")
				c.builder().latestReturnType = c.ddpinttyp
			default:
				c.err("invalid Parameter Types for MAL (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		case c.ddpfloattyp:
			switch rhsTyp {
			case c.ddpinttyp:
				fp := c.builder().CreateSIToFP(rhs, c.ddpfloat, "")
				c.builder().latestReturn = c.builder().CreateFMul(lhs, fp, "")
			case c.ddpfloattyp:
				c.builder().latestReturn = c.builder().CreateFMul(lhs, rhs, "")
			case c.ddpbytetyp:
				fp := c.builder().CreateUIToFP(rhs, c.ddpfloat, "")
				c.builder().latestReturn = c.builder().CreateFMul(lhs, fp, "")
			default:
				c.err("invalid Parameter Types for MAL (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
			c.builder().latestReturnType = c.ddpfloattyp
		case c.ddpbytetyp:
			switch rhsTyp {
			case c.ddpinttyp:
				c.builder().latestReturn = c.builder().CreateMul(c.floatOrByteAsInt(lhs, c.ddpbytetyp), rhs, "")
				c.builder().latestReturnType = c.ddpinttyp
			case c.ddpfloattyp:
				fp := c.builder().CreateUIToFP(lhs, c.ddpfloat, "")
				c.builder().latestReturn = c.builder().CreateFMul(fp, rhs, "")
				c.builder().latestReturnType = c.ddpfloattyp
			case c.ddpbytetyp:
				c.builder().latestReturn = c.builder().CreateMul(lhs, rhs, "")
			default:
				c.err("invalid Parameter Types for MAL (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		default:
			c.err("invalid Parameter Types for MAL (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
		}
	case ast.BIN_DIV:
		switch lhsTyp {
		case c.ddpinttyp:
			switch rhsTyp {
			case c.ddpinttyp:
				lhs = c.builder().CreateSIToFP(lhs, c.ddpfloat, "")
				rhs = c.builder().CreateSIToFP(rhs, c.ddpfloat, "")
				c.builder().latestReturn = c.builder().CreateFDiv(lhs, rhs, "")
			case c.ddpfloattyp:
				fp := c.builder().CreateSIToFP(lhs, c.ddpfloat, "")
				c.builder().latestReturn = c.builder().CreateFDiv(fp, rhs, "")
			case c.ddpbytetyp:
				lhs = c.builder().CreateSIToFP(lhs, c.ddpfloat, "")
				rhs = c.builder().CreateUIToFP(rhs, c.ddpfloat, "")
				c.builder().latestReturn = c.builder().CreateFDiv(lhs, rhs, "")
			default:
				c.err("invalid Parameter Types for DURCH (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		case c.ddpfloattyp:
			switch rhsTyp {
			case c.ddpinttyp:
				fp := c.builder().CreateSIToFP(rhs, c.ddpfloat, "")
				c.builder().latestReturn = c.builder().CreateFDiv(lhs, fp, "")
			case c.ddpfloattyp:
				c.builder().latestReturn = c.builder().CreateFDiv(lhs, rhs, "")
			case c.ddpbytetyp:
				fp := c.builder().CreateUIToFP(rhs, c.ddpfloat, "")
				c.builder().latestReturn = c.builder().CreateFDiv(lhs, fp, "")
			default:
				c.err("invalid Parameter Types for DURCH (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		case c.ddpbytetyp:
			switch rhsTyp {
			case c.ddpinttyp:
				lhs = c.builder().CreateUIToFP(lhs, c.ddpfloat, "")
				rhs = c.builder().CreateSIToFP(rhs, c.ddpfloat, "")
				c.builder().latestReturn = c.builder().CreateFDiv(lhs, rhs, "")
			case c.ddpfloattyp:
				fp := c.builder().CreateSIToFP(lhs, c.ddpfloat, "")
				c.builder().latestReturn = c.builder().CreateFDiv(fp, rhs, "")
			case c.ddpbytetyp:
				lhs = c.builder().CreateUIToFP(lhs, c.ddpfloat, "")
				rhs = c.builder().CreateUIToFP(rhs, c.ddpfloat, "")
				c.builder().latestReturn = c.builder().CreateFDiv(lhs, rhs, "")
			default:
				c.err("invalid Parameter Types for DURCH (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		default:
			c.err("invalid Parameter Types for DURCH (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
		}
		c.builder().latestReturnType = c.ddpfloattyp
	case ast.BIN_INDEX:
		switch lhsTyp {
		case c.ddpstring:
			c.builder().latestReturn = c.builder().CreateCall(c.ddpint, c.ddpstring.indexIrFun, []llvm.Value{lhs, c.floatOrByteAsInt(rhs, rhsTyp)}, "")
			c.builder().latestReturnType = c.ddpchartyp
		default:
			if listType, isList := lhsTyp.(*ddpIrListType); isList {
				listLen := c.loadStructField(listType.typ, lhs, list_len_field_index)
				index := c.builder().CreateSub(c.floatOrByteAsInt(rhs, rhsTyp), c.newInt(1), "") // ddp indices start at 1, so subtract 1
				// index bounds check
				cond := c.builder().CreateAnd(c.builder().CreateICmp(llvm.IntSLT, index, listLen, ""), c.builder().CreateICmp(llvm.IntSGE, index, c.zero, ""), "")
				c.createIfElse(cond, func() {
					listArr := c.loadStructField(listType.typ, lhs, list_arr_field_index)
					elementPtr := c.indexArray(listType.elementType.LLType(), listArr, index)
					// if the list is a temporary, we need to copy the element
					if isTempLhs {
						if listType.elementType.IsPrimitive() { // primitives are simply loaded
							c.builder().latestReturn = c.builder().CreateLoad(listType.elementType.LLType(), elementPtr, "")
						} else {
							dest := c.NewAlloca(listType.elementType.LLType())
							c.builder().latestReturn, c.builder().latestReturnType = c.builder().scp.addTemporary(
								c.deepCopyInto(dest, elementPtr, listType.elementType),
								listType.elementType,
							)
							c.builder().latestIsTemp = true // the element is now also a temporary
						}
					} else {
						if listType.elementType.IsPrimitive() {
							c.builder().latestReturn = c.builder().CreateLoad(listType.elementType.LLType(), elementPtr, "")
						} else { // the list is not temporary, so a reference to the element is enough
							c.builder().latestReturn = elementPtr
							c.builder().latestIsTemp = false
						}
					}
				}, func() { // runtime error
					line, column := int64(e.Token().Range.Start.Line), int64(e.Token().Range.Start.Column)
					c.out_of_bounds_error(c.newInt(line), c.newInt(column), rhs, listLen)
				})
				c.builder().latestReturnType = listType.elementType
			} else {
				c.err("invalid Parameter Types for STELLE (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		}
	case ast.BIN_SLICE_FROM, ast.BIN_SLICE_TO:
		dest := c.NewAlloca(lhsTyp.LLType())
		rhs = c.floatOrByteAsInt(rhs, rhsTyp)

		switch lhsTyp {
		case c.ddpstring:
			if e.Operator == ast.BIN_SLICE_FROM {
				str_len := c.builder().CreateCall(c.ddpint, c.ddpstring.lengthIrFun, []llvm.Value{lhs}, "")
				c.builder().CreateCall(c.void, c.ddpstring.sliceIrFun, []llvm.Value{dest, lhs, rhs, str_len}, "")
			} else {
				c.builder().CreateCall(c.void, c.ddpstring.sliceIrFun, []llvm.Value{dest, lhs, c.newInt(1), rhs}, "")
			}
		default:
			if listTyp, isList := lhsTyp.(*ddpIrListType); isList {
				if e.Operator == ast.BIN_SLICE_FROM {
					lst_len := c.loadStructField(listTyp.typ, lhs, list_len_field_index)
					c.builder().CreateCall(c.void, listTyp.sliceIrFun, []llvm.Value{dest, lhs, rhs, lst_len}, "")
				} else {
					c.builder().CreateCall(c.void, listTyp.sliceIrFun, []llvm.Value{dest, lhs, c.newInt(1), rhs}, "")
				}
			} else {
				c.err("invalid Parameter Types for %s (%s, %s)", e.Operator.String(), lhsTyp.Name(), rhsTyp.Name())
			}
		}
		c.builder().latestReturn, c.builder().latestReturnType = c.builder().scp.addTemporary(dest, lhsTyp)
		c.builder().latestIsTemp = true
	case ast.BIN_POW:
		switch lhsTyp {
		case c.ddpinttyp:
			lhs = c.builder().CreateSIToFP(lhs, c.ddpfloat, "")
		case c.ddpbytetyp:
			lhs = c.builder().CreateUIToFP(lhs, c.ddpfloat, "")
		case c.ddpfloattyp:
		default:
			c.err("invalid Parameter Types for HOCH (Lhs: %s)", lhsTyp.Name())
		}
		switch rhsTyp {
		case c.ddpinttyp:
			rhs = c.builder().CreateSIToFP(rhs, c.ddpfloat, "")
		case c.ddpbytetyp:
			rhs = c.builder().CreateUIToFP(rhs, c.ddpfloat, "")
		case c.ddpfloattyp:
		default:
			c.err("invalid Parameter Types for HOCH (Rhs: %s)", lhsTyp.Name())
		}
		llFuncBuilder := c.functions["pow"].llFuncBuilder
		c.builder().latestReturn = c.builder().CreateCall(llFuncBuilder.llFnType.ReturnType(), llFuncBuilder.llFn, []llvm.Value{lhs, rhs}, "")
		c.builder().latestReturnType = c.ddpfloattyp
	case ast.BIN_LOG:
		switch lhsTyp {
		case c.ddpinttyp:
			lhs = c.builder().CreateSIToFP(lhs, c.ddpfloat, "")
		case c.ddpbytetyp:
			lhs = c.builder().CreateUIToFP(lhs, c.ddpfloat, "")
		case c.ddpfloattyp:
		default:
			c.err("invalid Parameter Types for Logarithmus (Lhs: %s)", lhsTyp.Name())
		}
		switch rhsTyp {
		case c.ddpinttyp:
			rhs = c.builder().CreateSIToFP(rhs, c.ddpfloat, "")
		case c.ddpbytetyp:
			rhs = c.builder().CreateUIToFP(rhs, c.ddpfloat, "")
		case c.ddpfloattyp:
		default:
			c.err("invalid Parameter Types for Logarithmus (Rhs: %s)", lhsTyp.Name())
		}
		log10FuncBuilder := c.functions["log10"].llFuncBuilder
		log10_num := c.builder().CreateCall(log10FuncBuilder.llFnType.ReturnType(), log10FuncBuilder.llFn, []llvm.Value{lhs}, "")
		log10_base := c.builder().CreateCall(log10FuncBuilder.llFnType.ReturnType(), log10FuncBuilder.llFn, []llvm.Value{rhs}, "")
		c.builder().latestReturn = c.builder().CreateFDiv(log10_num, log10_base, "")
		c.builder().latestReturnType = c.ddpfloattyp
	case ast.BIN_LOGIC_AND:
		if lhsTyp == c.ddpinttyp || rhsTyp == c.ddpinttyp {
			lhs, rhs = c.floatOrByteAsInt(lhs, lhsTyp), c.floatOrByteAsInt(rhs, rhsTyp)
			c.builder().latestReturnType = c.ddpinttyp
		} else {
			c.builder().latestReturnType = c.ddpbytetyp
		}
		c.builder().latestReturn = c.builder().CreateAnd(lhs, rhs, "")
	case ast.BIN_LOGIC_OR:
		if lhsTyp == c.ddpinttyp || rhsTyp == c.ddpinttyp {
			lhs, rhs = c.floatOrByteAsInt(lhs, lhsTyp), c.floatOrByteAsInt(rhs, rhsTyp)
			c.builder().latestReturnType = c.ddpinttyp
		} else {
			c.builder().latestReturnType = c.ddpbytetyp
		}
		c.builder().latestReturn = c.builder().CreateOr(lhs, rhs, "")
	case ast.BIN_LOGIC_XOR:
		if lhsTyp == c.ddpinttyp || rhsTyp == c.ddpinttyp {
			lhs, rhs = c.floatOrByteAsInt(lhs, lhsTyp), c.floatOrByteAsInt(rhs, rhsTyp)
			c.builder().latestReturnType = c.ddpinttyp
		} else {
			c.builder().latestReturnType = c.ddpbytetyp
		}
		c.builder().latestReturn = c.builder().CreateXor(lhs, rhs, "")
	case ast.BIN_MOD:
		if lhsTyp == c.ddpbytetyp && rhsTyp == c.ddpbytetyp {
			c.builder().latestReturn = c.builder().CreateURem(lhs, rhs, "")
			c.builder().latestReturnType = c.ddpbytetyp
		} else {
			c.builder().latestReturn = c.builder().CreateSRem(c.floatOrByteAsInt(lhs, lhsTyp), c.floatOrByteAsInt(rhs, rhsTyp), "")
			c.builder().latestReturnType = c.ddpinttyp
		}
	case ast.BIN_LEFT_SHIFT:
		if lhsTyp == c.ddpinttyp || rhsTyp == c.ddpinttyp {
			lhs, rhs = c.floatOrByteAsInt(lhs, lhsTyp), c.floatOrByteAsInt(rhs, rhsTyp)
			c.builder().latestReturnType = c.ddpinttyp
		} else {
			c.builder().latestReturnType = c.ddpbytetyp
		}
		c.builder().latestReturn = c.builder().CreateShl(lhs, rhs, "")
	case ast.BIN_RIGHT_SHIFT:
		if lhsTyp == c.ddpinttyp || rhsTyp == c.ddpinttyp {
			lhs, rhs = c.floatOrByteAsInt(lhs, lhsTyp), c.floatOrByteAsInt(rhs, rhsTyp)
			c.builder().latestReturnType = c.ddpinttyp
		} else {
			c.builder().latestReturnType = c.ddpbytetyp
		}
		c.builder().latestReturn = c.builder().CreateLShr(lhs, rhs, "")
	case ast.BIN_EQUAL:
		c.compare_values(lhs, rhs, lhsTyp)
	case ast.BIN_UNEQUAL:
		equal := c.compare_values(lhs, rhs, lhsTyp)
		c.builder().latestReturn = c.builder().CreateXor(equal, c.newInt(1), "")
	case ast.BIN_LESS:
		switch lhsTyp {
		case c.ddpinttyp:
			switch rhsTyp {
			case c.ddpinttyp, c.ddpbytetyp:
				c.builder().latestReturn = c.builder().CreateICmp(llvm.IntSLT, lhs, c.floatOrByteAsInt(rhs, rhsTyp), "")
			case c.ddpfloattyp:
				fp := c.builder().CreateSIToFP(lhs, c.ddpfloat, "")
				c.builder().latestReturn = c.builder().CreateFCmp(llvm.FloatOLT, fp, rhs, "")
			default:
				c.err("invalid Parameter Types for KLEINER (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		case c.ddpfloattyp:
			switch rhsTyp {
			case c.ddpinttyp:
				fp := c.builder().CreateSIToFP(rhs, c.ddpfloat, "")
				c.builder().latestReturn = c.builder().CreateFCmp(llvm.FloatOLT, lhs, fp, "")
			case c.ddpfloattyp:
				c.builder().latestReturn = c.builder().CreateFCmp(llvm.FloatOLT, lhs, rhs, "")
			case c.ddpbytetyp:
				fp := c.builder().CreateUIToFP(rhs, c.ddpfloat, "")
				c.builder().latestReturn = c.builder().CreateFCmp(llvm.FloatOLT, lhs, fp, "")
			default:
				c.err("invalid Parameter Types for KLEINER (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		case c.ddpbytetyp:
			switch rhsTyp {
			case c.ddpinttyp:
				c.builder().latestReturn = c.builder().CreateICmp(llvm.IntSLT, c.floatOrByteAsInt(lhs, lhsTyp), rhs, "")
			case c.ddpfloattyp:
				fp := c.builder().CreateUIToFP(lhs, c.ddpfloat, "")
				c.builder().latestReturn = c.builder().CreateFCmp(llvm.FloatOLT, fp, rhs, "")
			case c.ddpbytetyp:
				c.builder().latestReturn = c.builder().CreateICmp(llvm.IntULT, lhs, rhs, "")
			default:
				c.err("invalid Parameter Types for KLEINER (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		}
		c.builder().latestReturnType = c.ddpbooltyp
	case ast.BIN_LESS_EQ:
		switch lhsTyp {
		case c.ddpinttyp:
			switch rhsTyp {
			case c.ddpinttyp, c.ddpbytetyp:
				c.builder().latestReturn = c.builder().CreateICmp(llvm.IntSLE, lhs, c.floatOrByteAsInt(rhs, rhsTyp), "")
			case c.ddpfloattyp:
				fp := c.builder().CreateSIToFP(lhs, c.ddpfloat, "")
				c.builder().latestReturn = c.builder().CreateFCmp(llvm.FloatOLE, fp, rhs, "")
			default:
				c.err("invalid Parameter Types for KLEINER_ALS_ODER (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		case c.ddpfloattyp:
			switch rhsTyp {
			case c.ddpinttyp:
				fp := c.builder().CreateSIToFP(rhs, c.ddpfloat, "")
				c.builder().latestReturn = c.builder().CreateFCmp(llvm.FloatOLE, lhs, fp, "")
			case c.ddpfloattyp:
				c.builder().latestReturn = c.builder().CreateFCmp(llvm.FloatOLE, lhs, rhs, "")
			case c.ddpbytetyp:
				fp := c.builder().CreateUIToFP(rhs, c.ddpfloat, "")
				c.builder().latestReturn = c.builder().CreateFCmp(llvm.FloatOLE, lhs, fp, "")
			default:
				c.err("invalid Parameter Types for KLEINER_ALS_ODER (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		case c.ddpbytetyp:
			switch rhsTyp {
			case c.ddpinttyp:
				c.builder().latestReturn = c.builder().CreateICmp(llvm.IntSLE, c.floatOrByteAsInt(lhs, lhsTyp), rhs, "")
			case c.ddpfloattyp:
				fp := c.builder().CreateUIToFP(lhs, c.ddpfloat, "")
				c.builder().latestReturn = c.builder().CreateFCmp(llvm.FloatOLE, fp, rhs, "")
			case c.ddpbytetyp:
				c.builder().latestReturn = c.builder().CreateICmp(llvm.IntULE, lhs, rhs, "")
			default:
				c.err("invalid Parameter Types for KLEINER_ALS_ODER (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		}
		c.builder().latestReturnType = c.ddpbooltyp
	case ast.BIN_GREATER:
		switch lhsTyp {
		case c.ddpinttyp:
			switch rhsTyp {
			case c.ddpinttyp, c.ddpbytetyp:
				c.builder().latestReturn = c.builder().CreateICmp(llvm.IntSGT, lhs, c.floatOrByteAsInt(rhs, rhsTyp), "")
			case c.ddpfloattyp:
				fp := c.builder().CreateSIToFP(lhs, c.ddpfloat, "")
				c.builder().latestReturn = c.builder().CreateFCmp(llvm.FloatOGT, fp, rhs, "")
			default:
				c.err("invalid Parameter Types for GRÖßER (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		case c.ddpfloattyp:
			switch rhsTyp {
			case c.ddpinttyp:
				fp := c.builder().CreateSIToFP(rhs, c.ddpfloat, "")
				c.builder().latestReturn = c.builder().CreateFCmp(llvm.FloatOGT, lhs, fp, "")
			case c.ddpfloattyp:
				c.builder().latestReturn = c.builder().CreateFCmp(llvm.FloatOGT, lhs, rhs, "")
			case c.ddpbytetyp:
				fp := c.builder().CreateUIToFP(rhs, c.ddpfloat, "")
				c.builder().latestReturn = c.builder().CreateFCmp(llvm.FloatOGT, lhs, fp, "")
			default:
				c.err("invalid Parameter Types for GRÖßER (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		case c.ddpbytetyp:
			switch rhsTyp {
			case c.ddpinttyp:
				c.builder().latestReturn = c.builder().CreateICmp(llvm.IntSGT, c.floatOrByteAsInt(lhs, lhsTyp), rhs, "")
			case c.ddpfloattyp:
				fp := c.builder().CreateUIToFP(lhs, c.ddpfloat, "")
				c.builder().latestReturn = c.builder().CreateFCmp(llvm.FloatOGT, fp, rhs, "")
			case c.ddpbytetyp:
				c.builder().latestReturn = c.builder().CreateICmp(llvm.IntUGT, lhs, rhs, "")
			default:
				c.err("invalid Parameter Types for GRÖßER (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		}
		c.builder().latestReturnType = c.ddpbooltyp
	case ast.BIN_GREATER_EQ:
		switch lhsTyp {
		case c.ddpinttyp:
			switch rhsTyp {
			case c.ddpinttyp, c.ddpbytetyp:
				c.builder().latestReturn = c.builder().CreateICmp(llvm.IntSGE, lhs, c.floatOrByteAsInt(rhs, rhsTyp), "")
			case c.ddpfloattyp:
				fp := c.builder().CreateSIToFP(lhs, c.ddpfloat, "")
				c.builder().latestReturn = c.builder().CreateFCmp(llvm.FloatOGE, fp, rhs, "")
			default:
				c.err("invalid Parameter Types for GRÖßER_ALS_ODER (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		case c.ddpfloattyp:
			switch rhsTyp {
			case c.ddpinttyp:
				fp := c.builder().CreateSIToFP(rhs, c.ddpfloat, "")
				c.builder().latestReturn = c.builder().CreateFCmp(llvm.FloatOGE, lhs, fp, "")
			case c.ddpfloattyp:
				c.builder().latestReturn = c.builder().CreateFCmp(llvm.FloatOGE, lhs, rhs, "")
			case c.ddpbytetyp:
				fp := c.builder().CreateUIToFP(rhs, c.ddpfloat, "")
				c.builder().latestReturn = c.builder().CreateFCmp(llvm.FloatOGE, lhs, fp, "")
			default:
				c.err("invalid Parameter Types for GRÖßER_ALS_ODER (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		case c.ddpbytetyp:
			switch rhsTyp {
			case c.ddpinttyp:
				c.builder().latestReturn = c.builder().CreateICmp(llvm.IntSGE, c.floatOrByteAsInt(lhs, lhsTyp), rhs, "")
			case c.ddpfloattyp:
				fp := c.builder().CreateUIToFP(lhs, c.ddpfloat, "")
				c.builder().latestReturn = c.builder().CreateFCmp(llvm.FloatOGE, fp, rhs, "")
			case c.ddpbytetyp:
				c.builder().latestReturn = c.builder().CreateICmp(llvm.IntUGE, lhs, rhs, "")
			default:
				c.err("invalid Parameter Types for GRÖßER_ALS_ODER (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		}
		c.builder().latestReturnType = c.ddpbooltyp
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
		trueBlock, falseBlock, leaveBlock := c.builder().newBlock(), c.builder().newBlock(), c.builder().newBlock()
		c.builder().CreateCondBr(mid, trueBlock, falseBlock)

		c.builder().setBlock(trueBlock)
		// collect temporaries because of possible short-circuiting
		c.builder().scp = newScope(c.builder().scp)
		lhs, lhsTyp, lhsIsTemp := c.evaluate(e.Lhs)
		// claim the temporary, as the phi instruction will become the actual temporary
		if lhsIsTemp && !lhsTyp.IsPrimitive() {
			lhs = c.builder().scp.claimTemporary(lhs)
		}
		// free temporaries
		c.builder().scp = c.exitScope(c.builder().scp)
		c.builder().CreateBr(leaveBlock)
		trueBlock = c.builder().cb

		c.builder().setBlock(falseBlock)
		// collect temporaries because of possible short-circuiting
		c.builder().scp = newScope(c.builder().scp)
		rhs, rhsTyp, rhsIsTemp := c.evaluate(e.Rhs)
		// claim the temporary, as the phi instruction will become the actual temporary
		if rhsIsTemp && !rhsTyp.IsPrimitive() {
			rhs = c.builder().scp.claimTemporary(rhs)
		}
		// free temporaries
		c.builder().scp = c.exitScope(c.builder().scp)
		c.builder().CreateBr(leaveBlock)
		falseBlock = c.builder().cb

		// simple case, where both can be treated the same way
		if lhsIsTemp == rhsIsTemp {
			c.builder().latestIsTemp = lhsIsTemp
		} else {
			c.builder().latestIsTemp = true

			// we need to copy the non-temp value to be sure
			c.builder().setBlock(trueBlock)
			if lhsIsTemp {
				c.builder().setBlock(falseBlock)
			}

			// turn the non-temp into a temporary and claim the temporary,
			// as the phi instruction will become the actual temporary
			dest := c.NewAlloca(lhsTyp.LLType())
			if lhsIsTemp {
				rhs = c.deepCopyInto(dest, rhs, lhsTyp)
			} else {
				lhs = c.deepCopyInto(dest, lhs, lhsTyp)
			}
		}

		c.builder().setBlock(leaveBlock)
		phi := c.builder().CreatePHI(rhsTyp.LLType(), "")
		phi.AddIncoming([]llvm.Value{lhs, rhs}, []llvm.BasicBlock{trueBlock, falseBlock})
		c.builder().latestReturn = phi
		if c.builder().latestIsTemp {
			c.builder().scp.addTemporary(c.builder().latestReturn, lhsTyp)
		}
		c.builder().latestReturnType = lhsTyp
		return ast.VisitRecurse
	}

	lhs, lhsTyp, _ := c.evaluate(e.Lhs)
	mid, midTyp, _ := c.evaluate(e.Mid)
	rhs, rhsTyp, _ := c.evaluate(e.Rhs)

	switch e.Operator {
	case ast.TER_SLICE:
		dest := c.NewAlloca(lhsTyp.LLType())
		mid = c.floatOrByteAsInt(mid, midTyp)
		rhs = c.floatOrByteAsInt(rhs, rhsTyp)
		switch lhsTyp {
		case c.ddpstring:
			c.builder().CreateCall(c.void, c.ddpstring.sliceIrFun, []llvm.Value{dest, lhs, mid, rhs}, "")
		default:
			if listTyp, isList := lhsTyp.(*ddpIrListType); isList {
				c.builder().CreateCall(c.void, listTyp.sliceIrFun, []llvm.Value{dest, lhs, mid, rhs}, "")
			} else {
				c.err("invalid Parameter Types for VONBIS (%s, %s, %s)", lhsTyp.Name(), midTyp.Name(), rhsTyp.Name())
			}
		}
		c.builder().latestReturn, c.builder().latestReturnType = c.builder().scp.addTemporary(dest, lhsTyp)
		c.builder().latestIsTemp = true
	case ast.TER_BETWEEN:
		// lhs zwischen mid und rhs
		// = (lhs > rhs && lhs < mid) || (lhs > mid && lhs < rhs)
		if lhsTyp == c.ddpfloattyp || rhsTyp == c.ddpfloattyp || midTyp == c.ddpfloattyp {
			lhs, mid, rhs = c.intOrByteAsFloat(lhs, lhsTyp), c.intOrByteAsFloat(mid, midTyp), c.intOrByteAsFloat(rhs, rhsTyp)
			c.builder().latestReturn = c.builder().CreateOr(
				c.builder().CreateAnd(c.builder().CreateFCmp(llvm.FloatOGT, lhs, rhs, ""), c.builder().CreateFCmp(llvm.FloatOLT, lhs, mid, ""), ""),
				c.builder().CreateAnd(c.builder().CreateFCmp(llvm.FloatOGT, lhs, mid, ""), c.builder().CreateFCmp(llvm.FloatOLT, lhs, rhs, ""), ""),
				"",
			)
		} else if lhsTyp == c.ddpbytetyp && rhsTyp == c.ddpbytetyp && midTyp == c.ddpbytetyp {
			c.builder().latestReturn = c.builder().CreateOr(
				c.builder().CreateAnd(c.builder().CreateICmp(llvm.IntUGT, lhs, rhs, ""), c.builder().CreateICmp(llvm.IntULT, lhs, mid, ""), ""),
				c.builder().CreateAnd(c.builder().CreateICmp(llvm.IntUGT, lhs, mid, ""), c.builder().CreateICmp(llvm.IntULT, lhs, rhs, ""), ""),
				"",
			)
		} else {
			lhs, mid, rhs = c.floatOrByteAsInt(lhs, lhsTyp), c.floatOrByteAsInt(mid, midTyp), c.floatOrByteAsInt(rhs, rhsTyp)
			c.builder().latestReturn = c.builder().CreateOr(
				c.builder().CreateAnd(c.builder().CreateICmp(llvm.IntSGT, lhs, rhs, ""), c.builder().CreateICmp(llvm.IntSLT, lhs, mid, ""), ""),
				c.builder().CreateAnd(c.builder().CreateICmp(llvm.IntSGT, lhs, mid, ""), c.builder().CreateICmp(llvm.IntSLT, lhs, rhs, ""), ""),
				"",
			)
		}

		c.builder().latestReturnType = c.ddpbooltyp
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

		dest := c.NewAlloca(nonPrimTyp.LLType())
		c.createIfElse(c.compareAnyType(lhs, vtable), func() {
			// temporary values can be claimed
			if isTempLhs {
				val_ptr := c.loadAnyValuePtr(lhs, nonPrimTyp.LLType())
				val := c.builder().CreateLoad(nonPrimTyp.LLType(), val_ptr, "")
				c.builder().CreateStore(val, dest)
				c.createIfElse(c.isSmallAny(lhs), func() {}, func() {
					c.ddp_reallocate(val_ptr, c.newInt(int64(c.getTypeSize(nonPrimTyp))), c.zero)
				})
				c.builder().scp.claimTemporary(lhs) // don't call free func on the now invalid any
			} else {
				// non-temporaries are simply deep copied
				c.deepCopyInto(dest, c.loadAnyValuePtr(lhs, nonPrimTyp.LLType()), nonPrimTyp)
			}
			c.builder().latestReturn, c.builder().latestReturnType = c.builder().scp.addTemporary(dest, nonPrimTyp)
			c.builder().latestIsTemp = true
		}, func() {
			line, column := int64(e.Token().Range.Start.Line), int64(e.Token().Range.Start.Column)
			c.runtime_error(1, c.bad_cast_error_string, c.newInt(line), c.newInt(column))
		})
	}

	if ddptypes.IsList(targetType) {
		if lhsTyp == c.ddpany {
			nonPrimitiveAnyCast()
			return ast.VisitRecurse
		}

		listType := c.getListType(lhsTyp)
		list := c.NewAlloca(listType.typ)
		c.builder().CreateCall(c.void, listType.fromConstantsIrFun, []llvm.Value{list, c.newInt(1)}, "")
		elementPtr := c.indexArray(listType.elementType.LLType(), c.loadStructField(listType.typ, list, list_arr_field_index), c.zero)
		c.claimOrCopy(elementPtr, lhs, lhsTyp, isTempLhs)
		c.builder().latestReturn, c.builder().latestReturnType = c.builder().scp.addTemporary(list, listType)
		c.builder().latestIsTemp = true
	} else {
		// helper function to cast primitive from any to their concrete type
		primitiveAnyCast := func(primTyp ddpIrType) {
			c.createIfElse(c.compareAnyType(lhs, vtable), func() {
				c.builder().latestReturn = c.loadSmallAnyValue(lhs, primTyp.LLType())
				c.builder().latestReturnType, c.builder().latestIsTemp = primTyp, true
			}, func() {
				line, column := int64(e.Token().Range.Start.Line), int64(e.Token().Range.Start.Column)
				c.runtime_error(1, c.bad_cast_error_string, c.newInt(line), c.newInt(column))
			})
		}

		switch targetType {
		case ddptypes.ZAHL:
			switch lhsTyp {
			case c.ddpinttyp, c.ddpfloattyp, c.ddpbytetyp:
				c.builder().latestReturn = c.numericCast(lhs, lhsTyp, c.toIrType(targetType))
			case c.ddpbooltyp:
				cond := c.builder().CreateICmp(llvm.IntNE, lhs, c.zero, "")
				c.builder().latestReturn = c.builder().CreateZExt(cond, c.ddpint, "")
			case c.ddpchartyp:
				c.builder().latestReturn = c.builder().CreateSExt(lhs, c.ddpint, "")
			case c.ddpstring:
				c.builder().latestReturn = c.builder().CreateCall(c.ddpint, c.functions["ddp_string_to_int"].irFunc, []llvm.Value{lhs}, "")
			case c.ddpany:
				primitiveAnyCast(c.ddpinttyp)
			default:
				c.err("invalid Parameter Type for ZAHL: %s", lhsTyp.Name())
			}
		case ddptypes.KOMMAZAHL:
			switch lhsTyp {
			case c.ddpinttyp, c.ddpfloattyp, c.ddpbytetyp:
				c.builder().latestReturn = c.numericCast(lhs, lhsTyp, c.toIrType(targetType))
			case c.ddpstring:
				c.builder().latestReturn = c.builder().CreateCall(c.ddpfloat, c.functions["ddp_string_to_float"].irFunc, []llvm.Value{lhs}, "")
			case c.ddpany:
				primitiveAnyCast(c.ddpfloattyp)
			default:
				c.err("invalid Parameter Type for KOMMAZAHL: %s", lhsTyp.Name())
			}
		case ddptypes.BYTE:
			switch lhsTyp {
			case c.ddpinttyp, c.ddpfloattyp, c.ddpbytetyp:
				c.builder().latestReturn = c.numericCast(lhs, lhsTyp, c.toIrType(targetType))
			case c.ddpbooltyp:
				cond := c.builder().CreateICmp(llvm.IntNE, lhs, c.zero, "")
				c.builder().latestReturn = c.builder().CreateZExt(cond, c.ddpbyte, "")
			case c.ddpchartyp:
				c.builder().latestReturn = c.builder().CreateTrunc(lhs, c.ddpbyte, "")
			case c.ddpstring:
				intVal := c.builder().CreateCall(c.ddpint, c.functions["ddp_string_to_int"].irFunc, []llvm.Value{lhs}, "")
				c.builder().latestReturn = c.builder().CreateTrunc(intVal, c.ddpbyte, "")
			case c.ddpany:
				primitiveAnyCast(c.ddpinttyp)
			default:
				c.err("invalid Parameter Type for ZAHL: %s", lhsTyp.Name())
			}
		case ddptypes.WAHRHEITSWERT:
			switch lhsTyp {
			case c.ddpinttyp:
				c.builder().latestReturn = c.builder().CreateICmp(llvm.IntNE, lhs, c.zero, "")
			case c.ddpbytetyp:
				c.builder().latestReturn = c.builder().CreateICmp(llvm.IntNE, lhs, c.zero8, "")
			case c.ddpbooltyp:
				c.builder().latestReturn = lhs
			case c.ddpany:
				primitiveAnyCast(c.ddpbooltyp)
			default:
				c.err("invalid Parameter Type for WAHRHEITSWERT: %s", lhsTyp.Name())
			}
		case ddptypes.BUCHSTABE:
			switch lhsTyp {
			case c.ddpinttyp:
				c.builder().latestReturn = c.builder().CreateTrunc(lhs, c.ddpchar, "")
			case c.ddpbytetyp:
				c.builder().latestReturn = c.builder().CreateZExt(lhs, c.ddpchar, "")
			case c.ddpchartyp:
				c.builder().latestReturn = lhs
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
				c.builder().latestReturn = lhs
				c.builder().latestReturnType = c.ddpstring
				c.builder().latestIsTemp = isTempLhs
				return ast.VisitRecurse // don't free lhs
			}

			var to_string_func llvm.Value
			switch lhsTyp {
			case c.ddpinttyp:
				to_string_func = c.ddpstring.int_to_string_IrFun
			case c.ddpfloattyp:
				to_string_func = c.ddpstring.float_to_string_IrFun
			case c.ddpbytetyp:
				to_string_func = c.ddpstring.byte_to_string_IrFun
			case c.ddpbooltyp:
				to_string_func = c.ddpstring.bool_to_string_IrFun
			case c.ddpchartyp:
				to_string_func = c.ddpstring.char_to_string_IrFun
			default:
				c.err("invalid Parameter Type for TEXT: %s", lhsTyp.Name())
			}
			dest := c.NewAlloca(c.ddpstring.typ)
			c.builder().CreateCall(c.void, to_string_func, []llvm.Value{dest, lhs}, "")
			c.builder().latestReturn, c.builder().latestReturnType = c.builder().scp.addTemporary(dest, c.ddpstring)
			c.builder().latestIsTemp = true
		case ddptypes.VARIABLE:
			if lhsTyp == c.ddpany {
				break
			}

			c.builder().latestReturn, c.builder().latestReturnType, c.builder().latestIsTemp = c.castNonAnyToAny(lhs, lhsTyp, isTempLhs, lhsTyp.VTable())
		default:
			if lhsTyp == c.ddpany {
				nonPrimitiveAnyCast()
				return ast.VisitRecurse
			}
			// this is now valid because of typedefs/typealiases
			// c.err("Invalide Typumwandlung zu %s (%s)", e.TargetType, targetType)
		}
	}
	c.builder().latestReturnType = c.toIrType(targetType)
	return ast.VisitRecurse
}

func (c *compiler) VisitCastAssigneable(e *ast.CastAssigneable) ast.VisitResult {
	c.evaluate(&ast.CastExpr{Lhs: e.Lhs, TargetType: e.TargetType, Range: e.Range})
	return ast.VisitRecurse
}

func (c *compiler) VisitTypeOpExpr(e *ast.TypeOpExpr) ast.VisitResult {
	switch e.Operator {
	case ast.TYPE_SIZE:
		c.builder().latestReturn = c.sizeof(c.toIrType(e.Rhs).LLType())
		c.builder().latestReturnType = c.ddpinttyp
	case ast.TYPE_DEFAULT:
		switch t := ddptypes.TrueUnderlying(e.Rhs).(type) {
		case *ddptypes.StructType:
			result, resultType := c.evaluateStructLiteral(t, nil)
			c.builder().latestReturn, c.builder().latestReturnType = c.builder().scp.addTemporary(result, resultType)
		default:
			irType := c.toIrType(e.Rhs)
			var defaultValue llvm.Value = irType.DefaultValue()
			if !irType.IsPrimitive() {
				dest := c.NewAlloca(irType.LLType())
				c.builder().CreateStore(defaultValue, dest)
				defaultValue = dest
			}
			c.builder().latestReturn, c.builder().latestReturnType = c.builder().scp.addTemporary(defaultValue, irType)
		}
	default:
		c.err("invalid TypeOpExpr Operator: %d", e.Operator)
	}
	c.builder().latestIsTemp = true
	return ast.VisitRecurse
}

func (c *compiler) VisitTypeCheck(e *ast.TypeCheck) ast.VisitResult {
	lhs, _, _ := c.evaluate(e.Lhs)

	vtable := c.toIrType(e.CheckType).VTable()
	if typeDef, isTypeDef := ddptypes.CastTypeDef(e.CheckType); isTypeDef {
		vtable = c.typeDefVTables[c.mangledNameType(typeDef)]
	}

	c.builder().latestReturn = c.compareAnyType(lhs, vtable)
	c.builder().latestReturnType = c.ddpbooltyp
	return ast.VisitRecurse
}

func (c *compiler) VisitGrouping(e *ast.Grouping) ast.VisitResult {
	e.Expr.Accept(c) // visit like a normal expression, grouping is just precedence stuff which has already been parsed
	return ast.VisitRecurse
}

// helper for VisitAssignStmt and VisitFuncCall
// if as_ref is true, the assignable is treated as a reference parameter and the third return value can be ignored
// if as_ref is false, the assignable is treated as the lhs in an AssignStmt and might be a string indexing
func (c *compiler) evaluateAssignableOrReference(ass ast.Assigneable, as_ref bool) (llvm.Value, ddpIrType, *ast.Indexing) {
	switch assign := ass.(type) {
	case *ast.Ident:
		Var := c.builder().scp.lookupVar(assign.Declaration.(*ast.VarDecl))
		return Var.val, Var.typ, nil
	case *ast.Indexing:
		lhs, lhsTyp, _ := c.evaluateAssignableOrReference(assign.Lhs, as_ref) // get the (possibly nested) assignable
		if listTyp, isList := lhsTyp.(*ddpIrListType); isList {
			index, indexTyp, _ := c.evaluate(assign.Index)
			index = c.builder().CreateSub(c.floatOrByteAsInt(index, indexTyp), c.newInt(1), "") // ddpindices start at 1
			listLen := c.loadStructField(listTyp.typ, lhs, list_len_field_index)
			var elementPtr llvm.Value

			cond := c.builder().CreateAnd(c.builder().CreateICmp(llvm.IntSLT, index, listLen, ""), c.builder().CreateICmp(llvm.IntSGE, index, c.zero, ""), "")
			c.createIfElse(cond, func() {
				listArr := c.loadStructField(listTyp.typ, lhs, list_arr_field_index)
				elementPtr = c.indexArray(listTyp.elementType.LLType(), listArr, index)
			}, func() { // runtime error
				line, column := int64(assign.Token().Range.Start.Line), int64(assign.Token().Range.Start.Column)
				c.out_of_bounds_error(c.newInt(line), c.newInt(column), c.builder().CreateAdd(index, c.newInt(1), ""), listLen)
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
			fieldPtr := c.indexStruct(structTyp.typ, rhs, fieldIndex)
			return fieldPtr, structTyp.fieldIrTypes[fieldIndex], nil
		} else {
			c.err("non-struct type passed to FieldAccess")
		}
	case *ast.CastAssigneable:
		return c.evaluateAssignableOrReference(assign.Lhs, as_ref)
	}
	c.err("Invalid types in evaluateAssignableOrReference %s", ass)
	return llvm.Value{}, nil, nil
}

func (c *compiler) VisitFuncCall(e *ast.FuncCall) ast.VisitResult {
	mangledName := c.mangledNameDecl(e.Func)
	_, alreadyPresent := c.functions[mangledName] // retreive the function (the resolver took care that it is present)
	needsInstantiation := !alreadyPresent && ast.IsGenericInstantiation(e.Func)

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

	args := make([]llvm.Value, 0, len(fun.funcDecl.Parameters)+1)

	meta := annotators.ConstFuncParamMeta{}
	if attachement, ok := e.Func.Module().Ast.GetMetadataByKind(fun.funcDecl, annotators.ConstFuncParamMetaKind); ok {
		meta = attachement.(annotators.ConstFuncParamMeta)
	}

	irReturnType := c.getPossiblyGenericReturnType(fun.funcDecl)

	var ret llvm.Value
	if !irReturnType.IsPrimitive() {
		ret = c.NewAlloca(irReturnType.LLType())
		args = append(args, ret)
	}

	for _, param := range fun.funcDecl.Parameters {
		var val llvm.Value

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
				dest := c.NewAlloca(valTyp.LLType())
				c.claimOrCopy(dest, eval, valTyp, isTemp)
				val = dest // do not add it to the temporaries, as the callee will free it
			}
		}

		args = append(args, val) // add the value to the arguments
	}

	// compile the actual function call
	if irReturnType.IsPrimitive() {
		c.builder().latestReturn = c.builder().CreateCall(fun.llFuncBuilder.llFnType.ReturnType(), fun.irFunc, args, "")
	} else {
		c.builder().CreateCall(fun.llFuncBuilder.llFnType.ReturnType(), fun.irFunc, args, "")
		c.builder().latestReturn, c.builder().latestReturnType = c.builder().scp.addTemporary(ret, irReturnType)
		c.builder().latestIsTemp = true
	}
	c.builder().latestReturnType = irReturnType

	// the arguments of external functions must be freed by the caller
	// normal functions free their parameters in their body
	if !ast.IsExternFunc(e.Func) {
		return ast.VisitRecurse
	}

	for i, param := range e.Func.Parameters {
		if !param.Type.IsReference {
			paramIrType := c.toIrType(param.Type.Type)
			arg := args[i]
			if !irReturnType.IsPrimitive() {
				arg = args[i+1]
			}

			c.freeNonPrimitive(arg, paramIrType)
		}
	}
	return ast.VisitRecurse
}

func (c *compiler) evaluateStructLiteral(structType *ddptypes.StructType, args map[string]ast.Expression) (llvm.Value, ddpIrType) {
	// search in the types module for the decl, as it might not be present in this module due to transitive dependencies
	structDeclInterface, _, _ := c.typeMap[structType].Ast.Symbols.LookupDecl(structType.Name)
	structDecl := structDeclInterface.(*ast.StructDecl)
	resultType := c.toIrType(structType)
	result := c.NewAlloca(resultType.LLType())
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
			argExpr = &ast.TypeOpExpr{Operator: ast.TYPE_DEFAULT, Rhs: field.Type, Range: c.builder().currentNode.GetRange()}
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

		c.claimOrCopy(c.indexStruct(resultType.LLType(), result, i), argVal, argType, isTempArg)
	}
	return result, resultType
}

func (c *compiler) VisitStructLiteral(expr *ast.StructLiteral) ast.VisitResult {
	result, resultType := c.evaluateStructLiteral(expr.Type, expr.Args)
	c.builder().latestReturn, c.builder().latestReturnType = c.builder().scp.addTemporary(result, resultType)
	c.builder().latestIsTemp = true
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
	if ast.IsGeneric(decl) && !ast.IsExternFunc(decl) {
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

	retType := c.getPossiblyGenericReturnType(decl) // get the llvm type
	retTypeIr := retType.LLType()
	params := make([]llvm.Type, 0, len(decl.Parameters)+1)  // list of the ir parameters
	paramNames := make([]string, 0, len(decl.Parameters)+1) // list of the ir parameters

	hasReturnParam := !retType.IsPrimitive()
	// non-primitives are returned by passing a pointer to the struct as first parameter
	if hasReturnParam {
		params = append(params, c.ptr)
		paramNames = append(paramNames, "")
		retTypeIr = c.voidtyp.LLType()
	}

	// append all the other parameters
	for _, param := range decl.Parameters {
		ty := c.getPossiblyGenericParamType(&param) // convert the type of the parameter
		params = append(params, ty)                 // add it to the list
		paramNames = append(paramNames, param.Name.Literal)
	}

	llFuncTyp := llvm.FunctionType(retTypeIr, params, false)

	llFuncBuilder := c.createBuilder(mangledName, llFuncTyp, paramNames)
	// declare it as extern function
	llFuncBuilder.llFn.SetLinkage(llvm.ExternalLinkage)
	llFuncBuilder.llFn.SetVisibility(llvm.DefaultVisibility)
	c.insertFunction(llFuncBuilder.fnName, decl, llFuncBuilder.llFn, llFuncBuilder)
}

func (c *compiler) declareImportedVarDecl(decl *ast.VarDecl) {
	// imported decls are always in the global scope
	// even in generic instantiations
	scp := c.builder().scp
	for !scp.isGlobalScope() {
		scp = scp.enclosing
	}

	if !scp.lookupVar(decl).val.IsNil() {
		return
	}

	c.declareIfStruct(decl.Type)
	Typ := c.toIrType(decl.Type)
	globalDecl := llvm.AddGlobal(c.llmod, Typ.LLType(), c.mangledNameDecl(decl))
	globalDecl.SetLinkage(llvm.ExternalLinkage)
	globalDecl.SetVisibility(llvm.DefaultVisibility)

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

			moduleInitBuilder := c.createBuilder(init_name, c.void, nil)
			moduleInitBuilder.llFn.SetLinkage(llvm.ExternalLinkage)
			moduleInitBuilder.llFn.SetVisibility(llvm.DefaultVisibility)

			c.insertFunction(moduleInitBuilder.fnName, nil, moduleInitBuilder.llFn, moduleInitBuilder)
			if c.builder().isDDPMain() {
				c.builder().CreateCall(c.void, moduleInitBuilder.llFn, nil, "") // only call this in main modules
			}

			moduleDisposeBuilder := c.createBuilder(dispose_name, c.void, nil)
			moduleDisposeBuilder.llFn.SetLinkage(llvm.ExternalLinkage)
			moduleDisposeBuilder.llFn.SetVisibility(llvm.DefaultVisibility)

			c.insertFunction(moduleDisposeBuilder.fnName, nil, moduleDisposeBuilder.llFn, moduleDisposeBuilder)

			c.importedModules[module] = struct{}{}
		})
	}
	return ast.VisitRecurse
}

func (c *compiler) VisitAssignStmt(s *ast.AssignStmt) ast.VisitResult {
	rhs, rhsTyp, isTempRhs := c.evaluate(s.Rhs) // compile the expression

	lhs, lhsTyp, lhsStringIndexing := c.evaluateAssignableOrReference(s.Var, false)

	// implicit numeric casts
	if ddptypes.IsNumeric(s.VarType) && ddptypes.IsNumeric(s.RhsType) {
		rhs, rhsTyp = c.numericCast(rhs, rhsTyp, lhsTyp), lhsTyp
	}

	if lhsStringIndexing != nil {
		index, indexTyp, _ := c.evaluate(lhsStringIndexing.Index)
		index = c.floatOrByteAsInt(index, indexTyp)
		c.builder().CreateCall(c.void, c.ddpstring.replaceCharIrFun, []llvm.Value{lhs, rhs, index}, "")
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
	c.builder().scp = newScope(c.builder().scp) // a block gets its own scope
	wasReturn := false
	for _, stmt := range s.Statements {
		c.visitNode(stmt)
		if _, ok := stmt.(*ast.ReturnStmt); ok {
			wasReturn = true
			break
		}
	}
	if wasReturn {
		c.builder().scp = c.builder().scp.enclosing
	} else {
		c.builder().scp = c.exitScope(c.builder().scp) // free local variables and return to the previous scope
	}
	return ast.VisitRecurse
}

// for info on how the generated ir works you might want to see https://llir.github.io/document/user-guide/control/#If
func (c *compiler) VisitIfStmt(s *ast.IfStmt) ast.VisitResult {
	cond, _, _ := c.evaluate(s.Condition)
	thenBlock, elseBlock, leaveBlock := c.builder().newBlock(), c.builder().newBlock(), c.builder().newBlock()
	if s.Else != nil {
		c.builder().CreateCondBr(cond, thenBlock, elseBlock)
	} else {
		c.builder().CreateCondBr(cond, thenBlock, leaveBlock)
	}

	c.builder().setBlock(thenBlock)
	c.builder().scp = newScope(c.builder().scp)
	c.visitNode(s.Then)
	if c.builder().cb.Terminator().IsNil() {
		c.builder().CreateBr(leaveBlock)
	}
	c.builder().scp = c.exitScope(c.builder().scp)

	if s.Else != nil {
		c.builder().setBlock(elseBlock)
		c.builder().scp = newScope(c.builder().scp)
		c.visitNode(s.Else)
		if c.builder().cb.Terminator().IsNil() {
			c.builder().CreateBr(leaveBlock)
		}
		c.builder().scp = c.exitScope(c.builder().scp)
	} else {
		c.builder().withBlock(elseBlock, func() { c.builder().CreateUnreachable() })
	}

	c.builder().setBlock(leaveBlock)
	return ast.VisitRecurse
}

// for info on how the generated ir works you might want to see https://llir.github.io/document/user-guide/control/#Loop
func (c *compiler) VisitWhileStmt(s *ast.WhileStmt) ast.VisitResult {
	loopScopeBack, leaveBlockBack, continueBlockBack := c.builder().curLoopScope, c.builder().curLeaveBlock, c.builder().curContinueBlock
	switch op := s.While.Type; op {
	case token.SOLANGE, token.MACHE:
		condBlock, body, bodyScope := c.builder().newBlock(), c.builder().newBlock(), newScope(c.builder().scp)
		breakLeave := c.builder().newBlock()
		c.builder().curLoopScope, c.builder().curLeaveBlock, c.builder().curContinueBlock = bodyScope, breakLeave, condBlock

		if op == token.SOLANGE {
			c.builder().CreateBr(condBlock)
		} else {
			c.builder().CreateBr(body)
		}

		c.builder().setBlock(body)
		c.builder().scp = bodyScope
		c.visitNode(s.Body)
		if c.builder().cb.Terminator().IsNil() {
			c.builder().CreateBr(condBlock)
		}

		c.builder().setBlock(condBlock)
		c.builder().scp = c.exitScope(c.builder().scp)
		cond, _, _ := c.evaluate(s.Condition)
		leaveBlock := c.builder().newBlock()
		c.builder().CreateCondBr(cond, body, leaveBlock)

		trueLeave := c.builder().newBlock()
		c.builder().withBlock(leaveBlock, func() { c.builder().CreateBr(trueLeave) })
		c.builder().withBlock(breakLeave, func() { c.builder().CreateBr(trueLeave) })
		c.builder().setBlock(trueLeave)
	case token.WIEDERHOLE:
		counter := c.NewAlloca(c.ddpint)
		cond, _, _ := c.evaluate(s.Condition)
		c.builder().CreateStore(cond, counter)
		condBlock, body, bodyScope := c.builder().newBlock(), c.builder().newBlock(), newScope(c.builder().scp)
		breakLeave := c.builder().newBlock()
		c.builder().curLoopScope, c.builder().curLeaveBlock, c.builder().curContinueBlock = bodyScope, breakLeave, condBlock

		c.builder().CreateBr(condBlock)

		c.builder().setBlock(body)
		c.builder().scp = bodyScope
		c.builder().CreateStore(c.builder().CreateSub(c.builder().CreateLoad(c.ddpint, counter, ""), c.newInt(1), ""), counter)
		c.visitNode(s.Body)
		if c.builder().cb.Terminator().IsNil() {
			c.builder().CreateBr(condBlock)
		}

		leaveBlock := c.builder().newBlock()
		c.builder().setBlock(condBlock)
		c.builder().scp = c.exitScope(c.builder().scp)
		c.builder().CreateCondBr( // while counter != 0, execute body
			c.builder().CreateICmp(llvm.IntNE, c.builder().CreateLoad(c.ddpint, counter, ""), c.zero, ""),
			body,
			leaveBlock,
		)

		trueLeave := c.builder().newBlock()
		c.builder().withBlock(leaveBlock, func() { c.builder().CreateBr(trueLeave) })
		c.builder().withBlock(breakLeave, func() { c.builder().CreateBr(trueLeave) })
		c.builder().setBlock(trueLeave)
	}
	c.builder().curLoopScope, c.builder().curLeaveBlock, c.builder().curContinueBlock = loopScopeBack, leaveBlockBack, continueBlockBack
	return ast.VisitRecurse
}

// for info on how the generated ir works you might want to see https://llir.github.io/document/user-guide/control/#Loop
func (c *compiler) VisitForStmt(s *ast.ForStmt) ast.VisitResult {
	new_IorF_comp := func(ipred llvm.IntPredicate, fpred llvm.FloatPredicate, x llvm.Value, xType ddpIrType, yi llvm.Value, yiType ddpIrType, yf llvm.Value) llvm.Value {
		if ddptypes.DeepEqual(s.Initializer.Type, ddptypes.BYTE) {
			x, yi = c.floatOrByteAsInt(x, xType), c.floatOrByteAsInt(yi, yiType)
		}

		if ddptypes.DeepEqual(s.Initializer.Type, ddptypes.KOMMAZAHL) {
			return c.builder().CreateFCmp(fpred, x, yf, "")
		} else {
			return c.builder().CreateICmp(ipred, x, yi, "")
		}
	}

	loopScopeBack, leaveBlockBack, continueBlockBack := c.builder().curLoopScope, c.builder().curLeaveBlock, c.builder().curContinueBlock

	c.builder().scp = newScope(c.builder().scp) // scope for the for body
	c.visitNode(s.Initializer)                  // compile the counter variable declaration
	Var := c.builder().scp.lookupVar(s.Initializer)
	// this is the actual index used
	var indexVar llvm.Value
	var indexTyp ddpIrType
	if ddptypes.DeepEqual(s.Initializer.Type, ddptypes.KOMMAZAHL) {
		indexVar, indexTyp = c.NewAlloca(c.ddpfloat), c.ddpfloattyp
	} else {
		indexVar, indexTyp = c.NewAlloca(c.ddpint), c.ddpinttyp
	}
	var incrementer llvm.Value // Schrittgröße
	var incrementerType ddpIrType
	// if no stepsize was present it is 1
	if s.StepSize == nil {
		if ddptypes.DeepEqual(s.Initializer.Type, ddptypes.KOMMAZAHL) {
			incrementer, incrementerType = llvm.ConstFloat(c.ddpfloat, 1.0), c.ddpfloattyp
		} else {
			incrementer, incrementerType = c.newInt(1), c.ddpinttyp
		}
	} else { // stepsize was present, so compile it
		incrementer, incrementerType, _ = c.evaluate(s.StepSize)
	}

	condBlock, incrementBlock, forBody, breakLeave := c.builder().newBlock(), c.builder().newBlock(), c.builder().newBlock(), c.builder().newBlock()

	c.builder().curLoopScope, c.builder().curLeaveBlock, c.builder().curContinueBlock = c.builder().scp, breakLeave, incrementBlock

	c.builder().CreateStore(c.numericCast(c.builder().CreateLoad(Var.typ.LLType(), Var.val, ""), Var.typ, indexTyp), indexVar)

	c.builder().CreateBr(condBlock) // we begin by evaluating the condition (not compiled yet, but the ir starts here)
	// compile the for-body
	c.builder().setBlock(forBody)
	c.visitNode(s.Body)
	if c.builder().cb.Terminator().IsNil() { // if there is no return at the end we jump to the incrementBlock
		c.builder().CreateBr(incrementBlock)
	}

	// compile the incrementBlock
	c.builder().setBlock(incrementBlock)
	indexVal := c.builder().CreateLoad(indexTyp.LLType(), indexVar, "")

	// add the incrementer to the counter variable
	var add llvm.Value
	if ddptypes.DeepEqual(s.Initializer.Type, ddptypes.KOMMAZAHL) {
		add = c.builder().CreateFAdd(indexVal, c.intOrByteAsFloat(incrementer, incrementerType), "")
		c.builder().CreateStore(add, Var.val)
	} else {
		add = c.builder().CreateAdd(indexVal, c.floatOrByteAsInt(incrementer, incrementerType), "")
		c.builder().CreateStore(c.numericCast(add, c.ddpinttyp, Var.typ), Var.val)
	}
	c.builder().CreateStore(add, indexVar)
	c.builder().CreateBr(condBlock) // check the condition (loop)

	// finally compile the condition block(s)
	loopDown, loopUp, leaveBlock := c.builder().newBlock(), c.builder().newBlock(), c.builder().newBlock()

	c.builder().setBlock(condBlock)
	// we check the counter differently depending on wether or not we are looping up or down (positive vs negative stepsize)
	cond := new_IorF_comp(llvm.IntSLT, llvm.FloatOLT, incrementer, incrementerType, c.newInt(0), c.ddpinttyp, llvm.ConstFloat(c.ddpfloat, 0.0))
	c.builder().CreateCondBr(cond, loopDown, loopUp)

	c.builder().setBlock(loopUp)
	// we are counting up, so compare less-or-equal
	to, toType, _ := c.evaluate(s.To)
	cond = new_IorF_comp(llvm.IntSLE, llvm.FloatOLE, c.builder().CreateLoad(indexTyp.LLType(), indexVar, ""), indexTyp, to, toType, to)
	c.builder().CreateCondBr(cond, forBody, leaveBlock)

	c.builder().setBlock(loopDown)
	// we are counting down, so compare greater-or-equal
	to, toType, _ = c.evaluate(s.To)
	cond = new_IorF_comp(llvm.IntSGE, llvm.FloatOGE, c.builder().CreateLoad(indexTyp.LLType(), indexVar, ""), indexTyp, to, toType, to)
	c.builder().CreateCondBr(cond, forBody, leaveBlock)

	c.builder().setBlock(leaveBlock)
	c.builder().scp = c.exitScope(c.builder().scp) // leave the scope

	trueLeave, leaveBlock, breakLeave := c.builder().newBlock(), c.builder().newBlock(), c.builder().newBlock()
	c.builder().setBlock(trueLeave)

	c.builder().curLoopScope, c.builder().curLeaveBlock, c.builder().curContinueBlock = loopScopeBack, leaveBlockBack, continueBlockBack
	return ast.VisitRecurse
}

func (c *compiler) VisitForRangeStmt(s *ast.ForRangeStmt) ast.VisitResult {
	loopScopeBack, leaveBlockBack, continueBlockBack := c.builder().curLoopScope, c.builder().curLeaveBlock, c.builder().curContinueBlock

	c.builder().scp = newScope(c.builder().scp)
	in, inTyp, isTempIn := c.evaluate(s.In)

	temp := c.NewAlloca(inTyp.LLType())
	c.claimOrCopy(temp, in, inTyp, isTempIn)
	in, _ = c.builder().scp.addTemporary(temp, inTyp)
	c.builder().scp.protectTemporary(in)

	var (
		end_ptr llvm.Value // points to the one-after-last element
		length  llvm.Value
		index   llvm.Value = c.NewAlloca(c.ddpinttyp.LLType())
	)

	iter_ptr := c.NewAlloca(c.ptr) // pointer used for iteration

	if inTyp == c.ddpstring {
		iter_ptr_val := c.loadStructField(c.ddpstring.typ, in, string_str_field_index)
		c.builder().CreateStore(iter_ptr_val, iter_ptr)
		length = c.loadStructField(c.ddpstring.typ, in, string_cap_field_index)
		end_ptr = c.indexArray(c.ddpchar, iter_ptr_val, c.builder().CreateSub(length, c.newInt(1), ""))
	} else {
		iter_ptr_val := c.loadStructField(inTyp.LLType(), in, list_arr_field_index)
		c.builder().CreateStore(iter_ptr_val, iter_ptr)
		length = c.loadStructField(inTyp.LLType(), in, list_len_field_index)
		end_ptr = c.indexArray(inTyp.(*ddpIrListType).elementType.LLType(), iter_ptr_val, length)
	}

	loopStart, condBlock, bodyBlock, incrementBlock, leaveBlock := c.builder().newBlock(), c.builder().newBlock(), c.builder().newBlock(), c.builder().newBlock(), c.builder().newBlock()
	c.builder().CreateCondBr(c.builder().CreateICmp(llvm.IntEQ, length, c.zero, ""), leaveBlock, loopStart)

	c.builder().setBlock(loopStart)
	irType := c.toIrType(s.Initializer.Type)
	c.builder().scp.addProtected(s.Initializer, c.NewAlloca(irType.LLType()), irType, false)
	if s.Index != nil {
		c.builder().scp.addVar(s.Index, index, c.ddpinttyp, false)
		c.builder().CreateStore(c.newInt(1), index)
	}
	c.builder().CreateBr(condBlock)

	c.builder().setBlock(condBlock)
	c.builder().CreateCondBr(c.builder().CreateICmp(llvm.IntNE, c.builder().CreateLoad(c.ptr, iter_ptr, ""), end_ptr, ""), bodyBlock, leaveBlock)

	loopVar := c.builder().scp.lookupVar(s.Initializer)

	continueBlock := c.builder().newBlock()
	c.builder().setBlock(continueBlock)
	c.freeNonPrimitive(loopVar.val, loopVar.typ)
	c.builder().CreateBr(incrementBlock)

	c.builder().setBlock(bodyBlock)
	var num_bytes llvm.Value
	if inTyp == c.ddpstring {
		num_bytes = c.builder().CreateCall(c.ddpchar, utf8_string_to_char_irfun,
			[]llvm.Value{
				c.builder().CreateLoad(c.ptr, iter_ptr, ""),
				loopVar.val,
			},
			"",
		)
		c.createIfElse(c.builder().CreateICmp(llvm.IntEQ, num_bytes, c.all_ones, ""), func() {
			line, column := int64(s.In.Token().Range.Start.Line), int64(s.In.Token().Range.Start.Column)
			c.runtime_error(1, c.invalid_utf8_error_string, c.newInt(line), c.newInt(column))
		}, func() {})
	} else {
		elementPtr := c.builder().CreateLoad(c.ptr, iter_ptr, "")
		inListTyp := inTyp.(*ddpIrListType)
		if inListTyp.elementType.IsPrimitive() {
			element := c.builder().CreateLoad(inListTyp.elementType.LLType(), elementPtr, "")
			c.builder().CreateStore(element, loopVar.val)
		} else {
			c.deepCopyInto(loopVar.val, elementPtr, inListTyp.elementType)
		}
	}
	breakLeave := c.builder().newBlock()
	c.builder().withBlock(breakLeave, func() { c.builder().CreateBr(leaveBlock) })
	c.builder().curLoopScope, c.builder().curLeaveBlock, c.builder().curContinueBlock = c.builder().scp, breakLeave, continueBlock
	c.visitNode(s.Body)
	c.freeNonPrimitive(loopVar.val, loopVar.typ)
	if c.builder().cb.Terminator().IsNil() {
		c.builder().CreateBr(incrementBlock)
	}

	c.builder().setBlock(incrementBlock)
	if inTyp == c.ddpstring {
		c.builder().CreateStore(
			c.builder().CreateIntToPtr(
				c.builder().CreateAdd(
					c.builder().CreatePtrToInt(c.builder().CreateLoad(c.ptr, iter_ptr, ""), c.ddpint, ""),
					num_bytes,
					"",
				),
				c.ptr,
				"",
			),
			iter_ptr,
		)
	} else {
		inListTyp := inTyp.(*ddpIrListType)
		c.builder().CreateStore(
			c.builder().CreateIntToPtr(
				c.builder().CreateAdd(
					c.builder().CreatePtrToInt(c.builder().CreateLoad(c.ptr, iter_ptr, ""), c.ddpint, ""),
					c.newInt(int64(c.getTypeSize(inListTyp.elementType))),
					"",
				),
				c.ptr,
				"",
			),
			iter_ptr,
		)
	}
	if s.Index != nil {
		c.builder().CreateStore(c.builder().CreateAdd(c.builder().CreateLoad(c.ddpinttyp.LLType(), index, ""), c.newInt(1), ""), index) // index += 1
	}
	c.builder().CreateBr(condBlock)

	c.builder().setBlock(leaveBlock)
	c.builder().scp.unprotectTemporary(in)
	// delete(c.builder().scp.variables, s.Initializer.Name()) // the loopvar was already freed
	c.builder().scp = c.exitScope(c.builder().scp)

	c.builder().setBlock(breakLeave)
	c.freeNonPrimitive(in, inTyp)
	c.freeNonPrimitive(loopVar.val, loopVar.typ)

	trueLeave := c.builder().newBlock()
	c.builder().withBlock(leaveBlock, func() { c.builder().CreateBr(trueLeave) })
	c.builder().withBlock(breakLeave, func() { c.builder().CreateBr(trueLeave) })
	c.builder().setBlock(trueLeave)

	c.builder().curLoopScope, c.builder().curLeaveBlock, c.builder().curContinueBlock = loopScopeBack, leaveBlockBack, continueBlockBack
	return ast.VisitRecurse
}

func (c *compiler) VisitBreakContinueStmt(s *ast.BreakContinueStmt) ast.VisitResult {
	c.exitNestedScopes(c.builder().curLoopScope)
	if s.Tok.Type == token.VERLASSE {
		c.builder().CreateBr(c.builder().curLeaveBlock)
	} else {
		c.builder().CreateBr(c.builder().curContinueBlock)
	}
	c.builder().setBlock(c.builder().newBlock())
	return ast.VisitRecurse
}

func (c *compiler) VisitReturnStmt(s *ast.ReturnStmt) ast.VisitResult {
	exitScopeReturn := func() {
		for scp := c.builder().scp; scp != c.builder().scp; scp = scp.enclosing {
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
		c.builder().CreateRet(llvm.Value{})
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
			c.builder().CreateStore(c.builder().CreateLoad(valTyp.LLType(), val, ""), c.builder().params[0].val)
			c.claimOrCopy(c.builder().params[0].val, val, valTyp, isTemp)
			c.builder().CreateRet(llvm.Value{})
		} else {
			// normal return
			c.builder().CreateRet(val)
		}
	} else {
		// implicit cast to any if required
		if ddptypes.DeepEqual(s.Func.ReturnType, ddptypes.VARIABLE) && valTyp != c.ddpany {
			val, valTyp, isTemp = c.castNonAnyToAny(val, valTyp, isTemp, vtable)
		}

		c.builder().CreateStore(c.builder().CreateLoad(valTyp.LLType(), val, ""), c.builder().params[0].val)
		c.claimOrCopy(c.builder().params[0].val, val, valTyp, isTemp)
		c.builder().CreateRet(llvm.Value{})
	}
	exitScopeReturn()
	return ast.VisitRecurse
}

func (c *compiler) VisitTodoStmt(stmt *ast.TodoStmt) ast.VisitResult {
	line, column := int64(stmt.Token().Range.Start.Line), int64(stmt.Token().Range.Start.Column)
	c.runtime_error(1, c.todo_error_string, c.newInt(line), c.newInt(column))
	return ast.VisitRecurse
}

// exits all scopes until the current function scope
// frees all scp.non_primitives
func (c *compiler) exitNestedScopes(targetScope *scope) {
	for scp := c.builder().scp; scp != targetScope.enclosing; scp = c.exitScope(scp) {
	}
}

func (c *compiler) addTypdefVTable(d *ast.TypeDefDecl, declarationOnly bool) {
	name := c.mangledNameType(d.Type)
	if _, ok := c.typeDefVTables[name]; ok {
		return
	}

	ir_type := c.toIrType(d.Type)

	vtable := llvm.AddGlobal(c.llmod, c.ptr, name+"_vtable")
	vtable.SetLinkage(llvm.ExternalLinkage)
	vtable.SetVisibility(llvm.DefaultVisibility)

	if !declarationOnly {
		vtable.SetInitializer(llvm.ConstStruct([]llvm.Value{
			llvm.ConstInt(c.ddpint, c.getTypeSize(ir_type), false),
			llvm.ConstNull(c.vtable_type.StructElementTypes()[0]),
			llvm.ConstNull(c.vtable_type.StructElementTypes()[1]),
			llvm.ConstNull(c.vtable_type.StructElementTypes()[2]),
		}, false))
	}

	c.typeDefVTables[name] = vtable
}
