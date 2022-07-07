package compiler

import (
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/DDP-Projekt/Kompilierer/pkg/ast"
	"github.com/DDP-Projekt/Kompilierer/pkg/scanner"
	"github.com/DDP-Projekt/Kompilierer/pkg/token"
	"github.com/bafto/Go-LLVM-Bindings/llvm"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"

	"github.com/llir/irutil"
)

// small wrapper for a ast.FuncDecl and the corresponding ir function
type funcWrapper struct {
	irFunc   *ir.Func      // the function in the llvm ir
	funcDecl *ast.FuncDecl // the ast.FuncDecl
}

// holds state to compile a DDP AST into llvm ir
type Compiler struct {
	ast          *ast.Ast
	mod          *ir.Module           // the ir module (basically the ir file)
	errorHandler scanner.ErrorHandler // errors are passed to this function

	cbb          *ir.Block               // current basic block in the ir
	cf           *ir.Func                // current function
	scp          *scope                  // current scope in the ast (not in the ir)
	functions    map[string]*funcWrapper // all the global functions
	latestReturn value.Value             // return of the latest evaluated expression (in the ir)
}

// create a new Compiler to compile the passed AST
func New(Ast *ast.Ast, errorHandler scanner.ErrorHandler) *Compiler {
	if errorHandler == nil { // default error handler does nothing
		errorHandler = func(token.Token, string) {}
	}
	return &Compiler{
		ast:          Ast,
		mod:          ir.NewModule(),
		errorHandler: errorHandler,
		cbb:          nil,
		cf:           nil,
		scp:          newScope(nil), // global scope
		functions:    map[string]*funcWrapper{},
		latestReturn: nil,
	}
}

// compile the AST contained in c
// if w is not nil, the resulting llir is written to w
// otherwise a string representation is returned
func (c *Compiler) Compile(w io.Writer) (result string, rerr error) {
	// catch panics and instead set the returned error
	defer func() {
		if err := recover(); err != nil {
			rerr = fmt.Errorf("%v", err)
			result = ""
		}
	}()

	// the ast must be valid (and should have been resolved and typechecked beforehand)
	if c.ast.Faulty {
		return "", fmt.Errorf("Fehlerhafter Syntax Baum")
	}

	c.mod.SourceFilename = c.ast.File // set the module filename (optional metadata)
	c.mod.TargetTriple = llvm.DefaultTargetTriple()
	c.mod.DataLayout = getDataLayout()
	c.setupRuntimeFunctions() // setup internal functions to interact with the ddp-c-runtime
	// called from the ddp-c-runtime after initialization
	ddpmain := c.insertFunction(
		"inbuilt_ddpmain",
		nil,
		c.mod.NewFunc("inbuilt_ddpmain", ddpint),
	)
	c.cf = ddpmain               // first function is ddpmain
	c.cbb = ddpmain.NewBlock("") // first block

	// visit every statement in the AST and compile it
	for _, stmt := range c.ast.Statements {
		c.visitNode(stmt)
	}

	c.scp = c.exitScope(c.scp) // exit the main scope

	// on success ddpmain returns 0
	c.cbb.NewRet(newInt(0))
	if w != nil {
		_, err := c.mod.WriteTo(w)
		return "", err
	} else {
		return c.mod.String(), nil // return the module as string
	}
}

// helper that might be extended later
// it is not intended for the end user to see these errors, as they are compiler bugs
// the errors in the ddp-code were already reported by the parser/typechecker/resolver
func err(msg string) {
	// retreive the file and line on which the error occured
	_, file, line, _ := runtime.Caller(1)
	panic(fmt.Errorf("%s, %d: %s", filepath.Base(file), line, msg))
}

func (c *Compiler) commentNode(block *ir.Block, node ast.Node, details string) {
	comment := fmt.Sprintf("File %s, Line %d, Column %d: %s", node.Token().File, node.Token().Line, node.Token().Column, node.String())
	if details != "" {
		comment += " (" + details + ")"
	}
	block.Insts = append(block.Insts, irutil.NewComment(comment))
}

func (c *Compiler) comment(comment string, block *ir.Block) {
	block.Insts = append(block.Insts, irutil.NewComment(comment))
}

// helper to visit a single node
func (c *Compiler) visitNode(node ast.Node) {
	c = node.Accept(c).(*Compiler)
}

// helper to evalueate an expression and return its ir value
func (c *Compiler) evaluate(expr ast.Expression) value.Value {
	c.visitNode(expr)
	return c.latestReturn
}

// helper to insert a function into the global function map
// returns the ir function
func (c *Compiler) insertFunction(name string, funcDecl *ast.FuncDecl, irFunc *ir.Func) *ir.Func {
	c.functions[name] = &funcWrapper{
		funcDecl: funcDecl,
		irFunc:   irFunc,
	}
	return irFunc
}

// helper to declare a inbuilt function
// sets the linkage to external, callingconvention to ccc
// and inserts the function into the compiler
func (c *Compiler) declareInbuiltFunction(name string, retType types.Type, params ...*ir.Param) {
	fun := c.mod.NewFunc(name, retType, params...)
	fun.CallingConv = enum.CallingConvC
	fun.Linkage = enum.LinkageExternal
	c.insertFunction(name, nil, fun)
}

// declares functions defined in the ddp-c-runtime
func (c *Compiler) setupRuntimeFunctions() {
	c.setupStringType()
	c.setupOperators()
}

// declares some internal string functions
// and completes the ddpstring struct
func (c *Compiler) setupStringType() {
	// complete the ddpstring definition to interact with the c ddp runtime
	ddpstring.Fields = make([]types.Type, 2)
	ddpstring.Fields[0] = ptr(i8)
	ddpstring.Fields[1] = ddpint
	c.mod.NewTypeDef("ddpstring", ddpstring)

	// declare all the external functions to work with strings

	// creates a ddpstring from a string literal and returns a pointer to it
	// the caller is responsible for calling increment_ref_count on this pointer
	c.declareInbuiltFunction("inbuilt_string_from_constant", ddpstrptr, ir.NewParam("str", ptr(i8)))

	// returns a copy of the passed string as a new pointer
	// the caller is responsible for calling increment_ref_count on this pointer
	c.declareInbuiltFunction("inbuilt_deep_copy_string", ddpstrptr, ir.NewParam("str", ddpstrptr))

	// decrement the ref-count on a pointer and
	// free the pointer if the ref-count becomes 0
	// takes the pointer and the type to which it points
	// (currently only string, but later lists and structs too)
	c.declareInbuiltFunction("inbuilt_decrement_ref_count", void, ir.NewParam("key", ptr(i8)))

	// increment the ref-count on a pointer
	// takes the pointer and the type to which it points
	// (currently only string, but later lists and structs too)
	c.declareInbuiltFunction("inbuilt_increment_ref_count", void, ir.NewParam("key", ptr(i8)), ir.NewParam("kind", i8))
}

