package compiler

import (
	"fmt"
	"path/filepath"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/compiler/llvm"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
	"github.com/DDP-Projekt/Kompilierer/src/parser/typechecker"
	"github.com/DDP-Projekt/Kompilierer/src/token"
)

var (
	DDP_COMPILER_DEBUG string = "undefined"
	DEBUG              bool   = DDP_COMPILER_DEBUG == "true"
)

// compiles a mainModule and all it's imports
// every module is written to a io.Writer created
// by calling destCreator with the given module
func compileWithImports(mod *ast.Module, contextCreator func(*ast.Module) (llvmTargetContext, error),
	errHndl ddperror.Handler, optimizationLevel uint,
) (map[*ast.Module]Result, map[string]struct{}, error) {
	compiledMods := map[*ast.Module]Result{}
	dependencies := map[string]struct{}{}
	return compileWithImportsRec(mod, contextCreator, compiledMods, dependencies, true, errHndl, optimizationLevel)
}

func compileWithImportsRec(mod *ast.Module, contextCreator func(*ast.Module) (llvmTargetContext, error),
	compiledMods map[*ast.Module]Result, dependencies map[string]struct{},
	isMainModule bool, errHndl ddperror.Handler, optimizationLevel uint,
) (map[*ast.Module]Result, map[string]struct{}, error) {
	// the ast must be valid (and should have been resolved and typechecked beforehand)
	if mod.Ast.Faulty {
		return compiledMods, dependencies, fmt.Errorf("Fehlerhafter Quellcode im Modul '%s', Kompilierung abgebrochen", mod.GetIncludeFilename())
	}

	// check if the module was already compiled
	if _, alreadyCompiled := compiledMods[mod]; !alreadyCompiled {
		compiledMods[mod] = Result{} // add the module to the set
	} else {
		return compiledMods, dependencies, nil // break the recursion if the module was already compiled
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

	context, err := contextCreator(mod)
	if err != nil {
		return compiledMods, dependencies, fmt.Errorf("Fehler beim erstellen des LLVM Context: %w", err)
	}

	// compile this module
	compiler, err := newCompiler(mod.FileName, mod, context, errHndl, optimizationLevel)
	if err != nil {
		return compiledMods, dependencies, err
	}
	compiledMods[mod] = compiler.compile(isMainModule)

	// recursively compile the other dependencies
	for _, imprt := range mod.Imports {
		for _, imprtMod := range imprt.Modules {
			if r, d, err := compileWithImportsRec(imprtMod, contextCreator, compiledMods, dependencies, false, errHndl, optimizationLevel); err != nil {
				return r, d, err
			}
		}
	}

	return compiledMods, dependencies, nil
}

// small wrapper for a ast.FuncDecl and the corresponding ir function
type funcWrapper struct {
	irFunc        llvm.Value    // the function in the llvm ir
	llFuncBuilder *llBuilder    // nil for inbuilt or runtime functions
	funcDecl      *ast.FuncDecl // the ast.FuncDecl
}

type llTypes struct {
	ptr, ptr_gc, void, i8, i32, i64             llvm.Type
	ddpint, ddpfloat, ddpbyte, ddpbool, ddpchar llvm.Type
	vtable_type                                 llvm.Type
	token                                       llvm.Type
}

func newLLTypes(llctx llvm.Context) llTypes {
	ptr := llctx.PointerType(0)
	ptr_gc := llctx.PointerType(1)
	i8 := llctx.Int8Type()
	i32 := llctx.Int32Type()
	i64 := llctx.Int64Type()
	return llTypes{
		ptr:      ptr,
		ptr_gc:   ptr_gc,
		void:     llctx.VoidType(),
		i8:       i8,
		i32:      i32,
		i64:      i64,
		ddpint:   i64,
		ddpfloat: llctx.DoubleType(),
		ddpbyte:  i8,
		ddpbool:  llctx.Int1Type(),
		ddpchar:  i32,
		vtable_type: llctx.StructType([]llvm.Type{
			i64,
			ptr,
			ptr,
			ptr,
			i64,
		}, false,
		),
		token: llctx.TokenType(),
	}
}

type llConstants struct {
	zero, zero32, zerof, zero8, one, two32, all_ones, all_ones8, False, True, Null llvm.Value
}

func newLLConstants(types llTypes) llConstants {
	return llConstants{
		zero:      llvm.ConstInt(types.i64, 0, false),
		zero32:    llvm.ConstInt(types.i32, 0, false),
		zerof:     llvm.ConstFloat(types.ddpfloat, 0),
		zero8:     llvm.ConstInt(types.i8, 0, false),
		one:       llvm.ConstInt(types.i64, 1, false),
		two32:     llvm.ConstInt(types.i32, 2, false),
		all_ones:  llvm.ConstAllOnes(types.i64),
		all_ones8: llvm.ConstAllOnes(types.i8),
		False:     llvm.ConstInt(types.ddpbool, 0, false),
		True:      llvm.ConstInt(types.ddpbool, 1, false),
		Null:      llvm.ConstNull(types.ptr),
	}
}

type llAttributes struct {
	attr_nounwind               llvm.Attribute
	attr_nonnull                llvm.Attribute
	attr_noalias                llvm.Attribute
	attr_nocallback             llvm.Attribute
	attr_nofree                 llvm.Attribute
	attr_nosync                 llvm.Attribute
	attr_willreturn             llvm.Attribute
	attr_memory_none            llvm.Attribute
	attr_elementtype_ptr_gc_ptr llvm.Attribute
}

func newLLAttributes(llctx llvm.Context, types llTypes) llAttributes {
	return llAttributes{
		attr_nounwind:               llctx.CreateEnumAttribute(llvm.AttributeKindID("nounwind"), 0),
		attr_nonnull:                llctx.CreateEnumAttribute(llvm.AttributeKindID("nonnull"), 0),
		attr_noalias:                llctx.CreateEnumAttribute(llvm.AttributeKindID("noalias"), 0),
		attr_nocallback:             llctx.CreateEnumAttribute(llvm.AttributeKindID("nocallback"), 0),
		attr_nofree:                 llctx.CreateEnumAttribute(llvm.AttributeKindID("nofree"), 0),
		attr_nosync:                 llctx.CreateEnumAttribute(llvm.AttributeKindID("nosync"), 0),
		attr_willreturn:             llctx.CreateEnumAttribute(llvm.AttributeKindID("willreturn"), 0),
		attr_memory_none:            llctx.CreateEnumAttribute(llvm.AttributeKindID("memory(none)"), 0),
		attr_elementtype_ptr_gc_ptr: llctx.CreateTypeAttribute(llvm.AttributeKindID("elementtype"), llvm.FunctionType(types.ptr_gc, []llvm.Type{types.ptr}, false)),
	}
}

// holds state to compile a DDP AST into llvm ir
type compiler struct {
	llvmTargetContext
	llmod             llvm.Module
	ddpModule         *ast.Module      // the module to be compiled
	errorHandler      ddperror.Handler // errors are passed to this function
	optimizationLevel uint             // level of optimization
	result            Result           // result of the compilation

	builderStack    []*llBuilder
	scp             *scope // current scope in the ast (not in the ir)
	fnScope         *scope
	functions       map[string]*funcWrapper                   // all the global functions
	typeMap         map[ddptypes.Type]*ast.Module             // maps ddpTypes to the module they originate from
	structTypes     map[*ddptypes.StructType]*ddpIrStructType // struct names mapped to their IR type
	refTypes        map[ddptypes.ReferenceType]*ddpIrReferenceType
	importedModules map[*ast.Module]struct{} // all the modules that have already been imported
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
	llAttributes
	// all the type definitions of inbuilt types used by the compiler
	voidtyp                                                                                    *ddpIrVoidType
	ddpinttyp, ddpfloattyp, ddpbytetyp, ddpbooltyp, ddpchartyp                                 *ddpIrPrimitiveType
	ddpstring                                                                                  *ddpIrStringType
	ddpany                                                                                     *ddpIrAnyType
	ddpintlist, ddpfloatlist, ddpbytelist, ddpboollist, ddpcharlist, ddpstringlist, ddpanylist *ddpIrListType
	ddpgenericlist                                                                             *ddpIrGenericListType
}

// create a new Compiler to compile the passed AST
func newCompiler(name string, module *ast.Module, ctx llvmTargetContext, errorHandler ddperror.Handler, optimizationLevel uint) (*compiler, error) {
	if errorHandler == nil { // default error handler does nothing
		errorHandler = ddperror.EmptyHandler
	}

	llmod := ctx.newModule(name)

	types := newLLTypes(ctx.llctx)
	constants := newLLConstants(types)
	attributes := newLLAttributes(ctx.llctx, types)

	return &compiler{
		llvmTargetContext: ctx,
		llmod:             llmod,
		ddpModule:         module,
		errorHandler:      errorHandler,
		optimizationLevel: optimizationLevel,
		result: Result{
			Dependencies: make(map[string]struct{}),
			llMod:        llmod,
		},

		functions:       make(map[string]*funcWrapper),
		typeMap:         createTypeMap(module),
		structTypes:     make(map[*ddptypes.StructType]*ddpIrStructType),
		refTypes:        make(map[ddptypes.ReferenceType]*ddpIrReferenceType),
		importedModules: make(map[*ast.Module]struct{}),
		typeDefVTables:  make(map[string]llvm.Value),

		llTypes:      types,
		llConstants:  constants,
		llAttributes: attributes,
	}, nil
}

// compile the AST contained in c
// if w is not nil, the resulting llir is written to w
// otherwise a string representation is returned in result
// if isMainModule is false, no ddp_main function will be generated
func (c *compiler) compile(isMainModule bool) Result {
	defer compiler_panic_wrapper(c)

	// annotate with implicit ref cast metadata
	ast.VisitModuleRec(c.ddpModule, &ImplicitRefCastAnnotator{})

	c.addExternalDependencies()

	c.pushBuilder(&llBuilder{
		c:       c,
		Builder: c.llctx.NewBuilder(),
	})
	c.scp = newScope(nil)

	c.setup()

	if isMainModule {
		c.disposeAndPop()
		c.newBuilder("ddp_ddpmain", llvm.FunctionType(c.ddpint, nil, false), nil, nil, nil, true, false)
		// called from the ddp-c-runtime after initialization
		c.insertFunction(
			"ddp_ddpmain",
			nil,
			c.builder().llFn,
			c.builder(),
		)

		stackMaps := llvm.AddGlobal(c.llmod, c.i8, "__LLVM_StackMaps")
		stackMaps.SetSection(".llvm_stackmaps")
		stackMaps.SetLinkage(llvm.ExternalLinkage)
		stackMaps.SetVisibility(llvm.DefaultVisibility)

		stackMapsExposed := llvm.AddGlobal(c.llmod, c.ptr, "__LLVM_StackMaps_External")
		stackMapsExposed.SetInitializer(stackMaps)
		stackMapsExposed.SetLinkage(llvm.ExternalLinkage)
		stackMapsExposed.SetVisibility(llvm.DefaultVisibility)
		stackMapsExposed.SetGlobalConstant(true)
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
			c.builder().createCall(dispose_fun.irFunc)
		}
		// on success ddpmain returns 0
		c.builder().CreateRet(c.zero)
	}

	c.moduleInitBuilder.CreateRet(llvm.Value{})    // terminate the module_init func
	c.moduleDisposeBuilder.CreateRet(llvm.Value{}) // terminate the module_init func

	c.disposeBuilders()

	return c.result
}

// dumps only the definitions for inbuilt list types to w
func (c *compiler) dumpListDefinitions() llvm.Module {
	defer compiler_panic_wrapper(c)

	c.pushBuilder(&llBuilder{
		c:       c,
		Builder: c.llctx.NewBuilder(),
	})
	c.scp = newScope(nil)

	c.setupErrorStrings()
	// the order of these function calls is important
	// because the primitive types need to be setup
	// before the list types
	// and the void type before everything else
	c.voidtyp = &ddpIrVoidType{}
	c.initRuntimeFunctions()
	c.setupPrimitiveTypes()
	c.ddpstring = c.defineStringType()
	c.ddpany = c.defineAnyType()
	c.setupListTypes(false) // we want definitions

	c.disposeBuilders()

	return c.llmod
}

func (c *compiler) disposeBuilders() {
	for _, wrapper := range c.functions {
		if wrapper.llFuncBuilder != nil {
			wrapper.llFuncBuilder.Dispose()
		}
	}
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
	oldNode := c.builder().currentNode
	c.builder().currentNode = node
	node.Accept(c)
	c.builder().currentNode = oldNode
}

type ddpValue struct {
	irVal       llvm.Value
	typ         ddpIrType
	isImmediate bool
	isStackRef  bool
}

func newNonImmediate(irVal llvm.Value, typ ddpIrType) ddpValue {
	return ddpValue{irVal: irVal, typ: typ, isImmediate: false, isStackRef: false}
}

func newImmediate(irVal llvm.Value, typ ddpIrType) ddpValue {
	return ddpValue{irVal: irVal, typ: typ, isImmediate: true, isStackRef: false}
}

// helper to evaluate an expression and return its ir value and type
// the  bool signals wether the returned value is a temporary value that can be claimed
// or if it is a 'reference' to a variable that must be copied
func (c *compiler) evaluate(expr ast.Expression) ddpValue {
	return c.evaluateNumeric(expr, nil)
}

// helper to evaluate an expression and return its ir value and type
// the  bool signals wether the returned value is a temporary value that can be claimed
// or if it is a 'reference' to a variable that must be copied
func (c *compiler) evaluateNumeric(expr ast.Expression, to ddpIrType) ddpValue {
	c.visitNode(expr)
	latest := c.builder().latestReturn

	if c.isDereferencedImplicitly(expr) {
		if refType, ok := latest.typ.(*ddpIrReferenceType); ok {
			latest.typ = c.toIrType(refType.ddpType.Type)

			if latest.typ.TriviallyCopyable() {
				latest.irVal = c.builder().CreateLoad(latest.typ.LLType(), latest.irVal, "")
			} else {
				// just make sure the latest.irValue is treated as a temporary
				latest.isImmediate = false
			}

			c.builder().latestReturn = latest
		}
		if _, ok := latest.typ.(*ddpIrPrimitiveType); to != nil && ok {
			return newNonImmediate(c.numericCast(latest.irVal, latest.typ, to), to)
		}
		return latest
	} else if c.isPromotedToRefImplicitly(expr) {
		if _, ok := latest.typ.(*ddpIrPrimitiveType); to != nil && ok {
			latest.irVal, latest.typ, latest.isImmediate = c.numericCast(latest.irVal, latest.typ, to), to, false
		}
		ref := c.allocateGCRef(latest.typ.VTable())
		// c.builder().CreateStore(latest.irVal, ref)
		c.claimOrCopy(ref, latest)
		return newNonImmediate(ref, c.getReferenceType(latest.typ))
	}

	if _, ok := latest.typ.(*ddpIrPrimitiveType); to != nil && ok {
		return newNonImmediate(c.numericCast(latest.irVal, latest.typ, to), to)
	}
	return latest
}

