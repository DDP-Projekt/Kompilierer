package compiler

import (
	"DDP/pkg/ast"
	"DDP/pkg/scanner"
	"DDP/pkg/token"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf8"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

type funcWrapper struct {
	irFunc   *ir.Func
	funcDecl *ast.FuncDecl
}

type Compiler struct {
	ast          *ast.Ast
	mod          *ir.Module
	errorHandler scanner.ErrorHandler

	cbb          *ir.Block // current basic block
	cf           *ir.Func  // current function
	scp          *scope
	functions    map[string]*funcWrapper
	latestReturn value.Value
}

func New(Ast *ast.Ast, errorHandler scanner.ErrorHandler) *Compiler {
	if errorHandler == nil {
		errorHandler = func(string) {}
	}
	return &Compiler{
		ast:          Ast,
		mod:          ir.NewModule(),
		errorHandler: errorHandler,
		cbb:          nil,
		cf:           nil,
		scp:          newScope(nil),
		functions:    map[string]*funcWrapper{},
		latestReturn: nil,
	}
}

func (c *Compiler) Compile() (result string, rerr error) {
	defer func() {
		if err := recover(); err != nil {
			rerr = err.(error)
			result = ""
		}
	}()

	if c.ast.Faulty {
		return "", fmt.Errorf("Fehlerhafter Syntax Baum")
	}

	c.mod.SourceFilename = c.ast.File
	c.setupStringType()                                                                        // setup some internal functions to work with strings; might be removed later
	main := c.insertFunction("inbuilt_ddpmain", nil, c.mod.NewFunc("inbuilt_ddpmain", ddpint)) // called from the ddp-c-runtime after initialization
	c.cf = main
	c.cbb = main.NewBlock("")

	for _, stmt := range c.ast.Statements {
		c.visitNode(stmt)
	}

	c.cbb.NewRet(newInt(0))
	return c.mod.String(), nil
}

// helper that might be extended later
func err(msg string) {
	_, file, line, _ := runtime.Caller(1)
	panic(fmt.Errorf("%s, %d: %s", filepath.Base(file), line, msg))
}

// convenience functions

func (c *Compiler) visitNode(node ast.Node) {
	c = node.Accept(c).(*Compiler)
}

func (c *Compiler) evaluate(expr ast.Expression) value.Value {
	c.visitNode(expr)
	return c.latestReturn
}

func (c *Compiler) insertFunction(name string, funcDecl *ast.FuncDecl, irFunc *ir.Func) *ir.Func {
	c.functions[name] = &funcWrapper{
		funcDecl: funcDecl,
		irFunc:   irFunc,
	}
	return irFunc
}

// declare some internal string functions
func (c *Compiler) setupStringType() {
	ddpstring.Fields = make([]types.Type, 2)
	ddpstring.Fields[0] = ptr(ddpchar)
	ddpstring.Fields[1] = ddpint
	c.mod.NewTypeDef("ddpstring", ddpstring)

	/*sfcret := ir.NewParam("", ddpstrptr)
	sfcret.Attrs = append(sfcret.Attrs, enum.ParamAttrNoAlias)
	sfcret.Attrs = append(sfcret.Attrs, ir.SRet{Typ: ddpstring})*/
	sfc := c.mod.NewFunc("inbuilt_string_from_constant", ddpstrptr, ir.NewParam("str", ptr(ddpchar)), ir.NewParam("len", ddpint))
	sfc.CallingConv = enum.CallingConvC
	sfc.Linkage = enum.LinkageExternal
	c.insertFunction("inbuilt_string_from_constant", nil, sfc)

	/*dcsret := ir.NewParam("", ddpstrptr)
	dcsret.Attrs = append(dcsret.Attrs, enum.ParamAttrNoAlias)
	dcsret.Attrs = append(dcsret.Attrs, ir.SRet{Typ: ddpstring})*/
	dcs := c.mod.NewFunc("inbuilt_deep_copy_string", ddpstrptr, ir.NewParam("str", ddpstrptr))
	dcs.CallingConv = enum.CallingConvC
	dcs.Linkage = enum.LinkageExternal
	c.insertFunction("inbuilt_deep_copy_string", nil, dcs)

	drc := c.mod.NewFunc("inbuilt_decrement_ref_count", void, ir.NewParam("key", ptr(i8)))
	drc.CallingConv = enum.CallingConvC
	drc.Linkage = enum.LinkageExternal
	c.insertFunction("inbuilt_decrement_ref_count", nil, drc)

	irc := c.mod.NewFunc("inbuilt_increment_ref_count", void, ir.NewParam("key", ptr(i8)), ir.NewParam("kind", i8))
	irc.CallingConv = enum.CallingConvC
	irc.Linkage = enum.LinkageExternal
	c.insertFunction("inbuilt_increment_ref_count", nil, irc)
}

/*func (c *Compiler) markString(strptr value.Value) {
	c.cbb.NewCall(c.functions["mark_gc"].irFunc, c.cbb.NewBitCast(strptr, ptr(garbage_collected)))
}*/

func (c *Compiler) incrementRC(key value.Value, kind *constant.Int) {
	c.cbb.NewCall(c.functions["inbuilt_increment_ref_count"].irFunc, c.cbb.NewBitCast(key, ptr(i8)), kind)
}

func (c *Compiler) decrementRC(key value.Value) {
	c.cbb.NewCall(c.functions["inbuilt_decrement_ref_count"].irFunc, c.cbb.NewBitCast(key, ptr(i8)))
}

func (c *Compiler) deepCopyStr(strptr value.Value) value.Value {
	return c.cbb.NewCall(c.functions["inbuilt_deep_copy_string"].irFunc, strptr)
}

func (c *Compiler) exitScope(scp *scope) *scope {
	for _, v := range c.scp.variables {
		if v.t == ddpstrptr {
			//c.markString(c.cbb.NewLoad(ddpstrptr, v.v))
			c.decrementRC(c.cbb.NewLoad(ddpstrptr, v.v))
		}
	}
	return scp.enclosing
}

func (c *Compiler) VisitBadDecl(d *ast.BadDecl) ast.Visitor {
	err("Es wurde eine invalide Deklaration gefunden")
	return c
}
func (c *Compiler) VisitVarDecl(d *ast.VarDecl) ast.Visitor {
	t := toDDPType(d.Type.Type)
	if t == ddpstring {
		t = ddpstrptr
	}
	v := c.scp.addVar(d.Name.Literal, c.cf.Blocks[0].NewAlloca(t), t) // allocate the variable on the function call frame
	initVal := c.evaluate(d.InitVal)
	c.cbb.NewStore(initVal, v) // store the init value
	if t == ddpstrptr {
		c.incrementRC(initVal, VK_STRING)
	}
	return c
}
func (c *Compiler) VisitFuncDecl(d *ast.FuncDecl) ast.Visitor {
	retType := toDDPType(d.Type.Type)
	params := make([]*ir.Param, 0, len(d.ParamTypes))
	if retType == ddpstring {
		retType = ddpstrptr
	}

	// append all the other parameters
	for i, tok := range d.ParamTypes {
		ty := toDDPType(tok.Type)
		if ty == ddpstring {
			ty = ddpstrptr // strings are passed as pointers
		}
		params = append(params, ir.NewParam(d.ParamNames[i].Literal, ty))
	}

	// append a prefix to every ir function to make it impossible for the user to break internal stuff
	name := d.Name.Literal
	if isInbuiltFunc(d) {
		name = "inbuilt_" + strings.TrimLeft(name, "§")
	} else {
		name = "ddpfunc_" + name
	}
	irFunc := c.mod.NewFunc(name, retType, params...)
	irFunc.CallingConv = enum.CallingConvC // every function is called with the c calling convention to make interaction with inbuilt stuff easier

	c.insertFunction(d.Name.Literal, d, irFunc)

	if isInbuiltFunc(d) {
		irFunc.Linkage = enum.LinkageExternal // inbuilt functions are defined in c
	} else {
		fun, block := c.cf, c.cbb // safe the state before the function body
		c.cf, c.cbb, c.scp = irFunc, irFunc.NewBlock(""), newScope(c.scp)
		// passed arguments are immutible (llvm uses ssa registers) so we declare them as local variables
		for i := range params {
			if d.ParamTypes[i].Type == token.TEXT {
				c.incrementRC(params[i], VK_STRING)
				v := c.scp.addVar(params[i].LocalIdent.Name(), c.cbb.NewAlloca(ddpstrptr), ddpstrptr)
				strptr := c.deepCopyStr(params[i]) // deep copy the passed pointer to string
				c.incrementRC(strptr, VK_STRING)
				c.cbb.NewStore(strptr, v)
				//c.markString(params[i])
				c.decrementRC(params[i])
			} else {
				v := c.scp.addVar(params[i].LocalIdent.Name(), c.cbb.NewAlloca(params[i].Type()), params[i].Type())
				c.cbb.NewStore(params[i], v)
			}
		}
		c.visitNode(d.Body)
		if c.cbb.Term == nil {
			c.cbb.NewRet(nil) // every block needs a terminator, and every function a return
		}
		c.cf, c.cbb, c.scp = fun, block, c.exitScope(c.scp) // restore state before the function (to main)
	}

	return c
}

func (c *Compiler) VisitBadExpr(e *ast.BadExpr) ast.Visitor {
	err("Es wurde ein invalider Ausdruck gefunden")
	return c
}
func (c *Compiler) VisitIdent(e *ast.Ident) ast.Visitor {
	v := c.scp.lookupVar(e.Literal.Literal)
	if v.t == ddpstrptr {
		c.latestReturn = c.deepCopyStr(c.cbb.NewLoad(v.t, v.v))
	} else {
		c.latestReturn = c.cbb.NewLoad(v.t, v.v)
	}
	return c
}
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
		//c.markString(expr)
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
		c.decrementRC(vr.v)
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
		//c.markString(oldRet)
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

func (c *Compiler) currentlyskipped() {
	file, line, function := getTraceInfo(2)
	c.errorHandler(fmt.Sprintf("%s, %d, %s: this function or a part of it is currently being ignored", filepath.Base(file), line, function))
}

func getTraceInfo(skip int) (file string, line int, function string) {
	pc := make([]uintptr, 15)
	n := runtime.Callers(skip+1, pc)
	frames := runtime.CallersFrames(pc[:n])
	frame, _ := frames.Next()
	return frame.File, frame.Line, frame.Function
}