func (c *Compiler) setupOperators() {
	// betrag operator for different types
	c.declareInbuiltFunction("inbuilt_int_betrag", ddpint, ir.NewParam("i", ddpint))
	c.declareInbuiltFunction("inbuilt_float_betrag", ddpfloat, ir.NewParam("f", ddpfloat))

	// hoch operator for different type combinations
	c.declareInbuiltFunction("inbuilt_hoch", ddpfloat, ir.NewParam("f1", ddpfloat), ir.NewParam("f2", ddpfloat))

	// trigonometric functions
	c.declareInbuiltFunction("inbuilt_sin", ddpfloat, ir.NewParam("f", ddpfloat))
	c.declareInbuiltFunction("inbuilt_cos", ddpfloat, ir.NewParam("f", ddpfloat))
	c.declareInbuiltFunction("inbuilt_tan", ddpfloat, ir.NewParam("f", ddpfloat))
	c.declareInbuiltFunction("inbuilt_asin", ddpfloat, ir.NewParam("f", ddpfloat))
	c.declareInbuiltFunction("inbuilt_acos", ddpfloat, ir.NewParam("f", ddpfloat))
	c.declareInbuiltFunction("inbuilt_atan", ddpfloat, ir.NewParam("f", ddpfloat))
	c.declareInbuiltFunction("inbuilt_sinh", ddpfloat, ir.NewParam("f", ddpfloat))
	c.declareInbuiltFunction("inbuilt_cosh", ddpfloat, ir.NewParam("f", ddpfloat))
	c.declareInbuiltFunction("inbuilt_tanh", ddpfloat, ir.NewParam("f", ddpfloat))

	// logarithm
	c.declareInbuiltFunction("inbuilt_log", ddpfloat, ir.NewParam("numerus", ddpfloat), ir.NewParam("base", ddpfloat))

	// ddpstring length
	c.declareInbuiltFunction("inbuilt_string_length", ddpint, ir.NewParam("str", ddpstrptr))

	// ddpstring to type cast
	c.declareInbuiltFunction("inbuilt_string_to_int", ddpint, ir.NewParam("str", ddpstrptr))
	c.declareInbuiltFunction("inbuilt_string_to_float", ddpfloat, ir.NewParam("str", ddpstrptr))

	// casts to ddpstring
	c.declareInbuiltFunction("inbuilt_int_to_string", ddpstrptr, ir.NewParam("i", ddpint))
	c.declareInbuiltFunction("inbuilt_float_to_string", ddpstrptr, ir.NewParam("f", ddpfloat))
	c.declareInbuiltFunction("inbuilt_bool_to_string", ddpstrptr, ir.NewParam("b", ddpbool))
	c.declareInbuiltFunction("inbuilt_char_to_string", ddpstrptr, ir.NewParam("c", ddpchar))

	// ddpstring equality
	c.declareInbuiltFunction("inbuilt_string_equal", ddpbool, ir.NewParam("str1", ddpstrptr), ir.NewParam("str2", ddpstrptr))

	// ddpstring concatenation for different types
	c.declareInbuiltFunction("inbuilt_string_string_verkettet", ddpstrptr, ir.NewParam("str1", ddpstrptr), ir.NewParam("str2", ddpstrptr))
	c.declareInbuiltFunction("inbuilt_char_string_verkettet", ddpstrptr, ir.NewParam("c", ddpchar), ir.NewParam("str", ddpstrptr))
	c.declareInbuiltFunction("inbuilt_string_char_verkettet", ddpstrptr, ir.NewParam("str", ddpstrptr), ir.NewParam("c", ddpchar))
	c.declareInbuiltFunction("inbuilt_char_char_verkettet", ddpstrptr, ir.NewParam("c1", ddpchar), ir.NewParam("c2", ddpchar))

	// string indexing
	c.declareInbuiltFunction("inbuilt_string_index", ddpchar, ir.NewParam("str", ddpstrptr), ir.NewParam("index", ddpint))
	c.declareInbuiltFunction("inbuilt_replace_char_in_string", void, ir.NewParam("str", ddpstrptr), ir.NewParam("ch", ddpchar), ir.NewParam("index", ddpint))
	c.declareInbuiltFunction("inbuilt_string_slice", ddpstrptr, ir.NewParam("str", ddpstrptr), ir.NewParam("index1", ddpint), ir.NewParam("index2", ddpint))
}

// helper to call increment_ref_count
func (c *Compiler) incrementRC(key value.Value, kind *constant.Int) {
	c.cbb.NewCall(c.functions["inbuilt_increment_ref_count"].irFunc, c.cbb.NewBitCast(key, ptr(i8)), kind)
}

// helper to call decrement_ref_count
func (c *Compiler) decrementRC(key value.Value) {
	c.cbb.NewCall(c.functions["inbuilt_decrement_ref_count"].irFunc, c.cbb.NewBitCast(key, ptr(i8)))
}

// helper to call deep_copy_string
func (c *Compiler) deepCopyStr(strptr value.Value) value.Value {
	return c.cbb.NewCall(c.functions["inbuilt_deep_copy_string"].irFunc, strptr)
}

// helper to exit a scope
// decrements the ref-count on all local variables
// returns the enclosing scope
func (c *Compiler) exitScope(scp *scope) *scope {
	for _, v := range c.scp.variables {
		if v.typ == ddpstrptr {
			c.decrementRC(c.cbb.NewLoad(ddpstrptr, v.val))
		}
	}
	return scp.enclosing
}

// should have been filtered by the resolver/typechecker, so err
func (c *Compiler) VisitBadDecl(d *ast.BadDecl) ast.Visitor {
	err("Es wurde eine invalide Deklaration gefunden")
	return c
}
func (c *Compiler) VisitVarDecl(d *ast.VarDecl) ast.Visitor {
	Typ := toDDPType(d.Type.Type) // get the llvm type
	// allocate the variable on the function call frame
	// all local variables are allocated in the first basic block of the function they are within
	// in the ir a local variable is a alloca instruction (a stack allocation)
	// global variables are allocated in the ddpmain function
	c.commentNode(c.cbb, d, d.Name.Literal)
	var varLocation value.Value
	if c.scp.enclosing == nil { // global scope
		varLocation = c.mod.NewGlobalDef("", getDefaultValue(d.Type.Type))
	} else {
		varLocation = c.cf.Blocks[0].NewAlloca(Typ)
	}
	Var := c.scp.addVar(d.Name.Literal, varLocation, Typ)
	initVal := c.evaluate(d.InitVal) // evaluate the initial value
	c.cbb.NewStore(initVal, Var)     // store the initial value
	if Typ == ddpstrptr {            // strings must be added to the ref-table
		c.incrementRC(initVal, VK_STRING) // ref_count becomes 1
	}
	return c
}
func (c *Compiler) VisitFuncDecl(d *ast.FuncDecl) ast.Visitor {
	retType := toDDPType(d.Type.Type)                 // get the llvm type
	params := make([]*ir.Param, 0, len(d.ParamTypes)) // list of the ir parameters

	// append all the other parameters
	for i, tok := range d.ParamTypes {
		ty := toDDPType(tok.Type)                                         // convert the type of the parameter
		params = append(params, ir.NewParam(d.ParamNames[i].Literal, ty)) // add it to the list
	}

	// append a prefix to every ir function to make it impossible for the user to break internal stuff
	name := d.Name.Literal
	if isInbuiltFunc(d) { // inbuilt/runtime functions are prefixed with inbuilt_
		name = "inbuilt_" + strings.TrimLeft(name, "§")
	} else { // user-defined functions are prefixed with ddpfunc_
		name = "ddpfunc_" + name
	}
	irFunc := c.mod.NewFunc(name, retType, params...) // create the ir function
	irFunc.CallingConv = enum.CallingConvC            // every function is called with the c calling convention to make interaction with inbuilt stuff easier

	c.insertFunction(d.Name.Literal, d, irFunc)

	if isInbuiltFunc(d) {
		irFunc.Linkage = enum.LinkageExternal // inbuilt functions are defined in c
	} else {
		fun, block := c.cf, c.cbb // safe the state before the function body
		c.cf, c.cbb, c.scp = irFunc, irFunc.NewBlock(""), newScope(c.scp)
		// passed arguments are immutible (llvm uses ssa registers) so we declare them as local variables
		// the caller of the function is responsible for managing the ref-count of garbage collected values
		for i := range params {
			if d.ParamTypes[i].Type == token.TEXT { // strings (and later other garbage collected types) need special handling
				// add the local variable for the parameter
				v := c.scp.addVar(params[i].LocalIdent.Name(), c.cbb.NewAlloca(ddpstrptr), ddpstrptr)
				// we need to deep copy the passed string because the caller
				// must call increment/decrement_ref_count on it
				strptr := c.deepCopyStr(params[i])
				c.incrementRC(strptr, VK_STRING) // increment-ref-count on the new local variable
				c.cbb.NewStore(strptr, v)        // store the copy in the local variable
			} else {
				// non garbage-collected types are just declared as their ir type
				v := c.scp.addVar(params[i].LocalIdent.Name(), c.cbb.NewAlloca(params[i].Type()), params[i].Type())
				c.cbb.NewStore(params[i], v)
			}
		}
		c.visitNode(d.Body) // compile the function body
		if c.cbb.Term == nil {
			c.cbb.NewRet(nil) // every block needs a terminator, and every function a return
		}
		c.cf, c.cbb, c.scp = fun, block, c.exitScope(c.scp) // restore state before the function (to main)
	}

	return c
}

