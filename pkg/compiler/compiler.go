package compiler

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf8"

	"github.com/DDP-Projekt/Kompilierer/pkg/ast"
	"github.com/DDP-Projekt/Kompilierer/pkg/scanner"
	"github.com/DDP-Projekt/Kompilierer/pkg/token"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
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
		errorHandler = func(string) {}
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
func (c *Compiler) Compile() (result string, rerr error) {
	// catch panics and instead set the returned error
	defer func() {
		if err := recover(); err != nil {
			rerr = err.(error)
			result = ""
		}
	}()

	// the ast must be valid (and should have been resolved and typechecked beforehand)
	if c.ast.Faulty {
		return "", fmt.Errorf("Fehlerhafter Syntax Baum")
	}

	c.mod.SourceFilename = c.ast.File // set the module filename (optional metadata)
	c.setupStringType()               // setup some internal functions to work with strings
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

	// on success ddpmain returns 0
	c.cbb.NewRet(newInt(0))
	return c.mod.String(), nil // return the module as string
}

// helper that might be extended later
// it is not intended for the end user to see these errors, as they are compiler bugs
// the errors in the ddp-code were already reported by the parser/typechecker/resolver
func err(msg string) {
	// retreive the file and line on which the error occured
	_, file, line, _ := runtime.Caller(1)
	panic(fmt.Errorf("%s, %d: %s", filepath.Base(file), line, msg))
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

// declares some internal string functions
// and completes the ddpstring struct
func (c *Compiler) setupStringType() {
	// complete the ddpstring definition to interact with the c ddp runtime
	ddpstring.Fields = make([]types.Type, 2)
	ddpstring.Fields[0] = ptr(ddpchar)
	ddpstring.Fields[1] = ddpint
	c.mod.NewTypeDef("ddpstring", ddpstring)

	// declare all the external functions to work with strings

	// creates a ddpstring from a string literal
	sfc := c.mod.NewFunc("inbuilt_string_from_constant", ddpstrptr, ir.NewParam("str", ptr(ddpchar)), ir.NewParam("len", ddpint))
	sfc.CallingConv = enum.CallingConvC
	sfc.Linkage = enum.LinkageExternal
	c.insertFunction("inbuilt_string_from_constant", nil, sfc)

	// returns a copy of the passed string as a new pointer
	dcs := c.mod.NewFunc("inbuilt_deep_copy_string", ddpstrptr, ir.NewParam("str", ddpstrptr))
	dcs.CallingConv = enum.CallingConvC
	dcs.Linkage = enum.LinkageExternal
	c.insertFunction("inbuilt_deep_copy_string", nil, dcs)

	// decrement the ref-count on a pointer and
	// free the pointer if the ref-count becomes 0
	// takes the pointer and the type to which it points
	// (currently only string, but later lists and structs too)
	drc := c.mod.NewFunc("inbuilt_decrement_ref_count", void, ir.NewParam("key", ptr(i8)))
	drc.CallingConv = enum.CallingConvC
	drc.Linkage = enum.LinkageExternal
	c.insertFunction("inbuilt_decrement_ref_count", nil, drc)

	// increment the ref-count on a pointer
	// takes the pointer and the type to which it points
	// (currently only string, but later lists and structs too)
	irc := c.mod.NewFunc("inbuilt_increment_ref_count", void, ir.NewParam("key", ptr(i8)), ir.NewParam("kind", i8))
	irc.CallingConv = enum.CallingConvC
	irc.Linkage = enum.LinkageExternal
	c.insertFunction("inbuilt_increment_ref_count", nil, irc)
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
		if v.t == ddpstrptr {
			c.decrementRC(c.cbb.NewLoad(ddpstrptr, v.v))
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
	t := toDDPType(d.Type.Type) // get the llvm type
	// allocate the variable on the function call frame
	// all local variables are allocated in the first basic block of the function they are within
	// in the ir a local variable is a alloca instruction (a stack allocation)
	// global variables are allocated in the ddpmain function
	v := c.scp.addVar(d.Name.Literal, c.cf.Blocks[0].NewAlloca(t), t)
	initVal := c.evaluate(d.InitVal) // evaluate the initial value
	c.cbb.NewStore(initVal, v)       // store the initial value
	if t == ddpstrptr {              // strings must be added to the ref-table
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
		for i := range params {
			if d.ParamTypes[i].Type == token.TEXT { // strings (and later other garbage collected types) need special handling
				c.incrementRC(params[i], VK_STRING) // do we need that?
				// add the local variable for the parameter
				v := c.scp.addVar(params[i].LocalIdent.Name(), c.cbb.NewAlloca(ddpstrptr), ddpstrptr)
				strptr := c.deepCopyStr(params[i]) // deep copy the passed pointer to string
				c.incrementRC(strptr, VK_STRING)   // increment-ref-count on the new local variable
				c.cbb.NewStore(strptr, v)          // store the copy in the local variable
				c.decrementRC(params[i])           // do we need that?
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
	v := c.scp.lookupVar(e.Literal.Literal) // get the alloa in the ir
	if v.t == ddpstrptr {                   // strings must be copied in case the user of the expression modifies them
		c.latestReturn = c.deepCopyStr(c.cbb.NewLoad(v.t, v.v))
	} else { // other variables are simply copied
		c.latestReturn = c.cbb.NewLoad(v.t, v.v)
	}
	return c
}

// literals are simple ir constants
func (c *Compiler) VisitIntLit(e *ast.IntLit) ast.Visitor {
	c.latestReturn = constant.NewInt(ddpint, e.Value)
	return c
}
func (c *Compiler) VisitFLoatLit(e *ast.FloatLit) ast.Visitor {
	c.latestReturn = constant.NewFloat(ddpfloat, e.Value)
	return c
}
func (c *Compiler) VisitBoolLit(e *ast.BoolLit) ast.Visitor {
	c.latestReturn = constant.NewBool(e.Value)
	return c
}
func (c *Compiler) VisitCharLit(e *ast.CharLit) ast.Visitor {
	c.latestReturn = constant.NewInt(ddpchar, int64(e.Value))
	return c
}
func (c *Compiler) VisitStringLit(e *ast.StringLit) ast.Visitor {
	strlen := utf8.RuneCountInString(e.Value)
	str := make([]constant.Constant, 0, strlen)
	for _, v := range e.Value {
		str = append(str, constant.NewInt(ddpchar, int64(v)))
	}
	arrType := types.NewArray(uint64(strlen), ddpchar)
	arr := c.mod.NewGlobalDef("", constant.NewArray(arrType, str...))

	strptr := c.cbb.NewBitCast(arr, ptr(ddpchar))

	c.latestReturn = c.cbb.NewCall(c.functions["inbuilt_string_from_constant"].irFunc, strptr, newInt(int64(strlen)))
	return c
}
func (c *Compiler) VisitUnaryExpr(e *ast.UnaryExpr) ast.Visitor {
	rhs := c.evaluate(e.Rhs)
	switch e.Operator.Type {
	case token.BETRAG:
		err("Der BETRAG Operator ist noch nicht implementiert")
	case token.NEGATE:
		switch rhs.Type() {
		case ddpfloat:
			c.latestReturn = c.cbb.NewFNeg(rhs)
		case ddpint:
			c.latestReturn = c.cbb.NewSub(zero, rhs)
		default:
			err("invalid Parameter Type for NEGATE")
		}
	case token.NICHT:
		c.latestReturn = c.cbb.NewXor(rhs, newInt(1))
	case token.LÄNGE:
		notimplemented()
	case token.GRÖßE:
		switch rhs.Type() {
		case ddpint, ddpfloat:
			c.latestReturn = newInt(8)
		case ddpbool:
			c.latestReturn = newInt(1)
		case ddpchar:
			c.latestReturn = newInt(2)
		case ddpstring:
			notimplemented()
		default:
			err("invalid Parameter Type for GRÖßE")
		}
	case token.ZAHL:
		notimplemented()
		/*switch v := rhs.(type) {
		case ddpint:
			i.lastReturn = v
		case ddpfloat:
			i.lastReturn = ddpint(v)
		case ddpbool:
			if v {
				i.lastReturn = ddpint(1)
			} else {
				i.lastReturn = ddpint(0)
			}
		case ddpchar:
			i.lastReturn = ddpint(v)
		case ddpstring:
			if result, Err := strconv.ParseInt(string(v), 10, 64); Err != nil {
				err(e.Token(), fmt.Sprintf("Der Text '%s' kann nicht in eine Zahl umgewandelt werden", string(v)))
			} else {
				i.lastReturn = ddpint(result)
			}
		}*/
	case token.KOMMAZAHL:
		notimplemented()
		/*switch v := rhs.(type) {
		case ddpint:
			i.lastReturn = ddpfloat(v)
		case ddpfloat:
			i.lastReturn = v
		case ddpstring:
			if result, Err := strconv.ParseFloat(string(v), 64); Err != nil {
				err(e.Token(), fmt.Sprintf("Der Text '%s' kann nicht in eine Kommazahl umgewandelt werden", string(v)))
			} else {
				i.lastReturn = ddpfloat(result)
			}
		}*/
	case token.BOOLEAN:
		notimplemented()
		/*switch v := rhs.(type) {
		case ddpint:
			if v == 0 {
				i.lastReturn = ddpbool(false)
			} else {
				i.lastReturn = ddpbool(true)
			}
		case ddpbool:
			i.lastReturn = v
		}*/
	case token.BUCHSTABE:
		notimplemented()
		/*switch v := rhs.(type) {
		case ddpint:
			i.lastReturn = ddpchar(v)
		case ddpchar:
			i.lastReturn = v
		}*/
	case token.TEXT:
		notimplemented()
		/*switch v := rhs.(type) {
		case ddpint:
			i.lastReturn = ddpstring(strconv.FormatInt(int64(v), 64))
		case ddpfloat:
			i.lastReturn = ddpstring(strconv.FormatFloat(float64(v), 'f', -1, 64))
		case ddpchar:
			i.lastReturn = ddpstring(v)
		case ddpstring:
			i.lastReturn = v
		}*/
	default:
		err(fmt.Sprintf("Unbekannter Operator '%s'", e.Operator.String()))
	}
	return c
}
func (c *Compiler) VisitBinaryExpr(e *ast.BinaryExpr) ast.Visitor {
	lhs := c.evaluate(e.Lhs)
	rhs := c.evaluate(e.Rhs)
	switch e.Operator.Type {
	case token.VERKETTET:
		notimplemented()
		/*switch lhs.Type() {
		case ddpstring:
			switch rhs.Type() {
			case ddpstring:
				c.lastReturn = left + right
			case ddpchar:
				c.lastReturn = left + ddpstring(rune(right))
			}
		case ddpchar:
			switch rhs.Type() {
			case ddpstring:
				i.lastReturn = ddpstring(rune(left)) + right
			case ddpchar:
				i.lastReturn = ddpstring(rune(left)) + ddpstring(rune(right))
			}
		}*/
	case token.PLUS:
		switch lhs.Type() {
		case ddpint:
			switch rhs.Type() {
			case ddpint:
				c.latestReturn = c.cbb.NewAdd(lhs, rhs)
			case ddpfloat:
				fp := c.cbb.NewSIToFP(lhs, ddpfloat)
				c.latestReturn = c.cbb.NewFAdd(fp, rhs)
			default:
				err(fmt.Sprintf("invalid Parameter Types for PLUS (%s, %s)", lhs.Type().String(), rhs.Type().String()))
			}
		case ddpfloat:
			switch rhs.Type() {
			case ddpint:
				fp := c.cbb.NewSIToFP(rhs, ddpfloat)
				c.latestReturn = c.cbb.NewFAdd(lhs, fp)
			case ddpfloat:
				c.latestReturn = c.cbb.NewFAdd(lhs, rhs)
			default:
				err(fmt.Sprintf("invalid Parameter Types for PLUS (%s, %s)", lhs.Type().String(), rhs.Type().String()))
			}
		default:
			err(fmt.Sprintf("invalid Parameter Types for PLUS (%s, %s)", lhs.Type().String(), rhs.Type().String()))
		}
	case token.MINUS:
		switch lhs.Type() {
		case ddpint:
			switch rhs.Type() {
			case ddpint:
				c.latestReturn = c.cbb.NewSub(lhs, rhs)
			case ddpfloat:
				fp := c.cbb.NewSIToFP(lhs, ddpfloat)
				c.latestReturn = c.cbb.NewFSub(fp, rhs)
			default:
				err(fmt.Sprintf("invalid Parameter Types for MINUS (%s, %s)", lhs.Type().String(), rhs.Type().String()))
			}
		case ddpfloat:
			switch rhs.Type() {
			case ddpint:
				fp := c.cbb.NewSIToFP(rhs, ddpfloat)
				c.latestReturn = c.cbb.NewFSub(lhs, fp)
			case ddpfloat:
				c.latestReturn = c.cbb.NewFSub(lhs, rhs)
			default:
				err(fmt.Sprintf("invalid Parameter Types for MINUS (%s, %s)", lhs.Type().String(), rhs.Type().String()))
			}
		default:
			err(fmt.Sprintf("invalid Parameter Types for MINUS (%s, %s)", lhs.Type().String(), rhs.Type().String()))
		}
	case token.MAL:
		switch lhs.Type() {
		case ddpint:
			switch rhs.Type() {
			case ddpint:
				c.latestReturn = c.cbb.NewMul(lhs, rhs)
			case ddpfloat:
				fp := c.cbb.NewSIToFP(lhs, ddpfloat)
				c.latestReturn = c.cbb.NewFMul(fp, rhs)
			default:
				err(fmt.Sprintf("invalid Parameter Types for MAL (%s, %s)", lhs.Type().String(), rhs.Type().String()))
			}
		case ddpfloat:
			switch rhs.Type() {
			case ddpint:
				fp := c.cbb.NewSIToFP(rhs, ddpfloat)
				c.latestReturn = c.cbb.NewFMul(lhs, fp)
			case ddpfloat:
				c.latestReturn = c.cbb.NewFMul(lhs, rhs)
			default:
				err(fmt.Sprintf("invalid Parameter Types for MAL (%s, %s)", lhs.Type().String(), rhs.Type().String()))
			}
		default:
			err(fmt.Sprintf("invalid Parameter Types for MAL (%s, %s)", lhs.Type().String(), rhs.Type().String()))
		}
	case token.DURCH:
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
				err(fmt.Sprintf("invalid Parameter Types for DURCH (%s, %s)", lhs.Type().String(), rhs.Type().String()))
			}
		case ddpfloat:
			switch rhs.Type() {
			case ddpint:
				fp := c.cbb.NewSIToFP(rhs, ddpfloat)
				c.latestReturn = c.cbb.NewFDiv(lhs, fp)
			case ddpfloat:
				c.latestReturn = c.cbb.NewFDiv(lhs, rhs)
			default:
				err(fmt.Sprintf("invalid Parameter Types for DURCH (%s, %s)", lhs.Type().String(), rhs.Type().String()))
			}
		default:
			err(fmt.Sprintf("invalid Parameter Types for DURCH (%s, %s)", lhs.Type().String(), rhs.Type().String()))
		}
	case token.MODULO:
		c.latestReturn = c.cbb.NewSRem(lhs, rhs)
	case token.HOCH:
		notimplemented()
		/*switch lhs.Type() {
		case ddpint:
			switch rhs.Type() {
			case ddpint:
				i.lastReturn = ddpint(math.Pow(float64(left), float64(right)))
			case ddpfloat:
				i.lastReturn = ddpfloat(math.Pow(float64(left), float64(right)))
			}
		case ddpfloat:
			switch rhs.Type() {
			case ddpint:
				i.lastReturn = ddpfloat(math.Pow(float64(left), float64(right)))
			case ddpfloat:
				i.lastReturn = ddpfloat(math.Pow(float64(left), float64(right)))
			}
		}*/
	case token.UND:
		c.latestReturn = c.cbb.NewAnd(lhs, rhs)
	case token.ODER:
		c.latestReturn = c.cbb.NewOr(lhs, rhs)
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
		case ddpstring:
			notimplemented()
		default:
			err(fmt.Sprintf("invalid Parameter Types for GLEICH (%s, %s)", lhs.Type().String(), rhs.Type().String()))
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
		case ddpstring:
			notimplemented()
		default:
			err(fmt.Sprintf("invalid Parameter Types for UNGLEICH (%s, %s)", lhs.Type().String(), rhs.Type().String()))
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
				err(fmt.Sprintf("invalid Parameter Types for KLEINER (%s, %s)", lhs.Type().String(), rhs.Type().String()))
			}
		case ddpfloat:
			switch rhs.Type() {
			case ddpint:
				fp := c.cbb.NewSIToFP(rhs, ddpfloat)
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOLT, lhs, fp)
			case ddpfloat:
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOLT, lhs, rhs)
			default:
				err(fmt.Sprintf("invalid Parameter Types for KLEINER (%s, %s)", lhs.Type().String(), rhs.Type().String()))
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
				err(fmt.Sprintf("invalid Parameter Types for KLEINERODER (%s, %s)", lhs.Type().String(), rhs.Type().String()))
			}
		case ddpfloat:
			switch rhs.Type() {
			case ddpint:
				fp := c.cbb.NewSIToFP(rhs, ddpfloat)
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOLE, lhs, fp)
			case ddpfloat:
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOLE, lhs, rhs)
			default:
				err(fmt.Sprintf("invalid Parameter Types for KLEINERODER (%s, %s)", lhs.Type().String(), rhs.Type().String()))
			}
		default:
			err(fmt.Sprintf("invalid Parameter Types for KLEINERODER (%s, %s)", lhs.Type().String(), rhs.Type().String()))
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
				err(fmt.Sprintf("invalid Parameter Types for GRÖßER (%s, %s)", lhs.Type().String(), rhs.Type().String()))
			}
		case ddpfloat:
			switch rhs.Type() {
			case ddpint:
				fp := c.cbb.NewSIToFP(rhs, ddpfloat)
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOGT, lhs, fp)
			case ddpfloat:
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOGT, lhs, rhs)
			default:
				err(fmt.Sprintf("invalid Parameter Types for GRÖßER (%s, %s)", lhs.Type().String(), rhs.Type().String()))
			}
		default:
			err(fmt.Sprintf("invalid Parameter Types for GRÖßER (%s, %s)", lhs.Type().String(), rhs.Type().String()))
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
				err(fmt.Sprintf("invalid Parameter Types for GRÖßERODER (%s, %s)", lhs.Type().String(), rhs.Type().String()))
			}
		case ddpfloat:
			switch rhs.Type() {
			case ddpint:
				fp := c.cbb.NewSIToFP(rhs, ddpfloat)
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOGE, lhs, fp)
			case ddpfloat:
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOGE, lhs, rhs)
			default:
				err(fmt.Sprintf("invalid Parameter Types for GRÖßERODER (%s, %s)", lhs.Type().String(), rhs.Type().String()))
			}
		default:
			err(fmt.Sprintf("invalid Parameter Types for GRÖßERODER (%s, %s)", lhs.Type().String(), rhs.Type().String()))
		}
	}
	return c
}
func (c *Compiler) VisitGrouping(e *ast.Grouping) ast.Visitor {
	return e.Expr.Accept(c)
}
func (c *Compiler) VisitFuncCall(e *ast.FuncCall) ast.Visitor {
	fun := c.functions[e.Name]
	args := make([]value.Value, 0, len(fun.funcDecl.ParamNames))

	toBeFreed := make([]*value.Value, 0)
	for _, param := range fun.funcDecl.ParamNames {
		val := c.evaluate(e.Args[param.Literal])
		if val.Type() == ddpstrptr {
			c.incrementRC(val, VK_STRING)
			toBeFreed = append(toBeFreed, &val)
		}
		args = append(args, val) // possible string ref count is incremented by funcDecl
	}

	c.latestReturn = c.cbb.NewCall(fun.irFunc, args...)
	for i := range toBeFreed {
		c.decrementRC(*toBeFreed[i])
	}
	return c
}