// wether expr gets implicitly dereferenced as annotated
func (c *compiler) isDereferencedImplicitly(expr ast.Expression) bool {
	if att, ok := expr.GetMetadataByKind(ImplicitRefCastMetaKind); ok {
		return att.(ImplicitRefCastMeta).FromRef
	}
	return false
}

// wether expr gets implicitly dereferenced as annotated
func (c *compiler) isPromotedToRefImplicitly(expr ast.Expression) bool {
	if att, ok := expr.GetMetadataByKind(ImplicitRefCastMetaKind); ok {
		return !att.(ImplicitRefCastMeta).FromRef
	}
	return false
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
	c.setupPrimitiveTypes()
	c.ddpstring = c.defineStringType()
	c.ddpany = c.defineAnyType()
	c.setupListTypes(true)

	c.setupModuleInitDispose()

	c.setupOperators()
}

// used in setup()
func (c *compiler) setupErrorStrings() {
	createErrorString := func(msg string) llvm.Value {
		str := llvm.ConstString(msg, true)
		error_string := llvm.AddGlobal(c.llmod, str.Type(), "")
		error_string.SetLinkage(llvm.InternalLinkage)
		error_string.SetVisibility(llvm.DefaultVisibility)
		error_string.SetGlobalConstant(true)
		error_string.SetLinkage(llvm.PrivateLinkage)
		error_string.SetUnnamedAddr(true)
		error_string.SetAlignment(1)
		error_string.SetInitializer(str)
		return error_string
	}

	c.out_of_bounds_error_string = createErrorString("Zeile %lld, Spalte %lld: Index außerhalb der Listen Länge (Index war %ld, Listen Länge war %ld)\n")
	c.slice_error_string = createErrorString("Invalide Indexe (Index 1 war %ld, Index 2 war %ld)\n")
	c.todo_error_string = createErrorString("Zeile %lld, Spalte %lld: Dieser Teil des Programms wurde noch nicht implementiert\n")
	c.bad_cast_error_string = createErrorString("Zeile %lld, Spalte %lld: Falsche Typumwandlung")
	c.invalid_utf8_error_string = createErrorString("Zeile %lld, Spalte %lld: Invalider UTF8 Wert im Text")
}

// used in setup()
func (c *compiler) setupPrimitiveTypes() {
	c.ddpinttyp = c.definePrimitiveType(ddptypes.ZAHL, c.ddpint, c.zero, "ddpint")
	c.ddpfloattyp = c.definePrimitiveType(ddptypes.KOMMAZAHL, c.ddpfloat, c.zerof, "ddpfloat")
	c.ddpbytetyp = c.definePrimitiveType(ddptypes.BYTE, c.ddpbyte, c.zero8, "ddpbyte")
	c.ddpbooltyp = c.definePrimitiveType(ddptypes.WAHRHEITSWERT, c.ddpbool, c.False, "ddpbool")
	c.ddpchartyp = c.definePrimitiveType(ddptypes.BUCHSTABE, c.ddpchar, llvm.ConstInt(c.ddpchar, 0, false), "ddpchar")
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
	c.moduleInitBuilder = c.createBuilder(init_name, llvm.FunctionType(c.void, nil, false), nil, nil, nil, true, false)
	c.moduleInitBuilder.llFn.SetVisibility(llvm.DefaultVisibility)
	c.insertFunction(init_name, nil, c.moduleInitBuilder.llFn, c.moduleInitBuilder)

	c.moduleDisposeBuilder = c.createBuilder(dispose_name, llvm.FunctionType(c.void, nil, false), nil, nil, nil, true, false)
	c.moduleInitBuilder.llFn.SetVisibility(llvm.DefaultVisibility)
	c.insertFunction(dispose_name, nil, c.moduleDisposeBuilder.llFn, c.moduleDisposeBuilder)
}

// used in setup()
func (c *compiler) setupOperators() {
	// hoch operator for different type combinations
	c.declareExternalRuntimeFunction("pow", false, false, c.ddpfloat, c.ddpfloat, c.ddpfloat)

	// logarithm
	c.declareExternalRuntimeFunction("log10", false, false, c.ddpfloat, c.ddpfloat)

	// ddpstring to type cast
	c.declareExternalRuntimeFunction("ddp_string_to_int", false, false, c.ddpint, c.ptr)
	c.declareExternalRuntimeFunction("ddp_string_to_float", false, false, c.ddpfloat, c.ptr)
}

// deep copies the value pointed to by src into dest
// and returns dest
func (c *compiler) deepCopyInto(dest, src llvm.Value, typ ddpIrType) llvm.Value {
	c.builder().createCall(typ.DeepCopyFunc(), dest, src)
	return dest
}

// calls the corresponding free function on val
// if typ.IsPrimitive() == false
func (c *compiler) freeNonPrimitive(val llvm.Value, typ ddpIrType) {
	if !typ.TriviallyCopyable() {
		c.builder().createCall(typ.FreeFunc(), val)
	}
}