// should have been filtered by the resolver/typechecker, so err
func (c *Compiler) VisitBadExpr(e *ast.BadExpr) ast.Visitor {
	err("Es wurde ein invalider Ausdruck gefunden")
	return c
}
func (c *Compiler) VisitIdent(e *ast.Ident) ast.Visitor {
	Var := c.scp.lookupVar(e.Literal.Literal) // get the alloca in the ir
	c.commentNode(c.cbb, e, e.Literal.Literal)
	if Var.typ == ddpstrptr { // strings must be copied in case the user of the expression modifies them
		c.latestReturn = c.deepCopyStr(c.cbb.NewLoad(Var.typ, Var.val))
	} else { // other variables are simply copied
		c.latestReturn = c.cbb.NewLoad(Var.typ, Var.val)
	}
	return c
}
func (c *Compiler) VisitIndexing(e *ast.Indexing) ast.Visitor {
	lhs := c.evaluate(e.Name)
	rhs := c.evaluate(e.Index)
	if lhs.Type() == ddpstrptr {
		c.incrementRC(lhs, VK_STRING)
	}
	if rhs.Type() == ddpstrptr {
		c.incrementRC(rhs, VK_STRING)
	}
	switch lhs.Type() {
	case ddpstrptr:
		switch rhs.Type() {
		case ddpint:
			c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_string_index"].irFunc, lhs, rhs)
		default:
			err(fmt.Sprintf("invalid Parameter Types for DURCH (%s, %s)", lhs.Type(), rhs.Type()))
		}
	default:
		err(fmt.Sprintf("invalid Parameter Types for STELLE (%s, %s)", lhs.Type(), rhs.Type()))
	}
	if lhs.Type() == ddpstrptr {
		c.decrementRC(lhs)
	}
	if rhs.Type() == ddpstrptr {
		c.decrementRC(rhs)
	}
	return c
}

// literals are simple ir constants
func (c *Compiler) VisitIntLit(e *ast.IntLit) ast.Visitor {
	c.commentNode(c.cbb, e, "")
	c.latestReturn = constant.NewInt(ddpint, e.Value)
	return c
}
func (c *Compiler) VisitFLoatLit(e *ast.FloatLit) ast.Visitor {
	c.commentNode(c.cbb, e, "")
	c.latestReturn = constant.NewFloat(ddpfloat, e.Value)
	return c
}
func (c *Compiler) VisitBoolLit(e *ast.BoolLit) ast.Visitor {
	c.commentNode(c.cbb, e, "")
	c.latestReturn = constant.NewBool(e.Value)
	return c
}
func (c *Compiler) VisitCharLit(e *ast.CharLit) ast.Visitor {
	c.commentNode(c.cbb, e, "")
	c.latestReturn = constant.NewInt(ddpchar, int64(e.Value))
	return c
}