func (c *Compiler) VisitBadStmt(s *ast.BadStmt) ast.Visitor {
	err("Es wurde eine invalide Aussage gefunden")
	return c
}
func (c *Compiler) VisitDeclStmt(s *ast.DeclStmt) ast.Visitor {
	return s.Decl.Accept(c)
}
func (c *Compiler) VisitExprStmt(s *ast.ExprStmt) ast.Visitor {
	expr := c.evaluate(s.Expr)
	// TODO: fix memory error
	if expr.Type() == ddpstrptr { // maybe works?
		c.incrementRC(expr, VK_STRING) // add it to the table (will be made better later)
		c.decrementRC(expr)
	}
	return c
}
func (c *Compiler) VisitAssignStmt(s *ast.AssignStmt) ast.Visitor {
	val := c.evaluate(s.Rhs)
	if val.Type() == ddpstrptr {
		c.incrementRC(val, VK_STRING)
	}
	vr := c.scp.lookupVar(s.Name.Literal)
	if vr.t == ddpstrptr {
		c.decrementRC(c.cbb.NewLoad(vr.t, vr.v))
	}
	c.cbb.NewStore(val, vr.v)
	return c
}
func (c *Compiler) VisitBlockStmt(s *ast.BlockStmt) ast.Visitor {
	c.scp = newScope(c.scp)
	for _, stmt := range s.Statements {
		c.visitNode(stmt)
	}

	c.scp = c.exitScope(c.scp)
	return c
}