// claims the given value if possible, copies it otherwise
// dest should be a value that is definetly freed at some point (meaning a variable or list-element etc.)
func (c *compiler) claimOrCopy(dest llvm.Value, val ddpValue) {
	if !val.typ.TriviallyCopyable() {
		if val.isImmediate { // temporaries can be claimed
			val.irVal = c.builder().CreateLoad(val.typ.LLType(), c.scp.claimTemporary(val.irVal), "")
			c.builder().CreateStore(val.irVal, dest)
		} else { // non-temporaries need to be copied
			c.deepCopyInto(dest, val.irVal, val.typ)
		}
	} else { // primitives are trivially copied
		c.builder().CreateStore(val.irVal, dest) // store the value
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
	// don't overwrite a possible return
	if !c.builder().cb.Terminator().IsNil() {
		c.builder().SetInsertPointBefore(c.builder().cb.LastInstruction())
		defer func() {
			c.builder().SetInsertPointAtEnd(c.builder().cb)
		}()
	}

	for _, v := range scp.variables {
		if !v.protected {
			c.freeNonPrimitive(v.val, v.typ)
		}
	}
	c.freeTemporaries(scp, false)
	return scp.enclosing
}

func (c *compiler) exitFuncScope() *scope {
	// don't overwrite a possible return
	if !c.builder().cb.Terminator().IsNil() {
		c.builder().SetInsertPointBefore(c.builder().cb.LastInstruction())
		defer func() {
			c.builder().SetInsertPointAtEnd(c.builder().cb)
		}()
	}

	for _, v := range c.fnScope.variables {
		c.freeNonPrimitive(v.val, v.typ)
	}
	c.freeTemporaries(c.fnScope, true)
	return c.fnScope.enclosing
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
	if c.scp.isGlobalScope() { // global scope
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
		var initVal ddpValue

		initDDPType := typechecker.TypeOfTypecheckedExpression(d.InitVal)
		// implicit numeric casts
		if ddptypes.IsNumericDeref(d.Type) && ddptypes.IsNumericDeref(initDDPType) {
			numericType := Typ
			for ref, ok := numericType.(*ddpIrReferenceType); ok; ref, ok = numericType.(*ddpIrReferenceType) {
				numericType = ref.underlying
			}
			initVal = c.evaluateNumeric(d.InitVal, numericType) // evaluate the initial value
		} else {
			initVal = c.evaluate(d.InitVal) // evaluate the initial value
		}

		// implicit cast to any if required
		_, t, _ := ddptypes.CastReference(d.Type)
		if ddptypes.DeepEqual(t, ddptypes.VARIABLE) && initVal.typ != c.ddpany {
			vtable := initVal.typ.VTable()
			if typeDef, isTypeDef := ddptypes.CastTypeDef(initDDPType); isTypeDef {
				vtable = c.typeDefVTables[c.mangledNameType(typeDef)]
			}

			initVal = c.castNonAnyToAny(initVal, vtable)
		}

		c.claimOrCopy(varLocation, initVal)
	}

	if c.scp.isGlobalScope() { // module_init
		c.pushBuilder(c.moduleInitBuilder)
		current_temporaries_end := len(c.scp.temporaries)
		addInitializer() // initialize the variable in module_init
		if d.IsGlobal && ddptypes.IsReference(ddptypes.TrueUnderlying(d.Type)) {
			c.builder().createCall(ddp_register_gc_root, varLocation)
		}
		// free all temporaries that were created in the initializer
		for _, v := range c.scp.temporaries[current_temporaries_end:] {
			c.freeNonPrimitive(v.val, v.typ)
		}
		c.scp.temporaries = c.scp.temporaries[:current_temporaries_end]
		c.popBuilder()

		c.pushBuilder(c.moduleDisposeBuilder)
		c.freeNonPrimitive(varLocation, Typ) // free the variable in module_dispose
		c.popBuilder()
	}

	// if those are nil, we are at the global scope but there is no ddp_main func
	// meaning this module is being compiled as a non-main module
	if c.builder().isDDPMain() {
		addInitializer()
		if d.IsGlobal && ddptypes.IsReference(ddptypes.TrueUnderlying(d.Type)) {
			c.builder().createCall(ddp_register_gc_root, varLocation)
		}
	}

	c.scp.addVar(d, varLocation, Typ)
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

func (c *compiler) getPossiblyGenericParamType(param *ast.ParameterInfo) (llvm.Type, ddpIrType) {
	t := ddptypes.TrueUnderlying(param.Type)
	if _, isGeneric := ddptypes.CastDeeplyNestedGenerics(t); isGeneric || ddptypes.IsReference(t) {
		return c.ptr, nil
	}

	irType := c.toIrType(t)

	if irType.TriviallyCopyable() {
		return irType.LLType(), irType // convert the type of the parameter
	}
	return c.ptr, irType
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
	paramAttributes := make([][]llvm.Attribute, 0, len(decl.Parameters)+1)

	hasReturnParam := !retType.TriviallyCopyable()
	// non-primitives are returned by passing a pointer to the struct as first parameter
	if hasReturnParam {
		params = append(params, c.ptr)
		paramNames = append(paramNames, "")
		paramAttributes = append(paramAttributes, []llvm.Attribute{c.attr_nonnull, c.attr_noalias}) // the return param is never null and never aliased
		retTypeIr = c.voidtyp.LLType()
	}

	// append all the other parameters
	for _, param := range decl.Parameters {
		paramIrType, irType := c.getPossiblyGenericParamType(&param)

		params = append(params, paramIrType) // add it to the list
		paramNames = append(paramNames, param.Name.Literal)

		var attributes []llvm.Attribute = nil
		if irType != nil {
			switch irType.(type) {
			case *ddpIrAnyType, *ddpIrListType, *ddpIrStringType, *ddpIrStructType, *ddpIrGenericListType:
				attributes = append(attributes, c.attr_nonnull)
			}
		}

		paramAttributes = append(paramAttributes, attributes)
	}

	// createBuilder NOT newBuilder, because defineFuncBody pushes it
	llFuncBuilder := c.createBuilder(c.mangledNameDecl(decl), llvm.FunctionType(retTypeIr, params, false), nil, paramNames, paramAttributes, !ast.IsExternFunc(decl), ast.IsExternFunc(decl))
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

	c.defineFuncBody(fun.llFuncBuilder, !retType.TriviallyCopyable(), def.Func)
	return ast.VisitRecurse
}

// helper function for VisitFuncDef and VisitFuncDecl to compile the  body of a ir function
func (c *compiler) defineFuncBody(llFuncBuilder *llBuilder, hasReturnParam bool, decl *ast.FuncDecl) {
	fnScope := c.fnScope
	c.scp = newScope(c.scp)
	c.fnScope = c.scp
	c.pushBuilder(llFuncBuilder)
	defer func() {
		c.popBuilder()
		c.fnScope = fnScope
	}()

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
		irType := c.toIrType(decl.Parameters[i].Type)
		varDecl, _, _ := body.Symbols.LookupDecl(params[i].name)
		paramDecl := varDecl.(*ast.VarDecl)
		if !irType.TriviallyCopyable() { // strings and lists need special handling
			// add the local variable for the parameter
			v := c.scp.addVar(paramDecl, c.NewAlloca(irType.LLType()), irType)
			c.builder().CreateStore(c.builder().CreateLoad(irType.LLType(), params[i].val, ""), v) // store the copy in the local variable
		} else { // primitive types don't need any special handling
			v := c.scp.addVar(paramDecl, c.NewAlloca(irType.LLType()), irType)
			c.builder().CreateStore(params[i].val, v)
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

	// // don't overwrite a possible return
	// if !c.builder().cb.Terminator().IsNil() {
	// 	c.builder().SetInsertPointBefore(c.builder().cb.LastInstruction())
	// }

	// free the local variables of the function
	// then
	// free the parameters of the function
	if toplevelReturn {
		c.scp = c.scp.enclosing
		c.scp = c.scp.enclosing
	} else {
		c.scp = c.exitScope(c.scp)
		c.scp = c.exitFuncScope()
	}

	if c.builder().cb.Terminator().IsNil() {
		c.builder().CreateRet(llvm.Value{}) // every block needs a terminator, and every function a return
	}
}

func (c *compiler) VisitStructDecl(decl *ast.StructDecl) ast.VisitResult {
	c.defineOrDeclareAllDeclTypes(decl)
	return ast.VisitRecurse
}

func (c *compiler) VisitTypeAliasDecl(decl *ast.TypeAliasDecl) ast.VisitResult {
	return ast.VisitRecurse
}

func (c *compiler) VisitTypeDefDecl(decl *ast.TypeDefDecl) ast.VisitResult {
	c.addTypdefVTable(decl)
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

	if _, isRef := Var.typ.(*ddpIrReferenceType); isRef { // primitives are simply loaded
		c.builder().latestReturn.irVal = c.builder().CreateLoad(Var.typ.LLType(), Var.val, "")
		c.builder().latestReturn.typ = Var.typ
	} else { // non-primitives are used by pointer
		c.builder().latestReturn.irVal = Var.val
		c.builder().latestReturn.typ = c.getReferenceType(Var.typ)
		c.builder().latestReturn.isStackRef = true
	}
	c.builder().latestReturn.isImmediate = false
	return ast.VisitRecurse
}

// literals are simple ir constants
func (c *compiler) VisitIntLit(e *ast.IntLit) ast.VisitResult {
	c.builder().latestReturn.irVal = c.newInt(e.Value)
	c.builder().latestReturn.typ = c.ddpinttyp
	return ast.VisitRecurse
}

func (c *compiler) VisitFloatLit(e *ast.FloatLit) ast.VisitResult {
	c.builder().latestReturn.irVal = llvm.ConstFloat(c.ddpfloat, e.Value)
	c.builder().latestReturn.typ = c.ddpfloattyp
	return ast.VisitRecurse
}

func (c *compiler) VisitBoolLit(e *ast.BoolLit) ast.VisitResult {
	c.builder().latestReturn.irVal = c.newIntT(c.ddpbool, int64(boolToInt(e.Value)))
	c.builder().latestReturn.typ = c.ddpbooltyp
	return ast.VisitRecurse
}

func (c *compiler) VisitCharLit(e *ast.CharLit) ast.VisitResult {
	c.builder().latestReturn.irVal = c.newIntT(c.ddpchar, int64(e.Value))
	c.builder().latestReturn.typ = c.ddpchartyp
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
		c.builder().createCall(c.ddpstring.fromConstantsIrFun, dest, constStr)
	}
	c.builder().latestReturn = c.scp.addTemporary(dest, c.ddpstring) // so that it is freed later
	return ast.VisitRecurse
}

func (c *compiler) VisitListLit(e *ast.ListLit) ast.VisitResult {
	listType := c.toIrType(typechecker.TypeOfTypecheckedExpression(e)).(*ddpIrListType)
	list := c.NewAlloca(listType.LLType())

	// get the listLen as irValue
	listLen := newImmediate(c.zero, c.ddpinttyp)
	if e.Values != nil {
		listLen.irVal = c.newInt(int64(len(e.Values)))
	} else if e.Count != nil && e.Value != nil {
		listLen = c.evaluate(e.Count)
		listLen.irVal, listLen.typ = c.floatOrByteAsInt(listLen.irVal, listLen.typ), c.ddpinttyp
	} else { // empty list
		c.builder().CreateStore(listType.DefaultValue(), list)
		c.builder().latestReturn = c.scp.addTemporary(list, listType)
		return ast.VisitRecurse
	}

	// create a empty list of the correct length
	c.builder().createCall(listType.fromConstantsIrFun, list, listLen.irVal)

	listArr := c.loadStructField(listType.typ, list, list_arr_field_index) // load the array

	if e.Values != nil { // we got some values to copy
		// evaluate every value and copy it into the array
		for i, v := range e.Values {
			val := c.evaluate(v)
			elementPtr := c.indexArray(listType.elementType.LLType(), listArr, c.newInt(int64(i)))
			c.claimOrCopy(elementPtr, val)
		}
	} else if e.Count != nil && e.Value != nil { // single Value multiple times
		val := c.evaluate(e.Value) // if val is a temporary, it is freed automatically

		c.createFor(c.zero, c.forDefaultCond(listLen.irVal), func(index llvm.Value) {
			elementPtr := c.indexArray(listType.elementType.LLType(), listArr, index)
			if listType.elementType.TriviallyCopyable() {
				c.builder().CreateStore(val.irVal, elementPtr)
			} else {
				c.deepCopyInto(elementPtr, val.irVal, listType.elementType)
			}
		})
	}
	c.builder().latestReturn = c.scp.addTemporary(list, listType)
	return ast.VisitRecurse
}

func (c *compiler) VisitUnaryExpr(e *ast.UnaryExpr) ast.VisitResult {
	if e.OverloadedBy != nil {
		return c.VisitFuncCall(e.OverloadedBy.Call)
	}

	rhs := c.evaluate(e.Rhs) // compile the expression onto which the operator is applied

	// big switches for the different type combinations
	switch e.Operator {
	case ast.UN_ABS:
		switch rhs.typ {
		case c.ddpfloattyp:
			// c.builder().latestReturn.irVal = rhs < 0 ? 0 - rhs : rhs;
			c.builder().latestReturn.irVal = c.createTernary(c.ddpfloat, c.builder().CreateFCmp(llvm.FloatOLT, rhs.irVal, c.zerof, ""),
				func() llvm.Value { return c.builder().CreateFSub(c.zerof, rhs.irVal, "") },
				func() llvm.Value { return rhs.irVal },
			)
			c.builder().latestReturn.typ = c.ddpfloattyp
		case c.ddpinttyp:
			// c.builder().latestReturn.irVal = rhs.irVal < 0 ? 0 - rhs.irVal : rhs.irVal;
			c.builder().latestReturn.irVal = c.createTernary(c.ddpint, c.builder().CreateICmp(llvm.IntSLT, rhs.irVal, c.zero, ""),
				func() llvm.Value { return c.builder().CreateSub(c.zero, rhs.irVal, "") },
				func() llvm.Value { return rhs.irVal },
			)
			c.builder().latestReturn.typ = c.ddpinttyp
		case c.ddpbytetyp:
			// a byte is unsigned and therefore does not need to be changed
		default:
			c.err("invalid Parameter Type for BETRAG: %s", rhs.typ.Name())
		}
	case ast.UN_NEGATE:
		switch rhs.typ {
		case c.ddpfloattyp:
			c.builder().latestReturn.irVal = c.builder().CreateFNeg(rhs.irVal, "")
			c.builder().latestReturn.typ = c.ddpfloattyp
		case c.ddpinttyp:
			c.builder().latestReturn.irVal = c.builder().CreateSub(c.zero, rhs.irVal, "")
			c.builder().latestReturn.typ = c.ddpinttyp
		case c.ddpinttyp:
			c.builder().latestReturn.irVal = c.builder().CreateSub(c.zero, c.floatOrByteAsInt(rhs.irVal, c.ddpbytetyp), "")
			c.builder().latestReturn.typ = c.ddpinttyp
		default:
			c.err("invalid Parameter Type for NEGATE: %s", rhs.typ.Name())
		}
	case ast.UN_NOT:
		c.builder().latestReturn.irVal = c.builder().CreateXor(rhs.irVal, c.True, "")
		c.builder().latestReturn.typ = c.ddpbooltyp
	case ast.UN_LOGIC_NOT:
		switch rhs.typ {
		case c.ddpinttyp:
			c.builder().latestReturn.irVal = c.builder().CreateXor(rhs.irVal, c.all_ones, "")
			c.builder().latestReturn.typ = c.ddpinttyp
		case c.ddpbytetyp:
			c.builder().latestReturn.irVal = c.builder().CreateXor(rhs.irVal, c.all_ones8, "")
			c.builder().latestReturn.typ = c.ddpbytetyp
		}
	case ast.UN_LEN:
		switch rhs.typ {
		case c.ddpstring:
			c.builder().latestReturn.irVal = c.builder().createCall(c.ddpstring.lengthIrFun, rhs.irVal)
		default:
			if listTyp, isList := rhs.typ.(*ddpIrListType); isList {
				c.builder().latestReturn.irVal = c.loadStructField(listTyp.typ, rhs.irVal, list_len_field_index)
			} else {
				c.err("invalid Parameter Type for LÄNGE: %s", rhs.typ.Name())
			}
		}
		c.builder().latestReturn.typ = c.ddpinttyp
	default:
		c.err("Unbekannter Operator '%s'", e.Operator)
	}
	return ast.VisitRecurse
}

func (c *compiler) VisitBinaryExpr(e *ast.BinaryExpr) ast.VisitResult {
	if e.OverloadedBy != nil {
		return c.VisitFuncCall(e.OverloadedBy.Call)
	}

	if _, isStringIndexing := e.GetMetadataByKind(ast.StringIndexingMetaKind); isStringIndexing {
		c.evaluate(e.Lhs)
		return ast.VisitRecurse
	}

	// for UND and ODER both operands are booleans, so we don't need to worry about memory management
	// for BIN_FIELD_ACCESS we don't want to evaluate Lhs, as it is just the field name
	switch e.Operator {
	case ast.BIN_AND:
		lhs := c.evaluate(e.Lhs)
		startBlock, trueBlock, leaveBlock := c.builder().cb, c.builder().newBlock(), c.builder().newBlock()
		c.builder().CreateCondBr(lhs.irVal, trueBlock, leaveBlock)

		c.builder().setBlock(trueBlock)
		// collect temporaries because of possible short-circuiting
		c.scp = newScope(c.scp)
		rhs := c.evaluate(e.Rhs)
		// free temporaries
		c.scp = c.exitScope(c.scp)
		c.builder().CreateBr(leaveBlock)
		trueBlock = c.builder().cb

		c.builder().setBlock(leaveBlock)
		phi := c.builder().CreatePHI(c.ddpbool, "")
		phi.AddIncoming([]llvm.Value{rhs.irVal, lhs.irVal}, []llvm.BasicBlock{trueBlock, startBlock})
		c.builder().latestReturn.irVal = phi
		c.builder().latestReturn.typ = c.ddpbooltyp
		return ast.VisitRecurse
	case ast.BIN_OR:
		lhs := c.evaluate(e.Lhs)
		startBlock, falseBlock, leaveBlock := c.builder().cb, c.builder().newBlock(), c.builder().newBlock()
		c.builder().CreateCondBr(lhs.irVal, leaveBlock, falseBlock)

		c.builder().setBlock(falseBlock)
		// collect temporaries because of possible short-circuiting
		c.scp = newScope(c.scp)
		rhs := c.evaluate(e.Rhs)
		// free temporaries
		c.scp = c.exitScope(c.scp)
		c.builder().CreateBr(leaveBlock)
		falseBlock = c.builder().cb // in case c.evaluate has multiple blocks

		c.builder().setBlock(leaveBlock)
		phi := c.builder().CreatePHI(c.ddpbool, "")
		phi.AddIncoming([]llvm.Value{lhs.irVal, rhs.irVal}, []llvm.BasicBlock{startBlock, falseBlock})
		c.builder().latestReturn.irVal = phi
		c.builder().latestReturn.typ = c.ddpbooltyp
		return ast.VisitRecurse
	case ast.BIN_FIELD_ACCESS:
		rhs := c.evaluate(e.Rhs)

		rhsRefTyp, isRefRhs := rhs.typ.(*ddpIrReferenceType)
		if isRefRhs {
			rhs.typ = rhsRefTyp.underlying
		}

		structType, isStruct := rhs.typ.(*ddpIrStructType)
		if !isStruct {
			c.err("invalid Parameter Types for VON (%s)", rhs.typ.Name())
		}

		fieldIndex := getFieldIndex(e.Lhs.Token().Literal, structType)
		fieldType := structType.fieldIrTypes[fieldIndex]
		fieldPtr := c.indexStruct(structType.typ, rhs.irVal, fieldIndex)

		if fieldType.TriviallyCopyable() && !isRefRhs {
			c.builder().latestReturn.irVal = c.builder().CreateLoad(fieldType.LLType(), fieldPtr, "")
		} else if !rhs.isImmediate {
			c.builder().latestReturn = newNonImmediate(fieldPtr, c.getReferenceType(fieldType))
			return ast.VisitRecurse
		} else {
			dest := c.NewAlloca(fieldType.LLType())
			c.builder().CreateStore(c.builder().CreateLoad(fieldType.LLType(), fieldPtr, ""), dest)
			c.builder().CreateStore(fieldType.DefaultValue(), fieldPtr)
			c.builder().latestReturn = c.scp.addTemporary(dest, fieldType)
		}
		c.builder().latestReturn.typ = fieldType
		return ast.VisitRecurse
	}

	// compile the two expressions onto which the operator is applied
	lhs := c.evaluate(e.Lhs)
	rhs := c.evaluate(e.Rhs)
	// big switches on the different type combinations
	switch e.Operator {
	case ast.BIN_XOR:
		c.builder().latestReturn.irVal = c.builder().CreateXor(lhs.irVal, rhs.irVal, "")
		c.builder().latestReturn.typ = c.ddpbooltyp
	case ast.BIN_CONCAT:
		var (
			result    llvm.Value
			resultTyp ddpIrType
			claimsLhs bool
			claimsRhs bool
		)

		lhsListTyp, lhsIsList := lhs.typ.(*ddpIrListType)
		rhsListTyp, rhsIsList := rhs.typ.(*ddpIrListType)

		if lhsIsList {
			resultTyp = lhsListTyp
		} else if rhsIsList {
			resultTyp = rhsListTyp
		} else {
			if lhs.typ == c.ddpstring && !rhsIsList ||
				rhs.typ == c.ddpstring && !lhsIsList {
				resultTyp = c.ddpstring
			} else {
				resultTyp = c.getListType(lhs.typ)
			}
		}
		result = c.NewAlloca(resultTyp.LLType())

		// string concatenations
		var concat_func llvm.Value
		if lhs.typ == c.ddpstring && rhs.typ == c.ddpstring {
			concat_func = c.ddpstring.str_str_concat_IrFunc
			claimsLhs, claimsRhs = true, false
		} else if lhs.typ == c.ddpstring && rhs.typ == c.ddpchartyp {
			concat_func = c.ddpstring.str_char_concat_IrFunc
			claimsLhs, claimsRhs = true, false
		} else if lhs.typ == c.ddpchartyp && rhs.typ == c.ddpstring {
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
				concat_func = c.getListType(lhs.typ).scalar_scalar_concat_IrFunc
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
		if claimsLhs && !lhs.isImmediate {
			dest := c.NewAlloca(lhs.typ.LLType())
			lhs.irVal = c.deepCopyInto(dest, lhs.irVal, lhs.typ)
		}
		if claimsRhs && !rhs.isImmediate {
			dest := c.NewAlloca(rhs.typ.LLType())
			rhs.irVal = c.deepCopyInto(dest, rhs.irVal, rhs.typ)
		}

		c.builder().createCall(concat_func, result, lhs.irVal, rhs.irVal)
		c.builder().latestReturn = c.scp.addTemporary(result, resultTyp)
	case ast.BIN_PLUS:
		switch lhs.typ {
		case c.ddpinttyp:
			switch rhs.typ {
			case c.ddpinttyp:
				c.builder().latestReturn.irVal = c.builder().CreateAdd(lhs.irVal, rhs.irVal, "")
				c.builder().latestReturn.typ = c.ddpinttyp
			case c.ddpfloattyp:
				fp := c.builder().CreateSIToFP(lhs.irVal, c.ddpfloat, "")
				c.builder().latestReturn.irVal = c.builder().CreateFAdd(fp, rhs.irVal, "")
				c.builder().latestReturn.typ = c.ddpfloattyp
			case c.ddpbytetyp:
				c.builder().latestReturn.irVal = c.builder().CreateAdd(lhs.irVal, c.floatOrByteAsInt(rhs.irVal, c.ddpbytetyp), "")
				c.builder().latestReturn.typ = c.ddpinttyp
			default:
				c.err("invalid Parameter Types for PLUS (%s, %s)", lhs.typ.Name(), rhs.typ.Name())
			}
		case c.ddpfloattyp:
			switch rhs.typ {
			case c.ddpinttyp:
				fp := c.builder().CreateSIToFP(rhs.irVal, c.ddpfloat, "")
				c.builder().latestReturn.irVal = c.builder().CreateFAdd(lhs.irVal, fp, "")
			case c.ddpfloattyp:
				c.builder().latestReturn.irVal = c.builder().CreateFAdd(lhs.irVal, rhs.irVal, "")
			case c.ddpbytetyp:
				fp := c.builder().CreateUIToFP(rhs.irVal, c.ddpfloat, "")
				c.builder().latestReturn.irVal = c.builder().CreateFAdd(lhs.irVal, fp, "")
			default:
				c.err("invalid Parameter Types for PLUS (%s, %s)", lhs.typ.Name(), rhs.typ.Name())
			}
			c.builder().latestReturn.typ = c.ddpfloattyp
		case c.ddpbytetyp:
			switch rhs.typ {
			case c.ddpinttyp:
				c.builder().latestReturn.irVal = c.builder().CreateAdd(c.floatOrByteAsInt(lhs.irVal, c.ddpbytetyp), rhs.irVal, "")
				c.builder().latestReturn.typ = c.ddpinttyp
			case c.ddpfloattyp:
				fp := c.builder().CreateUIToFP(lhs.irVal, c.ddpfloat, "")
				c.builder().latestReturn.irVal = c.builder().CreateFAdd(fp, rhs.irVal, "")
			case c.ddpbytetyp:
				c.builder().latestReturn.irVal = c.builder().CreateAdd(lhs.irVal, rhs.irVal, "")
				c.builder().latestReturn.typ = c.ddpbytetyp
			default:
				c.err("invalid Parameter Types for PLUS (%s, %s)", lhs.typ.Name(), rhs.typ.Name())
			}
		default:
			c.err("invalid Parameter Types for PLUS (%s, %s)", lhs.typ.Name(), rhs.typ.Name())
		}
	case ast.BIN_MINUS:
		switch lhs.typ {
		case c.ddpinttyp:
			switch rhs.typ {
			case c.ddpinttyp:
				c.builder().latestReturn.irVal = c.builder().CreateSub(lhs.irVal, rhs.irVal, "")
				c.builder().latestReturn.typ = c.ddpinttyp
			case c.ddpfloattyp:
				fp := c.builder().CreateSIToFP(lhs.irVal, c.ddpfloat, "")
				c.builder().latestReturn.irVal = c.builder().CreateFSub(fp, rhs.irVal, "")
				c.builder().latestReturn.typ = c.ddpfloattyp
			case c.ddpbytetyp:
				c.builder().latestReturn.irVal = c.builder().CreateSub(lhs.irVal, c.floatOrByteAsInt(rhs.irVal, c.ddpbytetyp), "")
				c.builder().latestReturn.typ = c.ddpinttyp
			default:
				c.err("invalid Parameter Types for MINUS (%s, %s)", lhs.typ.Name(), rhs.typ.Name())
			}
		case c.ddpfloattyp:
			switch rhs.typ {
			case c.ddpinttyp:
				fp := c.builder().CreateSIToFP(rhs.irVal, c.ddpfloat, "")
				c.builder().latestReturn.irVal = c.builder().CreateFSub(lhs.irVal, fp, "")
			case c.ddpfloattyp:
				c.builder().latestReturn.irVal = c.builder().CreateFSub(lhs.irVal, rhs.irVal, "")
			case c.ddpbytetyp:
				fp := c.builder().CreateUIToFP(rhs.irVal, c.ddpfloat, "")
				c.builder().latestReturn.irVal = c.builder().CreateFSub(lhs.irVal, fp, "")
			default:
				c.err("invalid Parameter Types for MINUS (%s, %s)", lhs.typ.Name(), rhs.typ.Name())
			}
			c.builder().latestReturn.typ = c.ddpfloattyp
		case c.ddpbytetyp:
			switch rhs.typ {
			case c.ddpinttyp:
				c.builder().latestReturn.irVal = c.builder().CreateSub(c.floatOrByteAsInt(lhs.irVal, c.ddpbytetyp), rhs.irVal, "")
				c.builder().latestReturn.typ = c.ddpinttyp
			case c.ddpfloattyp:
				fp := c.builder().CreateUIToFP(lhs.irVal, c.ddpfloat, "")
				c.builder().latestReturn.irVal = c.builder().CreateFSub(fp, rhs.irVal, "")
				c.builder().latestReturn.typ = c.ddpfloattyp
			case c.ddpbytetyp:
				c.builder().latestReturn.irVal = c.builder().CreateSub(lhs.irVal, rhs.irVal, "")
				c.builder().latestReturn.typ = c.ddpbytetyp
			default:
				c.err("invalid Parameter Types for MINUS (%s, %s)", lhs.typ.Name(), rhs.typ.Name())
			}
		default:
			c.err("invalid Parameter Types for MINUS (%s, %s)", lhs.typ.Name(), rhs.typ.Name())
		}
	case ast.BIN_MULT:
		switch lhs.typ {
		case c.ddpinttyp:
			switch rhs.typ {
			case c.ddpinttyp:
				c.builder().latestReturn.irVal = c.builder().CreateMul(lhs.irVal, rhs.irVal, "")
			case c.ddpfloattyp:
				fp := c.builder().CreateSIToFP(lhs.irVal, c.ddpfloat, "")
				c.builder().latestReturn.irVal = c.builder().CreateFMul(fp, rhs.irVal, "")
				c.builder().latestReturn.typ = c.ddpfloattyp
			case c.ddpbytetyp:
				c.builder().latestReturn.irVal = c.builder().CreateMul(lhs.irVal, c.floatOrByteAsInt(rhs.irVal, c.ddpbytetyp), "")
				c.builder().latestReturn.typ = c.ddpinttyp
			default:
				c.err("invalid Parameter Types for MAL (%s, %s)", lhs.typ.Name(), rhs.typ.Name())
			}
		case c.ddpfloattyp:
			switch rhs.typ {
			case c.ddpinttyp:
				fp := c.builder().CreateSIToFP(rhs.irVal, c.ddpfloat, "")
				c.builder().latestReturn.irVal = c.builder().CreateFMul(lhs.irVal, fp, "")
			case c.ddpfloattyp:
				c.builder().latestReturn.irVal = c.builder().CreateFMul(lhs.irVal, rhs.irVal, "")
			case c.ddpbytetyp:
				fp := c.builder().CreateUIToFP(rhs.irVal, c.ddpfloat, "")
				c.builder().latestReturn.irVal = c.builder().CreateFMul(lhs.irVal, fp, "")
			default:
				c.err("invalid Parameter Types for MAL (%s, %s)", lhs.typ.Name(), rhs.typ.Name())
			}
			c.builder().latestReturn.typ = c.ddpfloattyp
		case c.ddpbytetyp:
			switch rhs.typ {
			case c.ddpinttyp:
				c.builder().latestReturn.irVal = c.builder().CreateMul(c.floatOrByteAsInt(lhs.irVal, c.ddpbytetyp), rhs.irVal, "")
				c.builder().latestReturn.typ = c.ddpinttyp
			case c.ddpfloattyp:
				fp := c.builder().CreateUIToFP(lhs.irVal, c.ddpfloat, "")
				c.builder().latestReturn.irVal = c.builder().CreateFMul(fp, rhs.irVal, "")
				c.builder().latestReturn.typ = c.ddpfloattyp
			case c.ddpbytetyp:
				c.builder().latestReturn.irVal = c.builder().CreateMul(lhs.irVal, rhs.irVal, "")
			default:
				c.err("invalid Parameter Types for MAL (%s, %s)", lhs.typ.Name(), rhs.typ.Name())
			}
		default:
			c.err("invalid Parameter Types for MAL (%s, %s)", lhs.typ.Name(), rhs.typ.Name())
		}
	case ast.BIN_DIV:
		switch lhs.typ {
		case c.ddpinttyp:
			switch rhs.typ {
			case c.ddpinttyp:
				lhs.irVal = c.builder().CreateSIToFP(lhs.irVal, c.ddpfloat, "")
				rhs.irVal = c.builder().CreateSIToFP(rhs.irVal, c.ddpfloat, "")
				c.builder().latestReturn.irVal = c.builder().CreateFDiv(lhs.irVal, rhs.irVal, "")
			case c.ddpfloattyp:
				fp := c.builder().CreateSIToFP(lhs.irVal, c.ddpfloat, "")
				c.builder().latestReturn.irVal = c.builder().CreateFDiv(fp, rhs.irVal, "")
			case c.ddpbytetyp:
				lhs.irVal = c.builder().CreateSIToFP(lhs.irVal, c.ddpfloat, "")
				rhs.irVal = c.builder().CreateUIToFP(rhs.irVal, c.ddpfloat, "")
				c.builder().latestReturn.irVal = c.builder().CreateFDiv(lhs.irVal, rhs.irVal, "")
			default:
				c.err("invalid Parameter Types for DURCH (%s, %s)", lhs.typ.Name(), rhs.typ.Name())
			}
		case c.ddpfloattyp:
			switch rhs.typ {
			case c.ddpinttyp:
				fp := c.builder().CreateSIToFP(rhs.irVal, c.ddpfloat, "")
				c.builder().latestReturn.irVal = c.builder().CreateFDiv(lhs.irVal, fp, "")
			case c.ddpfloattyp:
				c.builder().latestReturn.irVal = c.builder().CreateFDiv(lhs.irVal, rhs.irVal, "")
			case c.ddpbytetyp:
				fp := c.builder().CreateUIToFP(rhs.irVal, c.ddpfloat, "")
				c.builder().latestReturn.irVal = c.builder().CreateFDiv(lhs.irVal, fp, "")
			default:
				c.err("invalid Parameter Types for DURCH (%s, %s)", lhs.typ.Name(), rhs.typ.Name())
			}
		case c.ddpbytetyp:
			switch rhs.typ {
			case c.ddpinttyp:
				lhs.irVal = c.builder().CreateUIToFP(lhs.irVal, c.ddpfloat, "")
				rhs.irVal = c.builder().CreateSIToFP(rhs.irVal, c.ddpfloat, "")
				c.builder().latestReturn.irVal = c.builder().CreateFDiv(lhs.irVal, rhs.irVal, "")
			case c.ddpfloattyp:
				fp := c.builder().CreateSIToFP(lhs.irVal, c.ddpfloat, "")
				c.builder().latestReturn.irVal = c.builder().CreateFDiv(fp, rhs.irVal, "")
			case c.ddpbytetyp:
				lhs.irVal = c.builder().CreateUIToFP(lhs.irVal, c.ddpfloat, "")
				rhs.irVal = c.builder().CreateUIToFP(rhs.irVal, c.ddpfloat, "")
				c.builder().latestReturn.irVal = c.builder().CreateFDiv(lhs.irVal, rhs.irVal, "")
			default:
				c.err("invalid Parameter Types for DURCH (%s, %s)", lhs.typ.Name(), rhs.typ.Name())
			}
		default:
			c.err("invalid Parameter Types for DURCH (%s, %s)", lhs.typ.Name(), rhs.typ.Name())
		}
		c.builder().latestReturn.typ = c.ddpfloattyp
	case ast.BIN_INDEX:
		lhsRefType, isRefLhs := lhs.typ.(*ddpIrReferenceType)
		if isRefLhs {
			lhs.typ = lhsRefType.underlying
		}

		if lhs.typ == c.ddpstring {
			c.builder().latestReturn.irVal = c.builder().createCall(c.ddpstring.indexIrFun, lhs.irVal, c.floatOrByteAsInt(rhs.irVal, rhs.typ))
			c.builder().latestReturn.typ = c.ddpchartyp
			break
		}

		listType, isList := lhs.typ.(*ddpIrListType)
		if !isList {
			c.err("invalid Parameter Types for STELLE (%s, %s)", lhs.typ.Name(), rhs.typ.Name())
		}

		listLen := c.loadStructField(listType.typ, lhs.irVal, list_len_field_index)
		index := c.builder().CreateSub(c.floatOrByteAsInt(rhs.irVal, rhs.typ), c.newInt(1), "") // ddp indices start at 1, so subtract 1
		// index bounds check
		cond := c.builder().CreateAnd(c.builder().CreateICmp(llvm.IntSLT, index, listLen, ""), c.builder().CreateICmp(llvm.IntSGE, index, c.zero, ""), "")
		c.createIfElse(cond, func() {
			listArr := c.loadStructField(listType.typ, lhs.irVal, list_arr_field_index)
			elementPtr := c.indexArray(listType.elementType.LLType(), listArr, index)

			if listType.elementType.TriviallyCopyable() && !isRefLhs {
				c.builder().latestReturn.irVal, c.builder().latestReturn.typ = c.builder().CreateLoad(listType.elementType.LLType(), elementPtr, ""), listType.elementType
			} else if !lhs.isImmediate {
				c.builder().latestReturn = newNonImmediate(elementPtr, c.getReferenceType(listType.elementType))
				return
			} else {
				dest := c.NewAlloca(listType.elementType.LLType())
				c.builder().latestReturn = c.scp.addTemporary(
					c.deepCopyInto(dest, elementPtr, listType.elementType),
					listType.elementType,
				)
			}
		}, func() { // runtime error
			line, column := int64(e.Token().Range.Start.Line), int64(e.Token().Range.Start.Column)
			c.out_of_bounds_error(c.newInt(line), c.newInt(column), rhs.irVal, listLen)
		})
	case ast.BIN_SLICE_FROM, ast.BIN_SLICE_TO:
		dest := c.NewAlloca(lhs.typ.LLType())
		rhs.irVal = c.floatOrByteAsInt(rhs.irVal, rhs.typ)

		switch lhs.typ {
		case c.ddpstring:
			if e.Operator == ast.BIN_SLICE_FROM {
				str_len := c.builder().createCall(c.ddpstring.lengthIrFun, lhs.irVal)
				c.builder().createCall(c.ddpstring.sliceIrFun, dest, lhs.irVal, rhs.irVal, str_len)
			} else {
				c.builder().createCall(c.ddpstring.sliceIrFun, dest, lhs.irVal, c.newInt(1), rhs.irVal)
			}
		default:
			if listTyp, isList := lhs.typ.(*ddpIrListType); isList {
				if e.Operator == ast.BIN_SLICE_FROM {
					lst_len := c.loadStructField(listTyp.typ, lhs.irVal, list_len_field_index)
					c.builder().createCall(listTyp.sliceIrFun, dest, lhs.irVal, rhs.irVal, lst_len)
				} else {
					c.builder().createCall(listTyp.sliceIrFun, dest, lhs.irVal, c.newInt(1), rhs.irVal)
				}
			} else {
				c.err("invalid Parameter Types for %s (%s, %s)", e.Operator.String(), lhs.typ.Name(), rhs.typ.Name())
			}
		}
		c.builder().latestReturn = c.scp.addTemporary(dest, lhs.typ)
	case ast.BIN_POW:
		switch lhs.typ {
		case c.ddpinttyp:
			lhs.irVal = c.builder().CreateSIToFP(lhs.irVal, c.ddpfloat, "")
		case c.ddpbytetyp:
			lhs.irVal = c.builder().CreateUIToFP(lhs.irVal, c.ddpfloat, "")
		case c.ddpfloattyp:
		default:
			c.err("invalid Parameter Types for HOCH (Lhs: %s)", lhs.typ.Name())
		}
		switch rhs.typ {
		case c.ddpinttyp:
			rhs.irVal = c.builder().CreateSIToFP(rhs.irVal, c.ddpfloat, "")
		case c.ddpbytetyp:
			rhs.irVal = c.builder().CreateUIToFP(rhs.irVal, c.ddpfloat, "")
		case c.ddpfloattyp:
		default:
			c.err("invalid Parameter Types for HOCH (Rhs: %s)", lhs.typ.Name())
		}
		irFunc := c.functions["pow"].irFunc
		c.builder().latestReturn.irVal = c.builder().createCall(irFunc, lhs.irVal, rhs.irVal)
		c.builder().latestReturn.typ = c.ddpfloattyp
	case ast.BIN_LOG:
		switch lhs.typ {
		case c.ddpinttyp:
			lhs.irVal = c.builder().CreateSIToFP(lhs.irVal, c.ddpfloat, "")
		case c.ddpbytetyp:
			lhs.irVal = c.builder().CreateUIToFP(lhs.irVal, c.ddpfloat, "")
		case c.ddpfloattyp:
		default:
			c.err("invalid Parameter Types for Logarithmus (Lhs: %s)", lhs.typ.Name())
		}
		switch rhs.typ {
		case c.ddpinttyp:
			rhs.irVal = c.builder().CreateSIToFP(rhs.irVal, c.ddpfloat, "")
		case c.ddpbytetyp:
			rhs.irVal = c.builder().CreateUIToFP(rhs.irVal, c.ddpfloat, "")
		case c.ddpfloattyp:
		default:
			c.err("invalid Parameter Types for Logarithmus (Rhs: %s)", lhs.typ.Name())
		}
		irFunc := c.functions["log10"].irFunc
		log10_num := c.builder().createCall(irFunc, lhs.irVal)
		log10_base := c.builder().createCall(irFunc, rhs.irVal)
		c.builder().latestReturn.irVal = c.builder().CreateFDiv(log10_num, log10_base, "")
		c.builder().latestReturn.typ = c.ddpfloattyp
	case ast.BIN_LOGIC_AND:
		if lhs.typ == c.ddpinttyp || rhs.typ == c.ddpinttyp {
			lhs.irVal, rhs.irVal = c.floatOrByteAsInt(lhs.irVal, lhs.typ), c.floatOrByteAsInt(rhs.irVal, rhs.typ)
			c.builder().latestReturn.typ = c.ddpinttyp
		} else {
			c.builder().latestReturn.typ = c.ddpbytetyp
		}
		c.builder().latestReturn.irVal = c.builder().CreateAnd(lhs.irVal, rhs.irVal, "")
	case ast.BIN_LOGIC_OR:
		if lhs.typ == c.ddpinttyp || rhs.typ == c.ddpinttyp {
			lhs.irVal, rhs.irVal = c.floatOrByteAsInt(lhs.irVal, lhs.typ), c.floatOrByteAsInt(rhs.irVal, rhs.typ)
			c.builder().latestReturn.typ = c.ddpinttyp
		} else {
			c.builder().latestReturn.typ = c.ddpbytetyp
		}
		c.builder().latestReturn.irVal = c.builder().CreateOr(lhs.irVal, rhs.irVal, "")
	case ast.BIN_LOGIC_XOR:
		if lhs.typ == c.ddpinttyp || rhs.typ == c.ddpinttyp {
			lhs.irVal, rhs.irVal = c.floatOrByteAsInt(lhs.irVal, lhs.typ), c.floatOrByteAsInt(rhs.irVal, rhs.typ)
			c.builder().latestReturn.typ = c.ddpinttyp
		} else {
			c.builder().latestReturn.typ = c.ddpbytetyp
		}
		c.builder().latestReturn.irVal = c.builder().CreateXor(lhs.irVal, rhs.irVal, "")
	case ast.BIN_MOD:
		if lhs.typ == c.ddpbytetyp && rhs.typ == c.ddpbytetyp {
			c.builder().latestReturn.irVal = c.builder().CreateURem(lhs.irVal, rhs.irVal, "")
			c.builder().latestReturn.typ = c.ddpbytetyp
		} else {
			c.builder().latestReturn.irVal = c.builder().CreateSRem(c.floatOrByteAsInt(lhs.irVal, lhs.typ), c.floatOrByteAsInt(rhs.irVal, rhs.typ), "")
			c.builder().latestReturn.typ = c.ddpinttyp
		}
	case ast.BIN_LEFT_SHIFT:
		if lhs.typ == c.ddpinttyp || rhs.typ == c.ddpinttyp {
			lhs.irVal, rhs.irVal = c.floatOrByteAsInt(lhs.irVal, lhs.typ), c.floatOrByteAsInt(rhs.irVal, rhs.typ)
			c.builder().latestReturn.typ = c.ddpinttyp
		} else {
			c.builder().latestReturn.typ = c.ddpbytetyp
		}
		c.builder().latestReturn.irVal = c.builder().CreateShl(lhs.irVal, rhs.irVal, "")
	case ast.BIN_RIGHT_SHIFT:
		if lhs.typ == c.ddpinttyp || rhs.typ == c.ddpinttyp {
			lhs.irVal, rhs.irVal = c.floatOrByteAsInt(lhs.irVal, lhs.typ), c.floatOrByteAsInt(rhs.irVal, rhs.typ)
			c.builder().latestReturn.typ = c.ddpinttyp
		} else {
			c.builder().latestReturn.typ = c.ddpbytetyp
		}
		c.builder().latestReturn.irVal = c.builder().CreateLShr(lhs.irVal, rhs.irVal, "")
	case ast.BIN_EQUAL:
		c.compare_values(lhs.irVal, rhs.irVal, lhs.typ)
	case ast.BIN_UNEQUAL:
		equal := c.compare_values(lhs.irVal, rhs.irVal, lhs.typ)
		c.builder().latestReturn.irVal = c.builder().CreateXor(equal, c.True, "")
	case ast.BIN_LESS:
		switch lhs.typ {
		case c.ddpinttyp:
			switch rhs.typ {
			case c.ddpinttyp, c.ddpbytetyp:
				c.builder().latestReturn.irVal = c.builder().CreateICmp(llvm.IntSLT, lhs.irVal, c.floatOrByteAsInt(rhs.irVal, rhs.typ), "")
			case c.ddpfloattyp:
				fp := c.builder().CreateSIToFP(lhs.irVal, c.ddpfloat, "")
				c.builder().latestReturn.irVal = c.builder().CreateFCmp(llvm.FloatOLT, fp, rhs.irVal, "")
			default:
				c.err("invalid Parameter Types for KLEINER (%s, %s)", lhs.typ.Name(), rhs.typ.Name())
			}
		case c.ddpfloattyp:
			switch rhs.typ {
			case c.ddpinttyp:
				fp := c.builder().CreateSIToFP(rhs.irVal, c.ddpfloat, "")
				c.builder().latestReturn.irVal = c.builder().CreateFCmp(llvm.FloatOLT, lhs.irVal, fp, "")
			case c.ddpfloattyp:
				c.builder().latestReturn.irVal = c.builder().CreateFCmp(llvm.FloatOLT, lhs.irVal, rhs.irVal, "")
			case c.ddpbytetyp:
				fp := c.builder().CreateUIToFP(rhs.irVal, c.ddpfloat, "")
				c.builder().latestReturn.irVal = c.builder().CreateFCmp(llvm.FloatOLT, lhs.irVal, fp, "")
			default:
				c.err("invalid Parameter Types for KLEINER (%s, %s)", lhs.typ.Name(), rhs.typ.Name())
			}
		case c.ddpbytetyp:
			switch rhs.typ {
			case c.ddpinttyp:
				c.builder().latestReturn.irVal = c.builder().CreateICmp(llvm.IntSLT, c.floatOrByteAsInt(lhs.irVal, lhs.typ), rhs.irVal, "")
			case c.ddpfloattyp:
				fp := c.builder().CreateUIToFP(lhs.irVal, c.ddpfloat, "")
				c.builder().latestReturn.irVal = c.builder().CreateFCmp(llvm.FloatOLT, fp, rhs.irVal, "")
			case c.ddpbytetyp:
				c.builder().latestReturn.irVal = c.builder().CreateICmp(llvm.IntULT, lhs.irVal, rhs.irVal, "")
			default:
				c.err("invalid Parameter Types for KLEINER (%s, %s)", lhs.typ.Name(), rhs.typ.Name())
			}
		}
		c.builder().latestReturn.typ = c.ddpbooltyp
	case ast.BIN_LESS_EQ:
		switch lhs.typ {
		case c.ddpinttyp:
			switch rhs.typ {
			case c.ddpinttyp, c.ddpbytetyp:
				c.builder().latestReturn.irVal = c.builder().CreateICmp(llvm.IntSLE, lhs.irVal, c.floatOrByteAsInt(rhs.irVal, rhs.typ), "")
			case c.ddpfloattyp:
				fp := c.builder().CreateSIToFP(lhs.irVal, c.ddpfloat, "")
				c.builder().latestReturn.irVal = c.builder().CreateFCmp(llvm.FloatOLE, fp, rhs.irVal, "")
			default:
				c.err("invalid Parameter Types for KLEINER_ALS_ODER (%s, %s)", lhs.typ.Name(), rhs.typ.Name())
			}
		case c.ddpfloattyp:
			switch rhs.typ {
			case c.ddpinttyp:
				fp := c.builder().CreateSIToFP(rhs.irVal, c.ddpfloat, "")
				c.builder().latestReturn.irVal = c.builder().CreateFCmp(llvm.FloatOLE, lhs.irVal, fp, "")
			case c.ddpfloattyp:
				c.builder().latestReturn.irVal = c.builder().CreateFCmp(llvm.FloatOLE, lhs.irVal, rhs.irVal, "")
			case c.ddpbytetyp:
				fp := c.builder().CreateUIToFP(rhs.irVal, c.ddpfloat, "")
				c.builder().latestReturn.irVal = c.builder().CreateFCmp(llvm.FloatOLE, lhs.irVal, fp, "")
			default:
				c.err("invalid Parameter Types for KLEINER_ALS_ODER (%s, %s)", lhs.typ.Name(), rhs.typ.Name())
			}
		case c.ddpbytetyp:
			switch rhs.typ {
			case c.ddpinttyp:
				c.builder().latestReturn.irVal = c.builder().CreateICmp(llvm.IntSLE, c.floatOrByteAsInt(lhs.irVal, lhs.typ), rhs.irVal, "")
			case c.ddpfloattyp:
				fp := c.builder().CreateUIToFP(lhs.irVal, c.ddpfloat, "")
				c.builder().latestReturn.irVal = c.builder().CreateFCmp(llvm.FloatOLE, fp, rhs.irVal, "")
			case c.ddpbytetyp:
				c.builder().latestReturn.irVal = c.builder().CreateICmp(llvm.IntULE, lhs.irVal, rhs.irVal, "")
			default:
				c.err("invalid Parameter Types for KLEINER_ALS_ODER (%s, %s)", lhs.typ.Name(), rhs.typ.Name())
			}
		}
		c.builder().latestReturn.typ = c.ddpbooltyp
	case ast.BIN_GREATER:
		switch lhs.typ {
		case c.ddpinttyp:
			switch rhs.typ {
			case c.ddpinttyp, c.ddpbytetyp:
				c.builder().latestReturn.irVal = c.builder().CreateICmp(llvm.IntSGT, lhs.irVal, c.floatOrByteAsInt(rhs.irVal, rhs.typ), "")
			case c.ddpfloattyp:
				fp := c.builder().CreateSIToFP(lhs.irVal, c.ddpfloat, "")
				c.builder().latestReturn.irVal = c.builder().CreateFCmp(llvm.FloatOGT, fp, rhs.irVal, "")
			default:
				c.err("invalid Parameter Types for GRÖßER (%s, %s)", lhs.typ.Name(), rhs.typ.Name())
			}
		case c.ddpfloattyp:
			switch rhs.typ {
			case c.ddpinttyp:
				fp := c.builder().CreateSIToFP(rhs.irVal, c.ddpfloat, "")
				c.builder().latestReturn.irVal = c.builder().CreateFCmp(llvm.FloatOGT, lhs.irVal, fp, "")
			case c.ddpfloattyp:
				c.builder().latestReturn.irVal = c.builder().CreateFCmp(llvm.FloatOGT, lhs.irVal, rhs.irVal, "")
			case c.ddpbytetyp:
				fp := c.builder().CreateUIToFP(rhs.irVal, c.ddpfloat, "")
				c.builder().latestReturn.irVal = c.builder().CreateFCmp(llvm.FloatOGT, lhs.irVal, fp, "")
			default:
				c.err("invalid Parameter Types for GRÖßER (%s, %s)", lhs.typ.Name(), rhs.typ.Name())
			}
		case c.ddpbytetyp:
			switch rhs.typ {
			case c.ddpinttyp:
				c.builder().latestReturn.irVal = c.builder().CreateICmp(llvm.IntSGT, c.floatOrByteAsInt(lhs.irVal, lhs.typ), rhs.irVal, "")
			case c.ddpfloattyp:
				fp := c.builder().CreateUIToFP(lhs.irVal, c.ddpfloat, "")
				c.builder().latestReturn.irVal = c.builder().CreateFCmp(llvm.FloatOGT, fp, rhs.irVal, "")
			case c.ddpbytetyp:
				c.builder().latestReturn.irVal = c.builder().CreateICmp(llvm.IntUGT, lhs.irVal, rhs.irVal, "")
			default:
				c.err("invalid Parameter Types for GRÖßER (%s, %s)", lhs.typ.Name(), rhs.typ.Name())
			}
		}
		c.builder().latestReturn.typ = c.ddpbooltyp
	case ast.BIN_GREATER_EQ:
		switch lhs.typ {
		case c.ddpinttyp:
			switch rhs.typ {
			case c.ddpinttyp, c.ddpbytetyp:
				c.builder().latestReturn.irVal = c.builder().CreateICmp(llvm.IntSGE, lhs.irVal, c.floatOrByteAsInt(rhs.irVal, rhs.typ), "")
			case c.ddpfloattyp:
				fp := c.builder().CreateSIToFP(lhs.irVal, c.ddpfloat, "")
				c.builder().latestReturn.irVal = c.builder().CreateFCmp(llvm.FloatOGE, fp, rhs.irVal, "")
			default:
				c.err("invalid Parameter Types for GRÖßER_ALS_ODER (%s, %s)", lhs.typ.Name(), rhs.typ.Name())
			}
		case c.ddpfloattyp:
			switch rhs.typ {
			case c.ddpinttyp:
				fp := c.builder().CreateSIToFP(rhs.irVal, c.ddpfloat, "")
				c.builder().latestReturn.irVal = c.builder().CreateFCmp(llvm.FloatOGE, lhs.irVal, fp, "")
			case c.ddpfloattyp:
				c.builder().latestReturn.irVal = c.builder().CreateFCmp(llvm.FloatOGE, lhs.irVal, rhs.irVal, "")
			case c.ddpbytetyp:
				fp := c.builder().CreateUIToFP(rhs.irVal, c.ddpfloat, "")
				c.builder().latestReturn.irVal = c.builder().CreateFCmp(llvm.FloatOGE, lhs.irVal, fp, "")
			default:
				c.err("invalid Parameter Types for GRÖßER_ALS_ODER (%s, %s)", lhs.typ.Name(), rhs.typ.Name())
			}
		case c.ddpbytetyp:
			switch rhs.typ {
			case c.ddpinttyp:
				c.builder().latestReturn.irVal = c.builder().CreateICmp(llvm.IntSGE, c.floatOrByteAsInt(lhs.irVal, lhs.typ), rhs.irVal, "")
			case c.ddpfloattyp:
				fp := c.builder().CreateUIToFP(lhs.irVal, c.ddpfloat, "")
				c.builder().latestReturn.irVal = c.builder().CreateFCmp(llvm.FloatOGE, fp, rhs.irVal, "")
			case c.ddpbytetyp:
				c.builder().latestReturn.irVal = c.builder().CreateICmp(llvm.IntUGE, lhs.irVal, rhs.irVal, "")
			default:
				c.err("invalid Parameter Types for GRÖßER_ALS_ODER (%s, %s)", lhs.typ.Name(), rhs.typ.Name())
			}
		}
		c.builder().latestReturn.typ = c.ddpbooltyp
	}
	return ast.VisitRecurse
}

func (c *compiler) VisitTernaryExpr(e *ast.TernaryExpr) ast.VisitResult {
	if e.OverloadedBy != nil {
		return c.VisitFuncCall(e.OverloadedBy.Call)
	}

	// if due to short circuiting
	if e.Operator == ast.TER_FALLS {
		mid := c.evaluate(e.Mid)
		trueBlock, falseBlock, leaveBlock := c.builder().newBlock(), c.builder().newBlock(), c.builder().newBlock()
		c.builder().CreateCondBr(mid.irVal, trueBlock, falseBlock)

		c.builder().setBlock(trueBlock)
		// collect temporaries because of possible short-circuiting
		c.scp = newScope(c.scp)
		lhs := c.evaluate(e.Lhs)
		// claim the temporary, as the phi instruction will become the actual temporary
		if lhs.isImmediate && !lhs.typ.TriviallyCopyable() {
			lhs.irVal = c.scp.claimTemporary(lhs.irVal)
		}
		// free temporaries
		c.scp = c.exitScope(c.scp)
		trueBlock = c.builder().cb

		c.builder().setBlock(falseBlock)
		// collect temporaries because of possible short-circuiting
		c.scp = newScope(c.scp)
		rhs := c.evaluate(e.Rhs)
		// claim the temporary, as the phi instruction will become the actual temporary
		if rhs.isImmediate && !rhs.typ.TriviallyCopyable() {
			rhs.irVal = c.scp.claimTemporary(rhs.irVal)
		}
		// free temporaries
		c.scp = c.exitScope(c.scp)
		falseBlock = c.builder().cb

		// simple case, where both can be treated the same way
		if lhs.isImmediate == rhs.isImmediate {
			c.builder().latestReturn.isImmediate = lhs.isImmediate
		} else {
			c.builder().latestReturn.isImmediate = true

			// we need to copy the non-temp value to be sure
			c.builder().setBlock(trueBlock)
			if lhs.isImmediate {
				c.builder().setBlock(falseBlock)
			}

			// turn the non-temp into a temporary and claim the temporary,
			// as the phi instruction will become the actual temporary
			dest := c.NewAlloca(lhs.typ.LLType())
			if lhs.isImmediate {
				rhs.irVal = c.deepCopyInto(dest, rhs.irVal, lhs.typ)
			} else {
				lhs.irVal = c.deepCopyInto(dest, lhs.irVal, lhs.typ)
			}
		}

		c.builder().withBlock(falseBlock, func() { c.builder().CreateBr(leaveBlock) })
		c.builder().withBlock(trueBlock, func() { c.builder().CreateBr(leaveBlock) })

		c.builder().setBlock(leaveBlock)
		phiType := rhs.typ.LLType()
		if !rhs.typ.TriviallyCopyable() {
			phiType = c.ptr
		}
		phi := c.builder().CreatePHI(phiType, "")
		phi.AddIncoming([]llvm.Value{lhs.irVal, rhs.irVal}, []llvm.BasicBlock{trueBlock, falseBlock})
		c.builder().latestReturn.irVal = phi
		if c.builder().latestReturn.isImmediate {
			c.scp.addTemporary(c.builder().latestReturn.irVal, lhs.typ)
		}
		c.builder().latestReturn.typ = lhs.typ
		return ast.VisitRecurse
	}

	lhs := c.evaluate(e.Lhs)
	mid := c.evaluate(e.Mid)
	rhs := c.evaluate(e.Rhs)

	switch e.Operator {
	case ast.TER_SLICE:
		dest := c.NewAlloca(lhs.typ.LLType())
		mid.irVal = c.floatOrByteAsInt(mid.irVal, mid.typ)
		rhs.irVal = c.floatOrByteAsInt(rhs.irVal, rhs.typ)
		switch lhs.typ {
		case c.ddpstring:
			c.builder().createCall(c.ddpstring.sliceIrFun, dest, lhs.irVal, mid.irVal, rhs.irVal)
		default:
			if listTyp, isList := lhs.typ.(*ddpIrListType); isList {
				c.builder().createCall(listTyp.sliceIrFun, dest, lhs.irVal, mid.irVal, rhs.irVal)
			} else {
				c.err("invalid Parameter Types for VONBIS (%s, %s, %s)", lhs.typ.Name(), mid.typ.Name(), rhs.typ.Name())
			}
		}
		c.builder().latestReturn = c.scp.addTemporary(dest, lhs.typ)
	case ast.TER_BETWEEN:
		// lhs zwischen mid und rhs
		// = (lhs > rhs && lhs < mid) || (lhs > mid && lhs < rhs)
		if lhs.typ == c.ddpfloattyp || rhs.typ == c.ddpfloattyp || mid.typ == c.ddpfloattyp {
			lhs.irVal, mid.irVal, rhs.irVal = c.intOrByteAsFloat(lhs.irVal, lhs.typ), c.intOrByteAsFloat(mid.irVal, mid.typ), c.intOrByteAsFloat(rhs.irVal, rhs.typ)
			c.builder().latestReturn.irVal = c.builder().CreateOr(
				c.builder().CreateAnd(c.builder().CreateFCmp(llvm.FloatOGT, lhs.irVal, rhs.irVal, ""), c.builder().CreateFCmp(llvm.FloatOLT, lhs.irVal, mid.irVal, ""), ""),
				c.builder().CreateAnd(c.builder().CreateFCmp(llvm.FloatOGT, lhs.irVal, mid.irVal, ""), c.builder().CreateFCmp(llvm.FloatOLT, lhs.irVal, rhs.irVal, ""), ""),
				"",
			)
		} else if lhs.typ == c.ddpbytetyp && rhs.typ == c.ddpbytetyp && mid.typ == c.ddpbytetyp {
			c.builder().latestReturn.irVal = c.builder().CreateOr(
				c.builder().CreateAnd(c.builder().CreateICmp(llvm.IntUGT, lhs.irVal, rhs.irVal, ""), c.builder().CreateICmp(llvm.IntULT, lhs.irVal, mid.irVal, ""), ""),
				c.builder().CreateAnd(c.builder().CreateICmp(llvm.IntUGT, lhs.irVal, mid.irVal, ""), c.builder().CreateICmp(llvm.IntULT, lhs.irVal, rhs.irVal, ""), ""),
				"",
			)
		} else {
			lhs.irVal, mid.irVal, rhs.irVal = c.floatOrByteAsInt(lhs.irVal, lhs.typ), c.floatOrByteAsInt(mid.irVal, mid.typ), c.floatOrByteAsInt(rhs.irVal, rhs.typ)
			c.builder().latestReturn.irVal = c.builder().CreateOr(
				c.builder().CreateAnd(c.builder().CreateICmp(llvm.IntSGT, lhs.irVal, rhs.irVal, ""), c.builder().CreateICmp(llvm.IntSLT, lhs.irVal, mid.irVal, ""), ""),
				c.builder().CreateAnd(c.builder().CreateICmp(llvm.IntSGT, lhs.irVal, mid.irVal, ""), c.builder().CreateICmp(llvm.IntSLT, lhs.irVal, rhs.irVal, ""), ""),
				"",
			)
		}

		c.builder().latestReturn.typ = c.ddpbooltyp
	default:
		c.err("invalid Parameter Types for VONBIS (%s, %s, %s)", lhs.typ.Name(), mid.typ.Name(), rhs.typ.Name())
	}
	return ast.VisitRecurse
}

func (c *compiler) VisitCastExpr(e *ast.CastExpr) ast.VisitResult {
	if e.OverloadedBy != nil {
		return c.VisitFuncCall(e.OverloadedBy.Call)
	}

	targetType := ddptypes.TrueUnderlying(e.TargetType)
	targetIrType := c.toIrType(targetType)
	lhs := c.evaluate(e.Lhs)

	vtable := targetIrType.VTable()
	if typeDef, isTypeDef := ddptypes.CastTypeDef(e.TargetType); isTypeDef {
		vtable = c.typeDefVTables[c.mangledNameType(typeDef)]
	}

	// helper function to cast non-primitive from any to their concrete type
	nonPrimitiveAnyCast := func() {
		nonPrimTyp := targetIrType

		dest := c.NewAlloca(nonPrimTyp.LLType())
		c.createIfElse(c.compareAnyType(lhs.irVal, vtable), func() {
			// temporary values can be claimed
			if lhs.isImmediate {
				val_ptr := c.loadAnyValuePtr(lhs.irVal, nonPrimTyp.LLType())
				val := c.builder().CreateLoad(nonPrimTyp.LLType(), val_ptr, "")
				c.builder().CreateStore(val, dest)
				c.createIfElse(c.isSmallAny(lhs.irVal), func() {}, func() {
					c.ddp_reallocate(val_ptr, c.newInt(int64(c.getTypeSize(nonPrimTyp))), c.zero)
				})
				c.scp.claimTemporary(lhs.irVal) // don't call free func on the now invalid any
			} else {
				// non-temporaries are simply deep copied
				c.deepCopyInto(dest, c.loadAnyValuePtr(lhs.irVal, nonPrimTyp.LLType()), nonPrimTyp)
			}
			c.builder().latestReturn = c.scp.addTemporary(dest, nonPrimTyp)
		}, func() {
			line, column := int64(e.Token().Range.Start.Line), int64(e.Token().Range.Start.Column)
			c.runtime_error(1, c.bad_cast_error_string, c.newInt(line), c.newInt(column))
		})
	}

	// helper function to cast primitive from any to their concrete type
	primitiveAnyCast := func(primTyp ddpIrType) {
		c.createIfElse(c.compareAnyType(lhs.irVal, vtable), func() {
			c.builder().latestReturn = newImmediate(c.loadSmallAnyValue(lhs.irVal, primTyp.LLType()), primTyp)
		}, func() {
			line, column := int64(e.Token().Range.Start.Line), int64(e.Token().Range.Start.Column)
			c.runtime_error(1, c.bad_cast_error_string, c.newInt(line), c.newInt(column))
		})
	}

	lhsRefTyp, isRefLhs := lhs.typ.(*ddpIrReferenceType)
	targetRefTyp, isRefTarget := targetIrType.(*ddpIrReferenceType)

	// cast to ref
	if !isRefLhs && ddptypes.IsReference(targetType) {
		if lhs.typ == c.ddpany {
			primitiveAnyCast(targetIrType)
			return ast.VisitRecurse
		}

		ref := c.allocateGCRef(lhs.typ.VTable())
		c.claimOrCopy(ref, lhs)
		c.builder().latestReturn.irVal = ref
		c.builder().latestReturn.isImmediate = false
		c.builder().latestReturn.typ = targetIrType
		return ast.VisitRecurse
	} else if isRefLhs && isRefTarget {
		if lhsRefTyp == targetRefTyp {
			c.builder().latestReturn = newNonImmediate(lhs.irVal, targetIrType)
		} else if lhsRefTyp.underlying == targetRefTyp {
			// TODO
		} else if lhsRefTyp.underlying == c.ddpany {
			lhs.irVal = c.builder().CreateLoad(c.ddpany.typ, lhs.irVal, "")
			primitiveAnyCast(targetIrType)
		}

		return ast.VisitRecurse
	} else if isRefLhs {
		if lhsRefTyp.underlying.TriviallyCopyable() {
			lhs = newNonImmediate(c.builder().CreateLoad(lhsRefTyp.underlying.LLType(), lhs.irVal, ""), lhsRefTyp.underlying)
		} else {
			lhs.typ = lhsRefTyp.underlying
		}
	}

	if ddptypes.IsList(targetType) {
		if lhs.typ == c.ddpany {
			nonPrimitiveAnyCast()
			return ast.VisitRecurse
		}

		listType := c.getListType(lhs.typ)
		list := c.NewAlloca(listType.typ)
		c.builder().createCall(listType.fromConstantsIrFun, list, c.newInt(1))
		elementPtr := c.indexArray(listType.elementType.LLType(), c.loadStructField(listType.typ, list, list_arr_field_index), c.zero)
		c.claimOrCopy(elementPtr, lhs)
		c.builder().latestReturn = c.scp.addTemporary(list, listType)
	} else {
		switch targetType {
		case ddptypes.ZAHL:
			switch lhs.typ {
			case c.ddpinttyp, c.ddpfloattyp, c.ddpbytetyp:
				c.builder().latestReturn.irVal = c.numericCast(lhs.irVal, lhs.typ, targetIrType)
			case c.ddpbooltyp:
				cond := c.builder().CreateICmp(llvm.IntNE, lhs.irVal, c.False, "")
				c.builder().latestReturn.irVal = c.builder().CreateZExt(cond, c.ddpint, "")
			case c.ddpchartyp:
				c.builder().latestReturn.irVal = c.builder().CreateSExt(lhs.irVal, c.ddpint, "")
			case c.ddpstring:
				c.builder().latestReturn.irVal = c.builder().createCall(c.functions["ddp_string_to_int"].irFunc, lhs.irVal)
			case c.ddpany:
				primitiveAnyCast(c.ddpinttyp)
			default:
				c.err("invalid Parameter Type for ZAHL: %s", lhs.typ.Name())
			}
		case ddptypes.KOMMAZAHL:
			switch lhs.typ {
			case c.ddpinttyp, c.ddpfloattyp, c.ddpbytetyp:
				c.builder().latestReturn.irVal = c.numericCast(lhs.irVal, lhs.typ, targetIrType)
			case c.ddpstring:
				c.builder().latestReturn.irVal = c.builder().createCall(c.functions["ddp_string_to_float"].irFunc, lhs.irVal)
			case c.ddpany:
				primitiveAnyCast(c.ddpfloattyp)
			default:
				c.err("invalid Parameter Type for KOMMAZAHL: %s", lhs.typ.Name())
			}
		case ddptypes.BYTE:
			switch lhs.typ {
			case c.ddpinttyp, c.ddpfloattyp, c.ddpbytetyp:
				c.builder().latestReturn.irVal = c.numericCast(lhs.irVal, lhs.typ, targetIrType)
			case c.ddpbooltyp:
				cond := c.builder().CreateICmp(llvm.IntNE, lhs.irVal, c.False, "")
				c.builder().latestReturn.irVal = c.builder().CreateZExt(cond, c.ddpbyte, "")
			case c.ddpchartyp:
				c.builder().latestReturn.irVal = c.builder().CreateTrunc(lhs.irVal, c.ddpbyte, "")
			case c.ddpstring:
				intVal := c.builder().createCall(c.functions["ddp_string_to_int"].irFunc, lhs.irVal)
				c.builder().latestReturn.irVal = c.builder().CreateTrunc(intVal, c.ddpbyte, "")
			case c.ddpany:
				primitiveAnyCast(c.ddpinttyp)
			default:
				c.err("invalid Parameter Type for ZAHL: %s", lhs.typ.Name())
			}
		case ddptypes.WAHRHEITSWERT:
			switch lhs.typ {
			case c.ddpinttyp:
				c.builder().latestReturn.irVal = c.builder().CreateICmp(llvm.IntNE, lhs.irVal, c.zero, "")
			case c.ddpbytetyp:
				c.builder().latestReturn.irVal = c.builder().CreateICmp(llvm.IntNE, lhs.irVal, c.zero8, "")
			case c.ddpbooltyp:
				c.builder().latestReturn.irVal = lhs.irVal
			case c.ddpany:
				primitiveAnyCast(c.ddpbooltyp)
			default:
				c.err("invalid Parameter Type for WAHRHEITSWERT: %s", lhs.typ.Name())
			}
		case ddptypes.BUCHSTABE:
			switch lhs.typ {
			case c.ddpinttyp:
				c.builder().latestReturn.irVal = c.builder().CreateTrunc(lhs.irVal, c.ddpchar, "")
			case c.ddpbytetyp:
				c.builder().latestReturn.irVal = c.builder().CreateZExt(lhs.irVal, c.ddpchar, "")
			case c.ddpchartyp:
				c.builder().latestReturn.irVal = lhs.irVal
			case c.ddpany:
				primitiveAnyCast(c.ddpchartyp)
			default:
				c.err("invalid Parameter Type for BUCHSTABE: %s", lhs.typ.Name())
			}
		case ddptypes.TEXT:
			if lhs.typ == c.ddpany {
				nonPrimitiveAnyCast()
				return ast.VisitRecurse
			}

			if lhs.typ == c.ddpstring {
				c.builder().latestReturn = lhs
				return ast.VisitRecurse // don't free lhs
			}

			var to_string_func llvm.Value
			switch lhs.typ {
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
				c.err("invalid Parameter Type for TEXT: %s", lhs.typ.Name())
			}
			dest := c.NewAlloca(c.ddpstring.typ)
			c.builder().createCall(to_string_func, dest, lhs.irVal)
			c.builder().latestReturn = c.scp.addTemporary(dest, c.ddpstring)
		case ddptypes.VARIABLE:
			if lhs.typ == c.ddpany {
				break
			}

			c.builder().latestReturn = c.castNonAnyToAny(lhs, lhs.typ.VTable())
		default:
			if lhs.typ == c.ddpany {
				nonPrimitiveAnyCast()
				return ast.VisitRecurse
			}
			// this is now valid because of typedefs/typealiases
			// c.err("Invalide Typumwandlung zu %s (%s)", e.TargetType, targetType)
		}
	}
	c.builder().latestReturn.typ = targetIrType
	return ast.VisitRecurse
}

func (c *compiler) VisitTypeOpExpr(e *ast.TypeOpExpr) ast.VisitResult {
	switch e.Operator {
	case ast.TYPE_SIZE:
		c.builder().latestReturn.irVal = c.sizeof(c.toIrType(e.Rhs).LLType())
		c.builder().latestReturn.typ = c.ddpinttyp
	case ast.TYPE_DEFAULT:
		switch t := ddptypes.TrueUnderlying(e.Rhs).(type) {
		case *ddptypes.StructType:
			result, resultType := c.evaluateStructLiteral(t, nil)
			c.builder().latestReturn = c.scp.addTemporary(result, resultType)
		default:
			irType := c.toIrType(e.Rhs)
			defaultValue := irType.DefaultValue()
			if !irType.TriviallyCopyable() {
				dest := c.NewAlloca(irType.LLType())
				c.builder().CreateStore(defaultValue, dest)
				defaultValue = dest
			}
			c.builder().latestReturn = c.scp.addTemporary(defaultValue, irType)
		}
	default:
		c.err("invalid TypeOpExpr Operator: %d", e.Operator)
	}
	c.builder().latestReturn.isImmediate = true
	return ast.VisitRecurse
}

func (c *compiler) VisitTypeCheck(e *ast.TypeCheck) ast.VisitResult {
	lhs := c.evaluate(e.Lhs)

	vtable := c.toIrType(e.CheckType).VTable()
	if typeDef, isTypeDef := ddptypes.CastTypeDef(e.CheckType); isTypeDef {
		vtable = c.typeDefVTables[c.mangledNameType(typeDef)]
	}

	c.builder().latestReturn.irVal = c.compareAnyType(lhs.irVal, vtable)
	c.builder().latestReturn.typ = c.ddpbooltyp
	return ast.VisitRecurse
}

func (c *compiler) VisitGrouping(e *ast.Grouping) ast.VisitResult {
	e.Expr.Accept(c) // visit like a normal expression, grouping is just precedence stuff which has already been parsed
	return ast.VisitRecurse
}

// helper for VisitAssignStmt
func (c *compiler) evaluateAssignableOrReference(ass ast.Expression) (ddpValue, *ast.BinaryExpr) {
	if _, isStringIndexing := ass.GetMetadataByKind(ast.StringIndexingMetaKind); isStringIndexing {
		lhs := c.evaluate(ass)
		return lhs, ass.(*ast.BinaryExpr)
	}

	return c.evaluate(ass), nil
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

	irReturnType := c.getPossiblyGenericReturnType(fun.funcDecl)

	var ret llvm.Value
	if !irReturnType.TriviallyCopyable() {
		ret = c.NewAlloca(irReturnType.LLType())
		args = append(args, ret)
	}

	for _, param := range fun.funcDecl.Parameters {
		var val llvm.Value

		eval := c.evaluate(e.Args[param.Name.Literal]) // compile each argument for the function
		if eval.typ.TriviallyCopyable() {
			val = eval.irVal
		} else { // function parameters need to be copied by the caller
			dest := c.NewAlloca(eval.typ.LLType())
			c.claimOrCopy(dest, eval)
			val = dest // do not add it to the temporaries, as the callee will free it
		}

		args = append(args, val) // add the value to the arguments
	}

	// compile the actual function call
	if irReturnType.TriviallyCopyable() {
		c.builder().latestReturn.irVal = c.builder().createCall(fun.irFunc, args...)
	} else {
		c.builder().createCall(fun.irFunc, args...)
		c.builder().latestReturn = c.scp.addTemporary(ret, irReturnType)
	}
	c.builder().latestReturn.typ = irReturnType

	// the arguments of external functions must be freed by the caller
	// normal functions free their parameters in their body
	if !ast.IsExternFunc(e.Func) {
		return ast.VisitRecurse
	}

	for i, param := range e.Func.Parameters {
		if !ddptypes.IsReference(param.Type) {
			paramIrType := c.toIrType(param.Type)
			arg := args[i]
			if !irReturnType.TriviallyCopyable() {
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
		initType := typechecker.TypeOfTypecheckedExpression(fieldDecl.InitVal)
		argExpr := fieldDecl.InitVal
		if fieldArg, hasArg := args[field.Name]; hasArg {
			// the arg was passed so use that instead
			argExpr = fieldArg
		}

		// if no default value was given
		if argExpr == nil {
			argExpr = &ast.TypeOpExpr{Operator: ast.TYPE_DEFAULT, Rhs: field.Type, Range: c.builder().currentNode.GetRange()}
		}

		arg := c.evaluate(argExpr)

		// implicit cast to any if required
		if ddptypes.DeepEqual(field.Type, ddptypes.VARIABLE) && arg.typ != c.ddpany {
			vtable := arg.typ.VTable()
			if typeDef, isTypeDef := ddptypes.CastTypeDef(initType); isTypeDef {
				vtable = c.typeDefVTables[c.mangledNameType(typeDef)]
			}

			arg = c.castNonAnyToAny(arg, vtable)
		}

		c.claimOrCopy(c.indexStruct(resultType.LLType(), result, i), arg)
	}
	return result, resultType
}

func (c *compiler) VisitStructLiteral(expr *ast.StructLiteral) ast.VisitResult {
	result, resultType := c.evaluateStructLiteral(expr.Type, expr.Args)
	c.builder().latestReturn = c.scp.addTemporary(result, resultType)
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
		c.declareIfStruct(param.Type)
	}

	retType := c.getPossiblyGenericReturnType(decl) // get the llvm type
	retTypeIr := retType.LLType()
	params := make([]llvm.Type, 0, len(decl.Parameters)+1)  // list of the ir parameters
	paramNames := make([]string, 0, len(decl.Parameters)+1) // list of the ir parameters
	paramAttributes := make([][]llvm.Attribute, 0, len(decl.Parameters)+1)

	hasReturnParam := !retType.TriviallyCopyable()
	// non-primitives are returned by passing a pointer to the struct as first parameter
	if hasReturnParam {
		params = append(params, c.ptr)
		paramNames = append(paramNames, "")
		paramAttributes = append(paramAttributes, []llvm.Attribute{c.attr_nonnull, c.attr_noalias}) // the return param is never null and never aliased
		retTypeIr = c.voidtyp.LLType()
	}

	// append all the other parameters
	for _, param := range decl.Parameters {
		ty, irType := c.getPossiblyGenericParamType(&param) // convert the type of the parameter
		params = append(params, ty)                         // add it to the list
		paramNames = append(paramNames, param.Name.Literal)

		var attributes []llvm.Attribute = nil

		if irType != nil {
			switch irType.(type) {
			case *ddpIrAnyType, *ddpIrListType, *ddpIrStringType, *ddpIrStructType, *ddpIrGenericListType:
				attributes = append(attributes, c.attr_nonnull)
			}
		}

		paramAttributes = append(paramAttributes, attributes)
	}

	llFuncTyp := llvm.FunctionType(retTypeIr, params, false)

	llFuncBuilder := c.createBuilder(mangledName, llFuncTyp, nil, paramNames, paramAttributes, !ast.IsExternFunc(decl), true)
	// declare it as extern function
	llFuncBuilder.llFn.SetLinkage(llvm.ExternalLinkage)
	llFuncBuilder.llFn.SetVisibility(llvm.DefaultVisibility)
	c.insertFunction(llFuncBuilder.fnName, decl, llFuncBuilder.llFn, llFuncBuilder)
}

func (c *compiler) declareImportedVarDecl(decl *ast.VarDecl) {
	// imported decls are always in the global scope
	// even in generic instantiations
	scp := c.scp
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

	scp.addProtected(decl, globalDecl, Typ) // freed by module_dispose
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
			c.addTypdefVTable(decl)
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

			moduleInitBuilder := c.createBuilder(init_name, llvm.FunctionType(c.void, nil, false), nil, nil, nil, true, true)
			moduleInitBuilder.llFn.SetLinkage(llvm.ExternalLinkage)
			moduleInitBuilder.llFn.SetVisibility(llvm.DefaultVisibility)

			c.insertFunction(moduleInitBuilder.fnName, nil, moduleInitBuilder.llFn, moduleInitBuilder)
			if c.builder().isDDPMain() {
				c.builder().createCall(moduleInitBuilder.llFn) // only call this in main modules
			}

			moduleDisposeBuilder := c.createBuilder(dispose_name, llvm.FunctionType(c.void, nil, false), nil, nil, nil, true, true)
			moduleDisposeBuilder.llFn.SetLinkage(llvm.ExternalLinkage)
			moduleDisposeBuilder.llFn.SetVisibility(llvm.DefaultVisibility)

			c.insertFunction(moduleDisposeBuilder.fnName, nil, moduleDisposeBuilder.llFn, moduleDisposeBuilder)

			c.importedModules[module] = struct{}{}
		})
	}
	return ast.VisitRecurse
}

func (c *compiler) VisitAssignStmt(s *ast.AssignStmt) ast.VisitResult {
	var rhs ddpValue

	varDDPType, rhsDDPType := typechecker.TypeOfTypecheckedExpression(s.Var), typechecker.TypeOfTypecheckedExpression(s.Rhs)

	// implicit numeric casts
	if ddptypes.IsNumericDeref(varDDPType) && ddptypes.IsNumericDeref(rhsDDPType) {
		numericType := c.toIrType(varDDPType)
		for ref, ok := numericType.(*ddpIrReferenceType); ok; ref, ok = numericType.(*ddpIrReferenceType) {
			numericType = ref.underlying
		}
		rhs = c.evaluateNumeric(s.Rhs, numericType) // evaluate the initial value
	} else {
		rhs = c.evaluate(s.Rhs) // evaluate the initial value
	}

	lhs, lhsStringIndexing := c.evaluateAssignableOrReference(s.Var)

	if lhsStringIndexing != nil {
		index := c.evaluate(lhsStringIndexing.Rhs)
		index.irVal = c.floatOrByteAsInt(index.irVal, index.typ)
		c.builder().createCall(c.ddpstring.replaceCharIrFun, lhs.irVal, rhs.irVal, index.irVal)
	} else {
		if lhs.isStackRef {
			c.freeNonPrimitive(lhs.irVal, lhs.typ.(*ddpIrReferenceType).underlying) // free the old value in the variable/list
		} else {
			c.freeNonPrimitive(lhs.irVal, lhs.typ) // free the old value in the variable/list
		}

		// implicit cast to any if required
		if ddptypes.IsAnyDeref(varDDPType) && rhs.typ != c.ddpany {
			vtable := rhs.typ.VTable()
			if typeDef, isTypeDef := ddptypes.CastTypeDef(rhsDDPType); isTypeDef {
				vtable = c.typeDefVTables[c.mangledNameType(typeDef)]
			}
			rhs = c.castNonAnyToAny(rhs, vtable)
		}

		c.claimOrCopy(lhs.irVal, rhs) // copy/claim the new value
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

func (c *compiler) VisitIfStmt(s *ast.IfStmt) ast.VisitResult {
	cond := c.evaluate(s.Condition)
	thenBlock, elseBlock, leaveBlock := c.builder().newBlock(), c.builder().newBlock(), c.builder().newBlock()
	if s.Else != nil {
		c.builder().CreateCondBr(cond.irVal, thenBlock, elseBlock)
	} else {
		c.builder().CreateCondBr(cond.irVal, thenBlock, leaveBlock)
	}

	c.builder().setBlock(thenBlock)
	c.scp = newScope(c.scp)
	c.visitNode(s.Then)
	if c.builder().cb.Terminator().IsNil() {
		c.builder().CreateBr(leaveBlock)
	}
	c.scp = c.exitScope(c.scp)

	if s.Else != nil {
		c.builder().setBlock(elseBlock)
		c.scp = newScope(c.scp)
		c.visitNode(s.Else)
		if c.builder().cb.Terminator().IsNil() {
			c.builder().CreateBr(leaveBlock)
		}
		c.scp = c.exitScope(c.scp)
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
		condBlock, body, bodyScope := c.builder().newBlock(), c.builder().newBlock(), newScope(c.scp)
		breakLeave := c.builder().newBlock()
		c.builder().curLoopScope, c.builder().curLeaveBlock, c.builder().curContinueBlock = bodyScope, breakLeave, condBlock

		if op == token.SOLANGE {
			c.builder().CreateBr(condBlock)
		} else {
			c.builder().CreateBr(body)
		}

		c.builder().setBlock(body)
		c.scp = bodyScope
		c.visitNode(s.Body)
		if c.builder().cb.Terminator().IsNil() {
			c.builder().CreateBr(condBlock)
		}

		c.builder().setBlock(condBlock)
		c.scp = c.exitScope(c.scp)
		cond := c.evaluate(s.Condition)
		leaveBlock := c.builder().newBlock()
		c.builder().CreateCondBr(cond.irVal, body, leaveBlock)

		trueLeave := c.builder().newBlock()
		c.builder().withBlock(leaveBlock, func() { c.builder().CreateBr(trueLeave) })
		c.builder().withBlock(breakLeave, func() { c.builder().CreateBr(trueLeave) })
		c.builder().setBlock(trueLeave)
	case token.WIEDERHOLE:
		counter := c.NewAlloca(c.ddpint)
		cond := c.evaluate(s.Condition)
		c.builder().CreateStore(cond.irVal, counter)
		condBlock, body, bodyScope := c.builder().newBlock(), c.builder().newBlock(), newScope(c.scp)
		breakLeave := c.builder().newBlock()
		c.builder().curLoopScope, c.builder().curLeaveBlock, c.builder().curContinueBlock = bodyScope, breakLeave, condBlock

		c.builder().CreateBr(condBlock)

		c.builder().setBlock(body)
		c.scp = bodyScope
		c.builder().CreateStore(c.builder().CreateSub(c.builder().CreateLoad(c.ddpint, counter, ""), c.newInt(1), ""), counter)
		c.visitNode(s.Body)
		if c.builder().cb.Terminator().IsNil() {
			c.builder().CreateBr(condBlock)
		}

		leaveBlock := c.builder().newBlock()
		c.builder().setBlock(condBlock)
		c.scp = c.exitScope(c.scp)
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

	c.scp = newScope(c.scp)    // scope for the for body
	c.visitNode(s.Initializer) // compile the counter variable declaration
	Var := c.scp.lookupVar(s.Initializer)
	// this is the actual index used
	var indexVar llvm.Value
	var indexTyp ddpIrType
	if ddptypes.DeepEqual(s.Initializer.Type, ddptypes.KOMMAZAHL) {
		indexVar, indexTyp = c.NewAlloca(c.ddpfloat), c.ddpfloattyp
	} else {
		indexVar, indexTyp = c.NewAlloca(c.ddpint), c.ddpinttyp
	}
	var incrementer ddpValue // Schrittgröße
	// if no stepsize was present it is 1
	if s.StepSize == nil {
		if ddptypes.DeepEqual(s.Initializer.Type, ddptypes.KOMMAZAHL) {
			incrementer = newImmediate(llvm.ConstFloat(c.ddpfloat, 1.0), c.ddpfloattyp)
		} else {
			incrementer = newImmediate(c.newInt(1), c.ddpinttyp)
		}
	} else { // stepsize was present, so compile it
		incrementer = c.evaluate(s.StepSize)
	}

	condBlock, incrementBlock, forBody, breakLeave := c.builder().newBlock(), c.builder().newBlock(), c.builder().newBlock(), c.builder().newBlock()

	c.builder().curLoopScope, c.builder().curLeaveBlock, c.builder().curContinueBlock = c.scp, breakLeave, incrementBlock

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
		add = c.builder().CreateFAdd(indexVal, c.intOrByteAsFloat(incrementer.irVal, incrementer.typ), "")
		c.builder().CreateStore(add, Var.val)
	} else {
		add = c.builder().CreateAdd(indexVal, c.floatOrByteAsInt(incrementer.irVal, incrementer.typ), "")
		c.builder().CreateStore(c.numericCast(add, c.ddpinttyp, Var.typ), Var.val)
	}
	c.builder().CreateStore(add, indexVar)
	c.builder().CreateBr(condBlock) // check the condition (loop)

	// finally compile the condition block(s)
	loopDown, loopUp, leaveBlock := c.builder().newBlock(), c.builder().newBlock(), c.builder().newBlock()

	c.builder().setBlock(condBlock)
	// we check the counter differently depending on wether or not we are looping up or down (positive vs negative stepsize)
	cond := new_IorF_comp(llvm.IntSLT, llvm.FloatOLT, incrementer.irVal, incrementer.typ, c.newInt(0), c.ddpinttyp, llvm.ConstFloat(c.ddpfloat, 0.0))
	c.builder().CreateCondBr(cond, loopDown, loopUp)

	c.builder().setBlock(loopUp)
	c.scp = newScope(c.scp) // new scope to not create a double-free from s.To
	to := c.evaluate(s.To)
	// we are counting up, so compare less-or-equal
	cond = new_IorF_comp(llvm.IntSLE, llvm.FloatOLE, c.builder().CreateLoad(indexTyp.LLType(), indexVar, ""), indexTyp, to.irVal, to.typ, to.irVal)
	c.scp = c.exitScope(c.scp)
	c.builder().CreateCondBr(cond, forBody, leaveBlock)

	c.builder().setBlock(loopDown)
	c.scp = newScope(c.scp) // new scope to not create a double-free from s.To
	to = c.evaluate(s.To)
	// we are counting down, so compare greater-or-equal
	cond = new_IorF_comp(llvm.IntSGE, llvm.FloatOGE, c.builder().CreateLoad(indexTyp.LLType(), indexVar, ""), indexTyp, to.irVal, to.typ, to.irVal)
	c.scp = c.exitScope(c.scp)
	c.builder().CreateCondBr(cond, forBody, leaveBlock)

	trueLeave := c.builder().newBlock()

	c.builder().setBlock(leaveBlock)
	c.scp = c.exitScope(c.scp) // leave the scope
	c.builder().CreateBr(trueLeave)

	c.builder().setBlock(breakLeave)
	c.builder().CreateBr(trueLeave)

	c.builder().setBlock(trueLeave)

	c.builder().curLoopScope, c.builder().curLeaveBlock, c.builder().curContinueBlock = loopScopeBack, leaveBlockBack, continueBlockBack
	return ast.VisitRecurse
}

func (c *compiler) VisitForRangeStmt(s *ast.ForRangeStmt) ast.VisitResult {
	loopScopeBack, leaveBlockBack, continueBlockBack := c.builder().curLoopScope, c.builder().curLeaveBlock, c.builder().curContinueBlock

	c.scp = newScope(c.scp)
	in := c.evaluate(s.In)

	if refType, isRef := in.typ.(*ddpIrReferenceType); isRef {
		in.typ, in.isImmediate = refType.underlying, false
	}

	temp := c.NewAlloca(in.typ.LLType())
	c.claimOrCopy(temp, in)
	in = c.scp.addTemporary(temp, in.typ)
	c.scp.protectTemporary(in.irVal)

	var (
		end_ptr llvm.Value // points to the one-after-last element
		length  llvm.Value
		index   llvm.Value = c.NewAlloca(c.ddpinttyp.LLType())
	)

	iter_ptr := c.NewAlloca(c.ptr) // pointer used for iteration

	if in.typ == c.ddpstring {
		iter_ptr_val := c.loadStructField(c.ddpstring.typ, in.irVal, string_str_field_index)
		c.builder().CreateStore(iter_ptr_val, iter_ptr)
		length = c.loadStructField(c.ddpstring.typ, in.irVal, string_cap_field_index)
		end_ptr = c.indexArray(c.i8, iter_ptr_val, c.builder().CreateSub(length, c.newInt(1), ""))
	} else {
		iter_ptr_val := c.loadStructField(in.typ.LLType(), in.irVal, list_arr_field_index)
		c.builder().CreateStore(iter_ptr_val, iter_ptr)
		length = c.loadStructField(in.typ.LLType(), in.irVal, list_len_field_index)
		end_ptr = c.indexArray(in.typ.(*ddpIrListType).elementType.LLType(), iter_ptr_val, length)
	}

	loopStart, condBlock, bodyBlock, incrementBlock, leaveBlock := c.builder().newBlock(), c.builder().newBlock(), c.builder().newBlock(), c.builder().newBlock(), c.builder().newBlock()
	c.builder().CreateCondBr(c.builder().CreateICmp(llvm.IntEQ, length, c.zero, ""), leaveBlock, loopStart)

	c.builder().setBlock(loopStart)
	irType := c.toIrType(s.Initializer.Type)
	c.scp.addProtected(s.Initializer, c.NewAlloca(irType.LLType()), irType)
	if s.Index != nil {
		c.scp.addVar(s.Index, index, c.ddpinttyp)
		c.builder().CreateStore(c.newInt(1), index)
	}
	c.builder().CreateBr(condBlock)

	c.builder().setBlock(condBlock)
	c.builder().CreateCondBr(c.builder().CreateICmp(llvm.IntNE, c.builder().CreateLoad(c.ptr, iter_ptr, ""), end_ptr, ""), bodyBlock, leaveBlock)

	loopVar := c.scp.lookupVar(s.Initializer)

	continueBlock := c.builder().newBlock()
	c.builder().setBlock(continueBlock)
	c.freeNonPrimitive(loopVar.val, loopVar.typ)
	c.builder().CreateBr(incrementBlock)

	c.builder().setBlock(bodyBlock)
	var num_bytes llvm.Value
	if in.typ == c.ddpstring {
		num_bytes = c.builder().createCall(utf8_string_to_char_irfun,
			c.builder().CreateLoad(c.ptr, iter_ptr, ""),
			loopVar.val,
		)
		c.createIfElse(c.builder().CreateICmp(llvm.IntEQ, num_bytes, c.all_ones, ""), func() {
			line, column := int64(s.In.Token().Range.Start.Line), int64(s.In.Token().Range.Start.Column)
			c.runtime_error(1, c.invalid_utf8_error_string, c.newInt(line), c.newInt(column))
		}, func() {})
	} else {
		elementPtr := c.builder().CreateLoad(c.ptr, iter_ptr, "")
		inListTyp := in.typ.(*ddpIrListType)
		if inListTyp.elementType.TriviallyCopyable() {
			element := c.builder().CreateLoad(inListTyp.elementType.LLType(), elementPtr, "")
			c.builder().CreateStore(element, loopVar.val)
		} else {
			c.deepCopyInto(loopVar.val, elementPtr, inListTyp.elementType)
		}
	}
	breakLeave := c.builder().newBlock()
	c.builder().curLoopScope, c.builder().curLeaveBlock, c.builder().curContinueBlock = c.scp, breakLeave, continueBlock
	c.visitNode(s.Body)
	c.freeNonPrimitive(loopVar.val, loopVar.typ)
	if c.builder().cb.Terminator().IsNil() {
		c.builder().CreateBr(incrementBlock)
	}

	c.builder().setBlock(incrementBlock)
	if in.typ == c.ddpstring {
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
		inListTyp := in.typ.(*ddpIrListType)
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
	c.scp.unprotectTemporary(in.irVal)
	// delete(c.scp.variables, s.Initializer.Name()) // the loopvar was already freed
	c.scp = c.exitScope(c.scp)

	trueLeave := c.builder().newBlock()

	c.builder().setBlock(breakLeave)
	c.freeNonPrimitive(in.irVal, in.typ)
	c.freeNonPrimitive(loopVar.val, loopVar.typ)
	c.builder().CreateBr(trueLeave)

	c.builder().withBlock(leaveBlock, func() { c.builder().CreateBr(trueLeave) })

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
		// don't overwrite a possible return
		if !c.builder().cb.Terminator().IsNil() {
			c.builder().SetInsertPointBefore(c.builder().cb.LastInstruction())
			defer func() {
				c.builder().SetInsertPointAtEnd(c.builder().cb)
			}()
		}

		for scp := c.scp; scp != c.fnScope; scp = scp.enclosing {
			for _, Var := range scp.variables {
				c.freeNonPrimitive(Var.val, Var.typ)
			}
			c.freeTemporaries(scp, true)
		}
		c.exitFuncScope()
	}

	if s.Value == nil {
		exitScopeReturn()
		c.builder().CreateRet(llvm.Value{})
		return ast.VisitRecurse
	}
	val := c.evaluate(s.Value)
	vtable := val.typ.VTable()
	if typeDef, isTypeDef := ddptypes.CastTypeDef(s.Func.ReturnType); isTypeDef {
		vtable = c.typeDefVTables[c.mangledNameType(typeDef)]
	}
	if val.typ.TriviallyCopyable() {
		// implicit cast to any if required
		if ddptypes.DeepEqual(s.Func.ReturnType, ddptypes.VARIABLE) && val.typ != c.ddpany {

			val = c.castNonAnyToAny(val, vtable)
			c.claimOrCopy(c.builder().params[0].val, val)
			c.builder().CreateRet(llvm.Value{})
		} else {
			// normal return
			c.builder().CreateRet(val.irVal)
		}
	} else {
		// implicit cast to any if required
		if ddptypes.DeepEqual(s.Func.ReturnType, ddptypes.VARIABLE) && val.typ != c.ddpany {
			val = c.castNonAnyToAny(val, vtable)
		}

		c.claimOrCopy(c.builder().params[0].val, val)
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
	for scp := c.scp; scp != targetScope.enclosing; scp = c.exitScope(scp) {
	}
}

func (c *compiler) addTypdefVTable(d *ast.TypeDefDecl) {
	name := c.mangledNameType(d.Type)
	if _, ok := c.typeDefVTables[name]; ok {
		return
	}

	ir_type := c.toIrType(d.Type)

	vtable := llvm.AddGlobal(c.llmod, c.vtable_type, name+"_vtable")
	vtable.SetLinkage(llvm.WeakODRLinkage) // weak_odr to combine vtables, which are equivalent in all modules, see https://llvm.org/docs/LangRef.html#linkage
	vtable.SetVisibility(llvm.DefaultVisibility)

	vtable.SetGlobalConstant(true)
	vtable.SetInitializer(llvm.ConstNamedStruct(c.vtable_type, []llvm.Value{
		llvm.ConstInt(c.ddpint, c.getTypeSize(ir_type), false),
		ir_type.FreeFunc(),
		ir_type.DeepCopyFunc(),
		ir_type.EqualsFunc(),
		c.zero, // TODO: ptrmask
	}))

	c.typeDefVTables[name] = vtable
}