// string literals are created by the ddp-c-runtime
// so we need to do some work here
func (c *Compiler) VisitStringLit(e *ast.StringLit) ast.Visitor {
	constStr := c.mod.NewGlobalDef("", irutil.NewCString(e.Value))
	// call the ddp-runtime function to create the ddpstring
	c.commentNode(c.cbb, e, constStr.Name())
	c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_string_from_constant"].irFunc, c.cbb.NewBitCast(constStr, ptr(i8)))
	return c
}
func (c *Compiler) VisitUnaryExpr(e *ast.UnaryExpr) ast.Visitor {
	rhs := c.evaluate(e.Rhs) // compile the expression onto which the operator is applied
	if rhs.Type() == ddpstrptr {
		c.incrementRC(rhs, VK_STRING)
	}
	// big switches for the different type combinations
	c.commentNode(c.cbb, e, e.Operator.String())
	switch e.Operator.Type {
	case token.BETRAG:
		switch rhs.Type() {
		case ddpfloat:
			c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_float_betrag"].irFunc, rhs)
		case ddpint:
			c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_int_betrag"].irFunc, rhs)
		default:
			err(fmt.Sprintf("invalid Parameter Type for BETRAG: %s", rhs.Type()))
		}
	case token.NEGATE:
		switch rhs.Type() {
		case ddpfloat:
			c.latestReturn = c.cbb.NewFNeg(rhs)
		case ddpint:
			c.latestReturn = c.cbb.NewSub(zero, rhs)
		default:
			err(fmt.Sprintf("invalid Parameter Type for NEGATE: %s", rhs.Type()))
		}
	case token.NICHT:
		c.latestReturn = c.cbb.NewXor(rhs, newInt(1))
	case token.NEGIERE:
		switch rhs.Type() {
		case ddpbool:
			c.latestReturn = c.cbb.NewXor(rhs, newInt(1))
		case ddpint:
			c.latestReturn = c.cbb.NewXor(rhs, newInt(all_ones))
		}
	case token.LOGISCHNICHT:
		c.latestReturn = c.cbb.NewXor(rhs, newInt(all_ones))
	case token.LÄNGE:
		switch rhs.Type() {
		case ddpstrptr:
			c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_string_length"].irFunc, rhs)
		default:
			err(fmt.Sprintf("invalid Parameter Type for LÄNGE: %s", rhs.Type()))
		}
	case token.GRÖßE:
		switch rhs.Type() {
		case ddpint, ddpfloat:
			c.latestReturn = newInt(8)
		case ddpbool:
			c.latestReturn = newInt(1)
		case ddpchar:
			c.latestReturn = newInt(4)
		case ddpstrptr:
			strcapptr := c.cbb.NewGetElementPtr(ddpstring, rhs, constant.NewInt(i32, 0), constant.NewInt(i32, 1))
			c.latestReturn = c.cbb.NewLoad(ddpint, strcapptr)
		default:
			err(fmt.Sprintf("invalid Parameter Type for GRÖßE: %s", rhs.Type()))
		}
	case token.SINUS:
		switch rhs.Type() {
		case ddpint:
			fp := c.cbb.NewSIToFP(rhs, ddpfloat)
			c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_sin"].irFunc, fp)
		case ddpfloat:
			c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_sin"].irFunc, rhs)
		default:
			err(fmt.Sprintf("invalid Parameter Type for SINUS: %s", rhs.Type()))
		}
	case token.KOSINUS:
		switch rhs.Type() {
		case ddpint:
			fp := c.cbb.NewSIToFP(rhs, ddpfloat)
			c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_cos"].irFunc, fp)
		case ddpfloat:
			c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_cos"].irFunc, rhs)
		default:
			err(fmt.Sprintf("invalid Parameter Type for KOSINUS: %s", rhs.Type()))
		}
	case token.TANGENS:
		switch rhs.Type() {
		case ddpint:
			fp := c.cbb.NewSIToFP(rhs, ddpfloat)
			c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_tan"].irFunc, fp)
		case ddpfloat:
			c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_tan"].irFunc, rhs)
		default:
			err(fmt.Sprintf("invalid Parameter Type for TANGENS: %s", rhs.Type()))
		}
	case token.ARKSIN:
		switch rhs.Type() {
		case ddpint:
			fp := c.cbb.NewSIToFP(rhs, ddpfloat)
			c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_asin"].irFunc, fp)
		case ddpfloat:
			c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_asin"].irFunc, rhs)
		default:
			err(fmt.Sprintf("invalid Parameter Type for ARKUSSINUS: %s", rhs.Type()))
		}
	case token.ARKKOS:
		switch rhs.Type() {
		case ddpint:
			fp := c.cbb.NewSIToFP(rhs, ddpfloat)
			c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_acos"].irFunc, fp)
		case ddpfloat:
			c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_acos"].irFunc, rhs)
		default:
			err(fmt.Sprintf("invalid Parameter Type for ARKUSKOSINUS: %s", rhs.Type()))
		}
	case token.ARKTAN:
		switch rhs.Type() {
		case ddpint:
			fp := c.cbb.NewSIToFP(rhs, ddpfloat)
			c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_atan"].irFunc, fp)
		case ddpfloat:
			c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_atan"].irFunc, rhs)
		default:
			err(fmt.Sprintf("invalid Parameter Type for ARKUSTANGENS: %s", rhs.Type()))
		}
	case token.HYPSIN:
		switch rhs.Type() {
		case ddpint:
			fp := c.cbb.NewSIToFP(rhs, ddpfloat)
			c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_sinh"].irFunc, fp)
		case ddpfloat:
			c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_sinh"].irFunc, rhs)
		default:
			err(fmt.Sprintf("invalid Parameter Type for HYPERBELSINUS: %s", rhs.Type()))
		}
	case token.HYPKOS:
		switch rhs.Type() {
		case ddpint:
			fp := c.cbb.NewSIToFP(rhs, ddpfloat)
			c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_cosh"].irFunc, fp)
		case ddpfloat:
			c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_cosh"].irFunc, rhs)
		default:
			err(fmt.Sprintf("invalid Parameter Type for HYPERBELKOSINUS: %s", rhs.Type()))
		}
	case token.HYPTAN:
		switch rhs.Type() {
		case ddpint:
			fp := c.cbb.NewSIToFP(rhs, ddpfloat)
			c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_tanh"].irFunc, fp)
		case ddpfloat:
			c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_tanh"].irFunc, rhs)
		default:
			err(fmt.Sprintf("invalid Parameter Type for HYPERBELTANGENS: %s", rhs.Type()))
		}
	case token.ZAHL:
		switch rhs.Type() {
		case ddpint:
			c.latestReturn = rhs
		case ddpfloat:
			c.latestReturn = c.cbb.NewFPToSI(rhs, ddpint)
		case ddpbool:
			cond := c.cbb.NewICmp(enum.IPredNE, rhs, zero)
			c.latestReturn = c.cbb.NewZExt(cond, ddpint)
		case ddpchar:
			c.latestReturn = c.cbb.NewSExt(rhs, ddpint)
		case ddpstrptr:
			c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_string_to_int"].irFunc, rhs)
		default:
			err(fmt.Sprintf("invalid Parameter Type for ZAHL: %s", rhs.Type()))
		}
	case token.KOMMAZAHL:
		switch rhs.Type() {
		case ddpint:
			c.latestReturn = c.cbb.NewSIToFP(rhs, ddpfloat)
		case ddpfloat:
			c.latestReturn = rhs
		case ddpstrptr:
			c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_string_to_float"].irFunc, rhs)
		default:
			err(fmt.Sprintf("invalid Parameter Type for KOMMAZAHL: %s", rhs.Type()))
		}
	case token.BOOLEAN:
		switch rhs.Type() {
		case ddpint:
			c.latestReturn = c.cbb.NewICmp(enum.IPredNE, rhs, zero)
		case ddpbool:
			c.latestReturn = rhs
		default:
			err(fmt.Sprintf("invalid Parameter Type for BOOLEAN: %s", rhs.Type()))
		}
	case token.BUCHSTABE:
		switch rhs.Type() {
		case ddpint:
			c.latestReturn = c.cbb.NewTrunc(rhs, ddpchar)
		case ddpchar:
			c.latestReturn = rhs
		default:
			err(fmt.Sprintf("invalid Parameter Type for BUCHSTABE: %s", rhs.Type()))
		}
	case token.TEXT:
		switch rhs.Type() {
		case ddpint:
			c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_int_to_string"].irFunc, rhs)
		case ddpfloat:
			c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_float_to_string"].irFunc, rhs)
		case ddpbool:
			c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_bool_to_string"].irFunc, rhs)
		case ddpchar:
			c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_char_to_string"].irFunc, rhs)
		case ddpstrptr:
			c.latestReturn = c.deepCopyStr(rhs)
		default:
			err(fmt.Sprintf("invalid Parameter Type for TEXT: %s", rhs.Type()))
		}
	default:
		err(fmt.Sprintf("Unbekannter Operator '%s'", e.Operator.String()))
	}
	if rhs.Type() == ddpstrptr {
		c.decrementRC(rhs)
	}
	return c
}
func (c *Compiler) VisitBinaryExpr(e *ast.BinaryExpr) ast.Visitor {
	switch e.Operator.Type {
	case token.UND:
		lhs := c.evaluate(e.Lhs)
		startBlock, trueBlock, leaveBlock := c.cbb, c.cf.NewBlock(""), c.cf.NewBlock("")
		c.commentNode(c.cbb, e, "")
		c.cbb.NewCondBr(lhs, trueBlock, leaveBlock)

		c.cbb = trueBlock
		rhs := c.evaluate(e.Rhs)
		c.commentNode(c.cbb, e, "")
		c.cbb.NewBr(leaveBlock)

		c.cbb = leaveBlock
		c.commentNode(c.cbb, e, "")
		c.latestReturn = c.cbb.NewPhi(ir.NewIncoming(rhs, trueBlock), ir.NewIncoming(lhs, startBlock))
		return c
	case token.ODER:
		lhs := c.evaluate(e.Lhs)
		startBlock, falseBlock, leaveBlock := c.cbb, c.cf.NewBlock(""), c.cf.NewBlock("")
		c.commentNode(c.cbb, e, "")
		c.cbb.NewCondBr(lhs, leaveBlock, falseBlock)

		c.cbb = falseBlock
		rhs := c.evaluate(e.Rhs)
		c.commentNode(c.cbb, e, "")
		c.cbb.NewBr(leaveBlock)

		c.cbb = leaveBlock
		c.commentNode(c.cbb, e, "")
		c.latestReturn = c.cbb.NewPhi(ir.NewIncoming(lhs, startBlock), ir.NewIncoming(rhs, falseBlock))
		return c
	}

	// compile the two expressions onto which the operator is applied
	lhs := c.evaluate(e.Lhs)
	rhs := c.evaluate(e.Rhs)
	if lhs.Type() == ddpstrptr {
		c.incrementRC(lhs, VK_STRING)
	}
	if rhs.Type() == ddpstrptr {
		c.incrementRC(rhs, VK_STRING)
	}
	// big switches on the different type combinations
	c.commentNode(c.cbb, e, e.Operator.String())
	switch e.Operator.Type {
	case token.VERKETTET:
		switch lhs.Type() {
		case ddpstrptr:
			switch rhs.Type() {
			case ddpstrptr:
				c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_string_string_verkettet"].irFunc, lhs, rhs)
			case ddpchar:
				c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_string_char_verkettet"].irFunc, lhs, rhs)
			default:
				err(fmt.Sprintf("invalid Parameter Types for VERKETTET (%s, %s)", lhs.Type(), rhs.Type()))
			}
		case ddpchar:
			switch rhs.Type() {
			case ddpstrptr:
				c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_char_string_verkettet"].irFunc, lhs, rhs)
			case ddpchar:
				c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_char_char_verkettet"].irFunc, lhs, rhs)
			default:
				err(fmt.Sprintf("invalid Parameter Types for VERKETTET (%s, %s)", lhs.Type(), rhs.Type()))
			}
		default:
			err(fmt.Sprintf("invalid Parameter Types for VERKETTET (%s, %s)", lhs.Type(), rhs.Type()))
		}
	case token.PLUS, token.ADDIERE, token.ERHÖHE:
		switch lhs.Type() {
		case ddpint:
			switch rhs.Type() {
			case ddpint:
				c.latestReturn = c.cbb.NewAdd(lhs, rhs)
			case ddpfloat:
				fp := c.cbb.NewSIToFP(lhs, ddpfloat)
				c.latestReturn = c.cbb.NewFAdd(fp, rhs)
			default:
				err(fmt.Sprintf("invalid Parameter Types for PLUS (%s, %s)", lhs.Type(), rhs.Type()))
			}
		case ddpfloat:
			switch rhs.Type() {
			case ddpint:
				fp := c.cbb.NewSIToFP(rhs, ddpfloat)
				c.latestReturn = c.cbb.NewFAdd(lhs, fp)
			case ddpfloat:
				c.latestReturn = c.cbb.NewFAdd(lhs, rhs)
			default:
				err(fmt.Sprintf("invalid Parameter Types for PLUS (%s, %s)", lhs.Type(), rhs.Type()))
			}
		default:
			err(fmt.Sprintf("invalid Parameter Types for PLUS (%s, %s)", lhs.Type(), rhs.Type()))
		}
	case token.MINUS, token.SUBTRAHIERE, token.VERRINGERE:
		switch lhs.Type() {
		case ddpint:
			switch rhs.Type() {
			case ddpint:
				c.latestReturn = c.cbb.NewSub(lhs, rhs)
			case ddpfloat:
				fp := c.cbb.NewSIToFP(lhs, ddpfloat)
				c.latestReturn = c.cbb.NewFSub(fp, rhs)
			default:
				err(fmt.Sprintf("invalid Parameter Types for MINUS (%s, %s)", lhs.Type(), rhs.Type()))
			}
		case ddpfloat:
			switch rhs.Type() {
			case ddpint:
				fp := c.cbb.NewSIToFP(rhs, ddpfloat)
				c.latestReturn = c.cbb.NewFSub(lhs, fp)
			case ddpfloat:
				c.latestReturn = c.cbb.NewFSub(lhs, rhs)
			default:
				err(fmt.Sprintf("invalid Parameter Types for MINUS (%s, %s)", lhs.Type(), rhs.Type()))
			}
		default:
			err(fmt.Sprintf("invalid Parameter Types for MINUS (%s, %s)", lhs.Type(), rhs.Type()))
		}
	case token.MAL, token.MULTIPLIZIERE, token.VERVIELFACHE:
		switch lhs.Type() {
		case ddpint:
			switch rhs.Type() {
			case ddpint:
				c.latestReturn = c.cbb.NewMul(lhs, rhs)
			case ddpfloat:
				fp := c.cbb.NewSIToFP(lhs, ddpfloat)
				c.latestReturn = c.cbb.NewFMul(fp, rhs)
			default:
				err(fmt.Sprintf("invalid Parameter Types for MAL (%s, %s)", lhs.Type(), rhs.Type()))
			}
		case ddpfloat:
			switch rhs.Type() {
			case ddpint:
				fp := c.cbb.NewSIToFP(rhs, ddpfloat)
				c.latestReturn = c.cbb.NewFMul(lhs, fp)
			case ddpfloat:
				c.latestReturn = c.cbb.NewFMul(lhs, rhs)
			default:
				err(fmt.Sprintf("invalid Parameter Types for MAL (%s, %s)", lhs.Type(), rhs.Type()))
			}
		default:
			err(fmt.Sprintf("invalid Parameter Types for MAL (%s, %s)", lhs.Type(), rhs.Type()))
		}
	case token.DURCH, token.DIVIDIERE, token.TEILE:
		switch lhs.Type() {
		case ddpint:
			switch rhs.Type() {
			case ddpint:
				lhs = c.cbb.NewSIToFP(lhs, ddpfloat)
				rhs = c.cbb.NewSIToFP(rhs, ddpfloat)
				c.latestReturn = c.cbb.NewFDiv(lhs, rhs)
			case ddpfloat:
				fp := c.cbb.NewSIToFP(lhs, ddpfloat)
				c.latestReturn = c.cbb.NewFDiv(fp, rhs)
			default:
				err(fmt.Sprintf("invalid Parameter Types for DURCH (%s, %s)", lhs.Type(), rhs.Type()))
			}
		case ddpfloat:
			switch rhs.Type() {
			case ddpint:
				fp := c.cbb.NewSIToFP(rhs, ddpfloat)
				c.latestReturn = c.cbb.NewFDiv(lhs, fp)
			case ddpfloat:
				c.latestReturn = c.cbb.NewFDiv(lhs, rhs)
			default:
				err(fmt.Sprintf("invalid Parameter Types for DURCH (%s, %s)", lhs.Type(), rhs.Type()))
			}
		default:
			err(fmt.Sprintf("invalid Parameter Types for DURCH (%s, %s)", lhs.Type(), rhs.Type()))
		}
	case token.STELLE:
		switch lhs.Type() {
		case ddpstrptr:
			switch rhs.Type() {
			case ddpint:
				c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_string_index"].irFunc, lhs, rhs)
			default:
				err(fmt.Sprintf("invalid Parameter Types for DURCH (%s, %s)", lhs.Type(), rhs.Type()))
			}
		default:
			err(fmt.Sprintf("invalid Parameter Types for STELLE (%s, %s)", lhs.Type(), rhs.Type()))
		}
	case token.HOCH:
		switch lhs.Type() {
		case ddpint:
			switch rhs.Type() {
			case ddpint:
				lhs = c.cbb.NewSIToFP(lhs, ddpfloat)
				rhs = c.cbb.NewSIToFP(rhs, ddpfloat)
			case ddpfloat:
				lhs = c.cbb.NewSIToFP(lhs, ddpfloat)
			default:
				err(fmt.Sprintf("invalid Parameter Types for HOCH (%s, %s)", lhs.Type(), rhs.Type()))
			}
		case ddpfloat:
			switch rhs.Type() {
			case ddpint:
				rhs = c.cbb.NewSIToFP(rhs, ddpfloat)
			default:
				err(fmt.Sprintf("invalid Parameter Types for HOCH (%s, %s)", lhs.Type(), rhs.Type()))
			}
		default:
			err(fmt.Sprintf("invalid Parameter Types for HOCH (%s, %s)", lhs.Type(), rhs.Type()))
		}
		c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_hoch"].irFunc, lhs, rhs)
	case token.LOGARITHMUS:
		switch lhs.Type() {
		case ddpint:
			switch rhs.Type() {
			case ddpint:
				lhs = c.cbb.NewSIToFP(lhs, ddpfloat)
				rhs = c.cbb.NewSIToFP(rhs, ddpfloat)
			case ddpfloat:
				lhs = c.cbb.NewSIToFP(lhs, ddpfloat)
			default:
				err(fmt.Sprintf("invalid Parameter Types for LOGARITHMUS (%s, %s)", lhs.Type(), rhs.Type()))
			}
		case ddpfloat:
			switch rhs.Type() {
			case ddpint:
				rhs = c.cbb.NewSIToFP(rhs, ddpfloat)
			default:
				err(fmt.Sprintf("invalid Parameter Types for LOGARITHMUS (%s, %s)", lhs.Type(), rhs.Type()))
			}
		default:
			err(fmt.Sprintf("invalid Parameter Types for LOGARITHMUS (%s, %s)", lhs.Type(), rhs.Type()))
		}
		c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_log"].irFunc, lhs, rhs)
	case token.LOGISCHUND:
		c.latestReturn = c.cbb.NewAnd(lhs, rhs)
	case token.LOGISCHODER:
		c.latestReturn = c.cbb.NewOr(lhs, rhs)
	case token.KONTRA:
		c.latestReturn = c.cbb.NewXor(lhs, rhs)
	case token.MODULO:
		c.latestReturn = c.cbb.NewSRem(lhs, rhs)
	case token.LINKS:
		c.latestReturn = c.cbb.NewShl(lhs, rhs)
	case token.RECHTS:
		c.latestReturn = c.cbb.NewLShr(lhs, rhs)
	case token.GLEICH:
		switch lhs.Type() {
		case ddpint:
			c.latestReturn = c.cbb.NewICmp(enum.IPredEQ, lhs, rhs)
		case ddpfloat:
			c.latestReturn = c.cbb.NewFCmp(enum.FPredOEQ, lhs, rhs)
		case ddpbool:
			c.latestReturn = c.cbb.NewICmp(enum.IPredEQ, lhs, rhs)
		case ddpchar:
			c.latestReturn = c.cbb.NewICmp(enum.IPredEQ, lhs, rhs)
		case ddpstrptr:
			c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_string_equal"].irFunc, lhs, rhs)
		default:
			err(fmt.Sprintf("invalid Parameter Types for GLEICH (%s, %s)", lhs.Type(), rhs.Type()))
		}
	case token.UNGLEICH:
		switch lhs.Type() {
		case ddpint:
			c.latestReturn = c.cbb.NewICmp(enum.IPredNE, lhs, rhs)
		case ddpfloat:
			c.latestReturn = c.cbb.NewFCmp(enum.FPredONE, lhs, rhs)
		case ddpbool:
			c.latestReturn = c.cbb.NewICmp(enum.IPredNE, lhs, rhs)
		case ddpchar:
			c.latestReturn = c.cbb.NewICmp(enum.IPredNE, lhs, rhs)
		case ddpstrptr:
			equal := c.cbb.NewCall(c.functions["inbuilt_string_equal"].irFunc, lhs, rhs)
			c.latestReturn = c.cbb.NewXor(equal, newInt(1))
		default:
			err(fmt.Sprintf("invalid Parameter Types for UNGLEICH (%s, %s)", lhs.Type(), rhs.Type()))
		}
	case token.KLEINER:
		switch lhs.Type() {
		case ddpint:
			switch rhs.Type() {
			case ddpint:
				c.latestReturn = c.cbb.NewICmp(enum.IPredSLT, lhs, rhs)
			case ddpfloat:
				fp := c.cbb.NewSIToFP(lhs, ddpfloat)
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOLT, fp, rhs)
			default:
				err(fmt.Sprintf("invalid Parameter Types for KLEINER (%s, %s)", lhs.Type(), rhs.Type()))
			}
		case ddpfloat:
			switch rhs.Type() {
			case ddpint:
				fp := c.cbb.NewSIToFP(rhs, ddpfloat)
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOLT, lhs, fp)
			case ddpfloat:
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOLT, lhs, rhs)
			default:
				err(fmt.Sprintf("invalid Parameter Types for KLEINER (%s, %s)", lhs.Type(), rhs.Type()))
			}
		}
	case token.KLEINERODER:
		switch lhs.Type() {
		case ddpint:
			switch rhs.Type() {
			case ddpint:
				c.latestReturn = c.cbb.NewICmp(enum.IPredSLE, lhs, rhs)
			case ddpfloat:
				fp := c.cbb.NewSIToFP(lhs, ddpfloat)
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOLE, fp, rhs)
			default:
				err(fmt.Sprintf("invalid Parameter Types for KLEINERODER (%s, %s)", lhs.Type(), rhs.Type()))
			}
		case ddpfloat:
			switch rhs.Type() {
			case ddpint:
				fp := c.cbb.NewSIToFP(rhs, ddpfloat)
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOLE, lhs, fp)
			case ddpfloat:
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOLE, lhs, rhs)
			default:
				err(fmt.Sprintf("invalid Parameter Types for KLEINERODER (%s, %s)", lhs.Type(), rhs.Type()))
			}
		default:
			err(fmt.Sprintf("invalid Parameter Types for KLEINERODER (%s, %s)", lhs.Type(), rhs.Type()))
		}
	case token.GRÖßER:
		switch lhs.Type() {
		case ddpint:
			switch rhs.Type() {
			case ddpint:
				c.latestReturn = c.cbb.NewICmp(enum.IPredSGT, lhs, rhs)
			case ddpfloat:
				fp := c.cbb.NewSIToFP(lhs, ddpfloat)
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOGT, fp, rhs)
			default:
				err(fmt.Sprintf("invalid Parameter Types for GRÖßER (%s, %s)", lhs.Type(), rhs.Type()))
			}
		case ddpfloat:
			switch rhs.Type() {
			case ddpint:
				fp := c.cbb.NewSIToFP(rhs, ddpfloat)
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOGT, lhs, fp)
			case ddpfloat:
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOGT, lhs, rhs)
			default:
				err(fmt.Sprintf("invalid Parameter Types for GRÖßER (%s, %s)", lhs.Type(), rhs.Type()))
			}
		default:
			err(fmt.Sprintf("invalid Parameter Types for GRÖßER (%s, %s)", lhs.Type(), rhs.Type()))
		}
	case token.GRÖßERODER:
		switch lhs.Type() {
		case ddpint:
			switch rhs.Type() {
			case ddpint:
				c.latestReturn = c.cbb.NewICmp(enum.IPredSGE, lhs, rhs)
			case ddpfloat:
				fp := c.cbb.NewSIToFP(lhs, ddpfloat)
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOGE, fp, rhs)
			default:
				err(fmt.Sprintf("invalid Parameter Types for GRÖßERODER (%s, %s)", lhs.Type(), rhs.Type()))
			}
		case ddpfloat:
			switch rhs.Type() {
			case ddpint:
				fp := c.cbb.NewSIToFP(rhs, ddpfloat)
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOGE, lhs, fp)
			case ddpfloat:
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOGE, lhs, rhs)
			default:
				err(fmt.Sprintf("invalid Parameter Types for GRÖßERODER (%s, %s)", lhs.Type(), rhs.Type()))
			}
		default:
			err(fmt.Sprintf("invalid Parameter Types for GRÖßERODER (%s, %s)", lhs.Type(), rhs.Type()))
		}
	}
	if lhs.Type() == ddpstrptr {
		c.decrementRC(lhs)
	}
	if rhs.Type() == ddpstrptr {
		c.decrementRC(rhs)
	}
	return c
}
func (c *Compiler) VisitTernaryExpr(e *ast.TernaryExpr) ast.Visitor {
	lhs := c.evaluate(e.Lhs)
	mid := c.evaluate(e.Mid)
	rhs := c.evaluate(e.Rhs)
	if lhs.Type() == ddpstrptr {
		c.incrementRC(lhs, VK_STRING)
	}
	if mid.Type() == ddpstrptr {
		c.incrementRC(mid, VK_STRING)
	}
	if rhs.Type() == ddpstrptr {
		c.incrementRC(rhs, VK_STRING)
	}

	switch e.Operator.Type {
	case token.VONBIS:
		c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_string_slice"].irFunc, lhs, mid, rhs)
	default:
		err(fmt.Sprintf("invalid Parameter Types for VONBIS (%s, %s, %s)", lhs.Type(), mid.Type(), rhs.Type()))
	}

	if lhs.Type() == ddpstrptr {
		c.decrementRC(lhs)
	}
	if mid.Type() == ddpstrptr {
		c.decrementRC(mid)
	}
	if rhs.Type() == ddpstrptr {
		c.decrementRC(rhs)
	}
	return c
}
func (c *Compiler) VisitGrouping(e *ast.Grouping) ast.Visitor {
	return e.Expr.Accept(c) // visit like a normal expression, grouping is just precedence stuff which has already been parsed
}
func (c *Compiler) VisitFuncCall(e *ast.FuncCall) ast.Visitor {
	fun := c.functions[e.Name] // retreive the function (the resolver took care that it is present)
	args := make([]value.Value, 0, len(fun.funcDecl.ParamNames))

	// the caller of a function is responsible for managing the ref-count
	// of passed garbage collected values, so we call increment_ref_count on any
	// passed garbage collected argument
	toBeFreed := make([]*value.Value, 0)
	for _, param := range fun.funcDecl.ParamNames {
		val := c.evaluate(e.Args[param.Literal]) // compile each argument for the function
		// strings ref-count is managed by the caller, so increment it
		// and mark it toBeFreed for after the function returns
		if val.Type() == ddpstrptr {
			c.incrementRC(val, VK_STRING)
			toBeFreed = append(toBeFreed, &val)
		}
		args = append(args, val) // add the value to the arguments
	}

	c.commentNode(c.cbb, e, "")
	c.latestReturn = c.cbb.NewCall(fun.irFunc, args...) // compile the actual function call
	// after the function returns, we decrement the ref-count on
	// all passed arguments
	for i := range toBeFreed {
		c.decrementRC(*toBeFreed[i])
	}
	return c
}