// TODO: test ifs, they may be buggy as hell
// TODO: probably leaks memory as soon as strings/local variables are involved
func (c *Compiler) VisitIfStmt(s *ast.IfStmt) ast.Visitor {
	var handleIf func(*ast.IfStmt, *ir.Block) // declaration to use it recursively
	// inner function to handle if-else blocks (we need to pass the leaveBlock down the chain)
	handleIf = func(s *ast.IfStmt, leaveBlock *ir.Block) {
		cbb, scp := c.cbb, c.scp // to restore them later

		// handle the then Block
		thenScope := newScope(c.scp)
		thenBlock := c.cf.NewBlock("")
		c.cbb, c.scp = thenBlock, thenScope
		c.visitNode(s.Then)

		c.cbb, c.scp = cbb, scp

		if s.Else != nil { // handle else and possible else-ifs
			elseScope := newScope(c.scp)
			elseBlock := c.cf.NewBlock("")
			if leaveBlock == nil { // if we don't have a leaveBlock yet, make one
				leaveBlock = c.cf.NewBlock("")
			}
			c.cbb, c.scp = elseBlock, elseScope
			// either execute the else statement or handle the else-if with our leaveBlock
			if elseIf, ok := s.Else.(*ast.IfStmt); ok {
				handleIf(elseIf, leaveBlock)
			} else {
				c.visitNode(s.Else)
			}

			c.cbb, c.scp = cbb, scp
			c.cbb.NewCondBr(c.evaluate(s.Condition), thenBlock, elseBlock)

			// add a terminator
			if thenBlock.Term == nil {
				thenBlock.NewBr(leaveBlock)
			}
			if elseBlock.Term == nil {
				elseBlock.NewBr(leaveBlock)
			}

			c.cbb = leaveBlock
		} else { // if there is no else we just conditionally execute the then block
			if leaveBlock == nil {
				leaveBlock = c.cf.NewBlock("")
			}

			c.cbb, c.scp = cbb, scp
			c.cbb.NewCondBr(c.evaluate(s.Condition), thenBlock, leaveBlock)

			// we need a terminator (simply jump after the then block)
			if thenBlock.Term == nil {
				thenBlock.NewBr(leaveBlock)
			}

			c.cbb = leaveBlock
		}
	}

	handleIf(s, nil)
	return c
}
func (c *Compiler) VisitWhileStmt(s *ast.WhileStmt) ast.Visitor {
	scp := c.scp // to restore it later

	condBlock := c.cf.NewBlock("")
	body, bodyScope := c.cf.NewBlock(""), newScope(c.scp)

	c.cbb.NewBr(condBlock)
	c.cbb, c.scp = body, bodyScope
	c.visitNode(s.Body)
	if c.cbb.Term == nil {
		c.cbb.NewBr(condBlock)
	}

	leaveBlock := c.cf.NewBlock("")
	c.cbb = condBlock
	condBlock.NewCondBr(c.evaluate(s.Condition), body, leaveBlock)

	c.cbb, c.scp = leaveBlock, scp
	return c
}
func (c *Compiler) VisitForStmt(s *ast.ForStmt) ast.Visitor {
	scp := c.scp // to restore it at the end

	c.scp = newScope(c.scp) // scope for the for body
	c.visitNode(s.Initializer)

	condBlock := c.cf.NewBlock("")
	incrementBlock := c.cf.NewBlock("")
	forBody := c.cf.NewBlock("")

	c.cbb.NewBr(condBlock)
	c.cbb = forBody
	c.visitNode(s.Body)
	if c.cbb.Term == nil {
		c.cbb.NewBr(incrementBlock)
	}

	// compile the incrementBlock
	v := c.scp.lookupVar(s.Initializer.Name.Literal)
	indexVar := incrementBlock.NewLoad(v.t, v.v)
	var incrementer value.Value
	if s.StepSize == nil {
		incrementer = constant.NewInt(ddpint, 1)
	} else {
		c.cbb = incrementBlock
		incrementer = c.evaluate(s.StepSize)
	}
	add := incrementBlock.NewAdd(indexVar, incrementer)
	incrementBlock.NewStore(add, c.scp.lookupVar(s.Initializer.Name.Literal).v)
	incrementBlock.NewBr(condBlock)

	leaveBlock := c.cf.NewBlock("")
	c.cbb = condBlock
	cond := condBlock.NewICmp(enum.IPredEQ, condBlock.NewLoad(v.t, v.v), c.evaluate(s.To))
	condBlock.NewCondBr(cond, leaveBlock, forBody)

	leaveBlock2 := c.cf.NewBlock("")
	c.cbb = leaveBlock
	c.visitNode(s.Body)
	if c.cbb.Term == nil {
		c.cbb.NewBr(leaveBlock2)
	}

	c.cbb, c.scp = leaveBlock2, scp
	return c
}
func (c *Compiler) VisitFuncCallStmt(s *ast.FuncCallStmt) ast.Visitor {
	return s.Call.Accept(c)
}
func (c *Compiler) VisitReturnStmt(s *ast.ReturnStmt) ast.Visitor {
	ret := c.evaluate(s.Value)
	if ret.Type() == ddpstrptr {
		oldRet := ret
		c.incrementRC(oldRet, VK_STRING)
		ret = c.deepCopyStr(oldRet)
		c.decrementRC(oldRet)
	}
	c.cbb.NewRet(ret)
	return c
}

// helper functions

func notimplemented() {
	file, line, function := getTraceInfo(2)
	panic(fmt.Errorf("%s, %d, %s: this function or a part of it is not implemented", filepath.Base(file), line, function))
}

/*func (c *Compiler) currentlyskipped() {
	file, line, function := getTraceInfo(2)
	c.errorHandler(fmt.Sprintf("%s, %d, %s: this function or a part of it is currently being ignored", filepath.Base(file), line, function))
}*/

func getTraceInfo(skip int) (file string, line int, function string) {
	pc := make([]uintptr, 15)
	n := runtime.Callers(skip+1, pc)
	frames := runtime.CallersFrames(pc[:n])
	frame, _ := frames.Next()
	return frame.File, frame.Line, frame.Function
}