// should have been filtered by the resolver/typechecker, so err
func (c *Compiler) VisitBadStmt(s *ast.BadStmt) ast.Visitor {
	err("Es wurde eine invalide Aussage gefunden")
	return c
}
func (c *Compiler) VisitDeclStmt(s *ast.DeclStmt) ast.Visitor {
	return s.Decl.Accept(c)
}
func (c *Compiler) VisitExprStmt(s *ast.ExprStmt) ast.Visitor {
	expr := c.evaluate(s.Expr)
	// free garbage collected returns
	if expr.Type() == ddpstrptr {
		c.incrementRC(expr, VK_STRING) // add it to the table (will be made better later)
		c.decrementRC(expr)
	}
	return c
}
func (c *Compiler) VisitAssignStmt(s *ast.AssignStmt) ast.Visitor {
	val := c.evaluate(s.Rhs) // compile the expression
	// intermediate values ref-counts must be incremented/decremented
	if val.Type() == ddpstrptr {
		c.incrementRC(val, VK_STRING)
	}
	switch assign := s.Var.(type) {
	case *ast.Ident:
		Var := c.scp.lookupVar(assign.Literal.Literal) // get the variable
		// free the value which was previously contained in the variable
		if Var.typ == ddpstrptr {
			c.decrementRC(c.cbb.NewLoad(Var.typ, Var.val))
		}
		c.commentNode(c.cbb, s, assign.Literal.Literal)
		c.cbb.NewStore(val, Var.val) // store the new value
	case *ast.Indexing:
		Var := c.scp.lookupVar(assign.Name.Literal.Literal) // get the variable
		index := c.evaluate(assign.Index)
		switch Var.typ {
		case ddpstrptr:
			c.commentNode(c.cbb, s, assign.Name.Literal.Literal)
			c.cbb.NewCall(c.functions["inbuilt_replace_char_in_string"].irFunc, c.cbb.NewLoad(ddpstrptr, Var.val), val, index)
		}
	}
	return c
}
func (c *Compiler) VisitBlockStmt(s *ast.BlockStmt) ast.Visitor {
	c.scp = newScope(c.scp) // a block gets its own scope
	for _, stmt := range s.Statements {
		c.visitNode(stmt)
	}

	c.scp = c.exitScope(c.scp) // free local variables and return to the previous scope
	return c
}

// TODO: test ifs, they may be buggy as hell
// for info on how the generated ir works you might want to see https://llir.github.io/document/user-guide/control/#If
func (c *Compiler) VisitIfStmt(s *ast.IfStmt) ast.Visitor {
	var handleIf func(*ast.IfStmt, *ir.Block) // declaration to use it recursively for nested else-ifs
	// inner function to handle if-else blocks (we need to pass the leaveBlock down the chain)
	handleIf = func(s *ast.IfStmt, leaveBlock *ir.Block) {
		cbb := c.cbb // saved for later

		// compile the thenBlock
		// we compile it first and safe it to jump to it later
		thenBlock := c.cf.NewBlock("")
		c.comment("thenBlock", thenBlock)
		c.cbb, c.scp = thenBlock, newScope(c.scp) // with its own scope
		c.visitNode(s.Then)
		c.cbb, c.scp = cbb, c.exitScope(c.scp) // revert the scope and block

		if s.Else != nil { // handle else and possible else-ifs
			elseBlock := c.cf.NewBlock("")
			c.comment("elseBlock", elseBlock)
			if leaveBlock == nil { // if we don't have a leaveBlock, make one
				leaveBlock = c.cf.NewBlock("")
				c.comment("leaveBlock", leaveBlock)
			}
			c.cbb, c.scp = elseBlock, newScope(c.scp) // the else block has its own scope as well
			// either execute the else statement or handle the else-if with our leaveBlock
			if elseIf, ok := s.Else.(*ast.IfStmt); ok {
				handleIf(elseIf, leaveBlock) // recursively handle if-else statements (wenn aber)
			} else {
				c.visitNode(s.Else) // handle only the else
			}

			if c.cbb != leaveBlock && c.cbb.Term == nil { // I don't understand this, maybe you do
				c.commentNode(c.cbb, s, "")
				c.cbb.NewBr(leaveBlock)
			}
			c.cbb, c.scp = cbb, c.exitScope(c.scp) // exit the else scope and restore the block before the if
			c.commentNode(c.cbb, s, "")
			c.cbb.NewCondBr(c.evaluate(s.Condition), thenBlock, elseBlock) // jump with the condition

			// add a terminator
			if thenBlock.Term == nil {
				c.commentNode(c.cbb, s, "")
				thenBlock.NewBr(leaveBlock)
			}
			if elseBlock.Term == nil {
				c.commentNode(c.cbb, s, "")
				elseBlock.NewBr(leaveBlock)
			}

			c.cbb = leaveBlock // continue compilation in the leave block
		} else { // if there is no else we just conditionally execute the then block
			if leaveBlock == nil { // if we don't already have a leaveBlock make one
				leaveBlock = c.cf.NewBlock("")
				c.comment("leaveBlock", leaveBlock)
			}
			// no else, so jump to then or leave
			c.commentNode(c.cbb, s, "")
			c.cbb.NewCondBr(c.evaluate(s.Condition), thenBlock, leaveBlock)
			// we need a terminator (simply jump after the then block)
			if thenBlock.Term == nil {
				c.commentNode(c.cbb, s, "")
				thenBlock.NewBr(leaveBlock)
			}
			c.cbb = leaveBlock // continue compilation in the leave block
		}
	}

	handleIf(s, nil) // begin the recursive compilation of the if statement
	return c
}

// for info on how the generated ir works you might want to see https://llir.github.io/document/user-guide/control/#Loop
func (c *Compiler) VisitWhileStmt(s *ast.WhileStmt) ast.Visitor {
	switch op := s.While.Type; op {
	case token.SOLANGE, token.MACHE:
		condBlock := c.cf.NewBlock("")
		body, bodyScope := c.cf.NewBlock(""), newScope(c.scp)

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

		leaveBlock := c.cf.NewBlock("")
		c.cbb, c.scp = condBlock, c.exitScope(c.scp) // the condition is not in scope
		c.commentNode(c.cbb, s, "")
		condBlock.NewCondBr(c.evaluate(s.Condition), body, leaveBlock)

		c.cbb = leaveBlock
	case token.MAL:
		counter := c.cf.Blocks[0].NewAlloca(ddpint)
		c.cbb.NewStore(c.evaluate(s.Condition), counter)
		condBlock := c.cf.NewBlock("")
		body, bodyScope := c.cf.NewBlock(""), newScope(c.scp)

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

		c.cbb = leaveBlock
	}
	return c
}

// for info on how the generated ir works you might want to see https://llir.github.io/document/user-guide/control/#Loop
func (c *Compiler) VisitForStmt(s *ast.ForStmt) ast.Visitor {
	c.scp = newScope(c.scp)     // scope for the for body
	c.visitNode(s.Initializer)  // compile the counter variable declaration
	initValue := c.latestReturn // safe the initial value of the counter to check for less or greater then

	condBlock := c.cf.NewBlock("")
	c.comment("condBlock", condBlock)
	incrementBlock := c.cf.NewBlock("")
	c.comment("incrementBlock", incrementBlock)
	forBody := c.cf.NewBlock("")
	c.comment("forBody", forBody)

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
	Var := c.scp.lookupVar(s.Initializer.Name.Literal)
	indexVar := incrementBlock.NewLoad(Var.typ, Var.val)
	var incrementer value.Value // Schrittgröße
	// if no stepsize was present it is 1
	if s.StepSize == nil {
		incrementer = constant.NewInt(ddpint, 1)
	} else { // stepsize was present, so compile it
		c.cbb = incrementBlock
		incrementer = c.evaluate(s.StepSize)
	}
	// add the incrementer to the counter variable
	add := incrementBlock.NewAdd(indexVar, incrementer)
	incrementBlock.NewStore(add, c.scp.lookupVar(s.Initializer.Name.Literal).val)
	c.commentNode(incrementBlock, s, "")
	incrementBlock.NewBr(condBlock) // check the condition (loop)

	// finally compile the condition block(s)
	initGreaterTo := c.cf.NewBlock("")
	c.comment("initGreaterTo", initGreaterTo)
	initLessthenTo := c.cf.NewBlock("")
	c.comment("initLessthenTo", initLessthenTo)
	leaveBlock := c.cf.NewBlock("") // after the condition is false we jump to the leaveBlock
	c.comment("forLeaveBlock", leaveBlock)

	c.cbb = condBlock
	// we check the counter differently depending on wether or not we are looping up or down (positive vs negative stepsize)
	cond := condBlock.NewICmp(enum.IPredSLE, initValue, c.evaluate(s.To))
	c.commentNode(condBlock, s, "")
	condBlock.NewCondBr(cond, initLessthenTo, initGreaterTo)

	// we are counting up, so compare less-or-equal
	cond = initLessthenTo.NewICmp(enum.IPredSLE, initLessthenTo.NewLoad(Var.typ, Var.val), c.evaluate(s.To))
	c.commentNode(initLessthenTo, s, "")
	initLessthenTo.NewCondBr(cond, forBody, leaveBlock)

	// we are counting down, so compare greater-or-equal
	cond = initGreaterTo.NewICmp(enum.IPredSGE, initGreaterTo.NewLoad(Var.typ, Var.val), c.evaluate(s.To))
	c.commentNode(initGreaterTo, s, "")
	initGreaterTo.NewCondBr(cond, forBody, leaveBlock)

	c.cbb, c.scp = leaveBlock, c.exitScope(c.scp) // leave the scopee
	return c
}
func (c *Compiler) VisitForRangeStmt(s *ast.ForRangeStmt) ast.Visitor {
	c.scp = newScope(c.scp)
	in := c.evaluate(s.In)
	c.incrementRC(in, VK_STRING)

	strlen := c.cbb.NewCall(c.functions["inbuilt_string_length"].irFunc, in)
	loopStart, condBlock, bodyBlock, incrementBlock, leaveBlock := c.cf.NewBlock(""), c.cf.NewBlock(""), c.cf.NewBlock(""), c.cf.NewBlock(""), c.cf.NewBlock("")
	c.cbb.NewCondBr(c.cbb.NewICmp(enum.IPredEQ, strlen, zero), leaveBlock, loopStart)

	c.cbb = loopStart
	index := c.cf.Blocks[0].NewAlloca(ddpint)
	c.cbb.NewStore(newInt(1), index)
	c.scp.addVar(s.Initializer.Name.Literal, c.cf.Blocks[0].NewAlloca(ddpchar), ddpchar)
	c.cbb.NewBr(condBlock)

	c.cbb = condBlock
	c.cbb.NewCondBr(c.cbb.NewICmp(enum.IPredSLE, c.cbb.NewLoad(ddpint, index), strlen), bodyBlock, leaveBlock)

	c.cbb = bodyBlock
	c.cbb.NewStore(c.cbb.NewCall(c.functions["inbuilt_string_index"].irFunc, in, c.cbb.NewLoad(ddpint, index)), c.scp.lookupVar(s.Initializer.Name.Literal).val)
	c.visitNode(s.Body)
	if c.cbb.Term == nil {
		c.cbb.NewBr(incrementBlock)
	}

	c.cbb = incrementBlock
	c.cbb.NewStore(c.cbb.NewAdd(c.cbb.NewLoad(ddpint, index), newInt(1)), index)
	c.cbb.NewBr(condBlock)

	c.cbb, c.scp = leaveBlock, c.exitScope(c.scp)
	c.decrementRC(in)
	return c
}
func (c *Compiler) VisitFuncCallStmt(s *ast.FuncCallStmt) ast.Visitor {
	return s.Call.Accept(c)
}
func (c *Compiler) VisitReturnStmt(s *ast.ReturnStmt) ast.Visitor {
	ret := c.evaluate(s.Value)   // compile the return value
	if ret.Type() == ddpstrptr { // strings need to be copied and memory-managed
		oldRet := ret
		c.incrementRC(oldRet, VK_STRING)
		ret = c.deepCopyStr(oldRet)
		c.decrementRC(oldRet)
	}
	c.commentNode(c.cbb, s, "")
	c.cbb.NewRet(ret)
	return c
}

// helper functions

func notimplemented() {
	file, line, function := getTraceInfo(2)
	panic(fmt.Errorf("%s, %d, %s: this function or a part of it is not implemented", filepath.Base(file), line, function))
}

func getTraceInfo(skip int) (file string, line int, function string) {
	pc := make([]uintptr, 15)
	n := runtime.Callers(skip+1, pc)
	frames := runtime.CallersFrames(pc[:n])
	frame, _ := frames.Next()
	return frame.File, frame.Line, frame.Function
}

func getDataLayout() string {
	llvm.InitializeAllTargetInfos()
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllAsmParsers()
	llvm.InitializeAllAsmPrinters()

	target, err := llvm.GetTargetFromTriple(llvm.DefaultTargetTriple())
	if err != nil {
		return ""
	}
	targetMachine := target.CreateTargetMachine(
		llvm.DefaultTargetTriple(),
		"generic",
		"",
		llvm.CodeGenOptLevel(llvm.CodeGenLevelDefault),
		llvm.RelocMode(llvm.RelocDefault),
		llvm.CodeModel(llvm.CodeModelDefault),
	)
	defer targetMachine.Dispose()
	targetData := targetMachine.CreateTargetData()
	defer targetData.Dispose()
	return targetData.String()
}
