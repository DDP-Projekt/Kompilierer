package compiler

import (
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/DDP-Projekt/Kompilierer/pkg/ast"
	"github.com/DDP-Projekt/Kompilierer/pkg/ddperror"
	"github.com/DDP-Projekt/Kompilierer/pkg/ddptypes"
	"github.com/DDP-Projekt/Kompilierer/pkg/token"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
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
	mod          *ir.Module       // the ir module (basically the ir file)
	errorHandler ddperror.Handler // errors are passed to this function
	result       *Result          // result of the compilation

	cbb              *ir.Block               // current basic block in the ir
	cf               *ir.Func                // current function
	scp              *scope                  // current scope in the ast (not in the ir)
	cfscp            *scope                  // out-most scope of the current function
	functions        map[string]*funcWrapper // all the global functions
	latestReturn     value.Value             // return of the latest evaluated expression (in the ir)
	latestReturnType ddpIrType               // the type of latestReturn

	// all the type definitions of inbuilt types used by the compiler
	void                                                              *ddpIrVoidType
	ddpinttyp, ddpfloattyp, ddpbooltyp, ddpchartyp                    *ddpIrPrimitiveType
	ddpstring                                                         *ddpIrStringType
	ddpintlist, ddpfloatlist, ddpboollist, ddpcharlist, ddpstringlist *ddpIrListType
}

// create a new Compiler to compile the passed AST
func New(Ast *ast.Ast, errorHandler ddperror.Handler) *Compiler {
	if errorHandler == nil { // default error handler does nothing
		errorHandler = ddperror.EmptyHandler
	}
	return &Compiler{
		ast:          Ast,
		mod:          ir.NewModule(),
		errorHandler: errorHandler,
		result: &Result{
			Dependencies: make(map[string]struct{}),
			Output:       "",
		},
		cbb:          nil,
		cf:           nil,
		scp:          newScope(nil), // global scope
		cfscp:        nil,
		functions:    map[string]*funcWrapper{},
		latestReturn: nil,
	}
}

// compile the AST contained in c
// if w is not nil, the resulting llir is written to w
// otherwise a string representation is returned in result
func (c *Compiler) Compile(w io.Writer) (result *Result, rerr error) {
	// the ast must be valid (and should have been resolved and typechecked beforehand)
	if c.ast.Faulty {
		return nil, fmt.Errorf("Fehlerhafter Quellcode, Kompilierung abgebrochen")
	}

	c.mod.SourceFilename = c.ast.File // set the module filename (optional metadata)

	c.setup()

	// called from the ddp-c-runtime after initialization
	ddpmain := c.insertFunction(
		"ddp_ddpmain",
		nil,
		c.mod.NewFunc("ddp_ddpmain", ddpint),
	)
	c.cf = ddpmain               // first function is ddpmain
	c.cbb = ddpmain.NewBlock("") // first block

	// visit every statement in the AST and compile it
	for _, stmt := range c.ast.Statements {
		c.visitNode(stmt)
	}

	c.scp = c.exitScope(c.scp) // exit the main scope

	// on success ddpmain returns 0
	c.cbb.NewRet(zero)
	if w != nil {
		_, err := c.mod.WriteTo(w)
		return c.result, err
	} else {
		c.result.Output = c.mod.String()
		return c.result, nil // return the module as string
	}
}

// helper that might be extended later
// it is not intended for the end user to see these errors, as they are compiler bugs
// the errors in the ddp-code were already reported by the parser/typechecker/resolver
func err(msg string, args ...any) {
	// retreive the file and line on which the error occured
	_, file, line, _ := runtime.Caller(1)
	panic(fmt.Errorf("%s, %d: %s", filepath.Base(file), line, fmt.Sprintf(msg, args...)))
}

// if the llvm-ir should be commented
// increases the intermediate file size
var Comments_Enabled = true

func (c *Compiler) commentNode(block *ir.Block, node ast.Node, details string) {
	if Comments_Enabled {
		comment := fmt.Sprintf("F %s, %d:%d: %s", node.Token().File, node.Token().Line(), node.Token().Column(), node)
		if details != "" {
			comment += " (" + details + ")"
		}
		c.comment(comment, block)
	}
}

func (c *Compiler) comment(comment string, block *ir.Block) {
	if Comments_Enabled {
		block.Insts = append(block.Insts, irutil.NewComment(comment))
	}
}

// helper to visit a single node
func (c *Compiler) visitNode(node ast.Node) {
	node.Accept(c)
}

// helper to evalueate an expression and return its ir value
// if the result is refCounted it's refcount is usually 1
func (c *Compiler) evaluate(expr ast.Expression) (value.Value, ddpIrType) {
	c.visitNode(expr)
	return c.latestReturn, c.latestReturnType
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

func (c *Compiler) setup() {
	// the order of these function calls is important
	// because the primitive types need to be setup
	// before the list types
	c.void = &ddpIrVoidType{}
	c.initRuntimeFunctions()
	c.setupPrimitiveTypes()
	c.ddpstring = c.defineStringType()
	c.setupListTypes()

	c.setupOperators()
}

// used in setup()
func (c *Compiler) setupPrimitiveTypes() {
	c.ddpinttyp = c.definePrimitiveType(ddpint, zero, "ddpint")
	c.ddpfloattyp = c.definePrimitiveType(ddpfloat, zerof, "ddpfloat")
	c.ddpbooltyp = c.definePrimitiveType(ddpbool, constant.False, "ddpbool")
	c.ddpchartyp = c.definePrimitiveType(ddpchar, newIntT(ddpchar, 0), "ddpchar")
}

// used in setup()
func (c *Compiler) setupListTypes() {
	c.ddpintlist = c.defineListType("ddpintlist", c.ddpinttyp)
	c.ddpfloatlist = c.defineListType("ddpfloatlist", c.ddpfloattyp)
	c.ddpboollist = c.defineListType("ddpboollist", c.ddpbooltyp)
	c.ddpcharlist = c.defineListType("ddpcharlist", c.ddpchartyp)
	c.ddpstringlist = c.defineListType("ddpstringlist", c.ddpstring)
}

// used in setup()
func (c *Compiler) setupOperators() {
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
func (c *Compiler) deepCopyInto(dest, src value.Value, typ ddpIrType) value.Value {
	c.cbb.NewCall(typ.DeepCopyFunc(), dest, src)
	return dest
}

// calls the corresponding free function on val
// if typ.IsPrimitive() == false
func (c *Compiler) freeNonPrimitive(val value.Value, typ ddpIrType) {
	if !typ.IsPrimitive() {
		c.cbb.NewCall(typ.FreeFunc(), val)
	}
}

// helper to exit a scope
// decrements the ref-count on all local variables
// returns the enclosing scope
func (c *Compiler) exitScope(scp *scope) *scope {
	for _, v := range scp.variables {
		if !v.isRef {
			c.freeNonPrimitive(v.val, v.typ)
		}
	}
	return scp.enclosing
}

func (*Compiler) BaseVisitor() {}

// should have been filtered by the resolver/typechecker, so err
func (c *Compiler) VisitBadDecl(d *ast.BadDecl) {
	err("Es wurde eine invalide Deklaration gefunden")
}
func (c *Compiler) VisitVarDecl(d *ast.VarDecl) {
	Typ := c.toIrType(d.Type) // get the llvm type
	// allocate the variable on the function call frame
	// all local variables are allocated in the first basic block of the function they are within
	// in the ir a local variable is a alloca instruction (a stack allocation)
	// global variables are allocated in the ddpmain function
	c.commentNode(c.cbb, d, d.Name.Literal)
	var varLocation value.Value
	if c.scp.enclosing == nil { // global scope
		varLocation = c.mod.NewGlobalDef("", Typ.DefaultValue())
	} else {
		varLocation = c.cf.Blocks[0].NewAlloca(Typ.IrType())
	}
	Var := c.scp.addVar(d.Name.Literal, varLocation, Typ, false)
	initVal, _ := c.evaluate(d.InitVal) // evaluate the initial value
	if !Typ.IsPrimitive() {
		initVal = c.cbb.NewLoad(Typ.IrType(), initVal)
	}
	c.cbb.NewStore(initVal, Var) // store the initial value
}
func (c *Compiler) VisitFuncDecl(d *ast.FuncDecl) {
	retType := c.toIrType(d.Type) // get the llvm type
	retTypeIr := retType.IrType()
	params := make([]*ir.Param, 0, len(d.ParamTypes)) // list of the ir parameters

	hasReturnParam := !retType.IsPrimitive()
	// non-primitives are returned by passing a pointer to the struct as first parameter
	if hasReturnParam {
		params = append(params, ir.NewParam("", retType.PtrType()))
		retTypeIr = c.void.IrType()
	}

	// append all the other parameters
	for i, typ := range d.ParamTypes {
		ty := c.toIrParamType(typ)                                        // convert the type of the parameter
		params = append(params, ir.NewParam(d.ParamNames[i].Literal, ty)) // add it to the list
	}

	irFunc := c.mod.NewFunc(d.Name.Literal, retTypeIr, params...) // create the ir function
	irFunc.CallingConv = enum.CallingConvC                        // every function is called with the c calling convention to make interaction with inbuilt stuff easier

	c.insertFunction(d.Name.Literal, d, irFunc)

	// inbuilt or external functions are defined in c
	if ast.IsExternFunc(d) {
		irFunc.Linkage = enum.LinkageExternal
		path, err := filepath.Abs(filepath.Join(filepath.Dir(d.Token().File), strings.Trim(d.ExternFile.Literal, "\"")))
		if err != nil {
			c.errorHandler(ddperror.Error{File: d.ExternFile.File, Range: d.ExternFile.Range, Msg: err.Error()})
		}
		c.result.Dependencies[path] = struct{}{} // add the file-path where the function is defined to the dependencies set
	} else {
		fun, block := c.cf, c.cbb // safe the state before the function body
		c.cf, c.cbb, c.scp = irFunc, irFunc.NewBlock(""), newScope(c.scp)
		c.cfscp = c.scp

		// we want to skip the possible return-parameter
		if hasReturnParam {
			params = params[1:]
		}
		// passed arguments are immutible (llvm uses ssa registers) so we declare them as local variables
		// the caller has to take care of possible deep-copies
		for i := range params {
			irType := c.toIrType(d.ParamTypes[i].Type)
			if d.ParamTypes[i].IsReference {
				// references are implemented similar to name-shadowing
				// they basically just get another name in the function scope, which
				// refers to the same variable allocation
				c.scp.addVar(params[i].LocalIdent.Name(), params[i], irType, true)
			} else if !irType.IsPrimitive() { // strings and lists need special handling
				// add the local variable for the parameter
				v := c.scp.addVar(params[i].LocalIdent.Name(), c.cbb.NewAlloca(irType.IrType()), irType, false)
				c.cbb.NewStore(c.cbb.NewLoad(irType.IrType(), params[i]), v) // store the copy in the local variable
			} else { // primitive types don't need any special handling
				v := c.scp.addVar(params[i].LocalIdent.Name(), c.cbb.NewAlloca(irType.IrType()), irType, false)
				c.cbb.NewStore(params[i], v)
			}
		}

		// modified VisitBlockStmt
		c.scp = newScope(c.scp) // a block gets its own scope
		toplevelReturn := false
		for _, stmt := range d.Body.Statements {
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
			c.scp = c.exitScope(c.scp)
		}
		c.cf, c.cbb, c.cfscp = fun, block, nil // restore state before the function (to main)
	}
}

// should have been filtered by the resolver/typechecker, so err
func (c *Compiler) VisitBadExpr(e *ast.BadExpr) {
	err("Es wurde ein invalider Ausdruck gefunden")
}
func (c *Compiler) VisitIdent(e *ast.Ident) {
	Var := c.scp.lookupVar(e.Literal.Literal) // get the alloca in the ir
	c.commentNode(c.cbb, e, e.Literal.Literal)

	if Var.typ.IsPrimitive() { // primitives are simply loaded
		c.latestReturn = c.cbb.NewLoad(Var.typ.IrType(), Var.val)
	} else { // non-primitives need to be deep copied
		dest := c.cbb.NewAlloca(Var.typ.IrType())
		c.latestReturn = c.deepCopyInto(dest, Var.val, Var.typ)
	}
	c.latestReturnType = Var.typ
}

func (c *Compiler) VisitIndexing(e *ast.Indexing) {
	err("Indexings are only meant to be used as assignables")
}

// literals are simple ir constants
func (c *Compiler) VisitIntLit(e *ast.IntLit) {
	c.commentNode(c.cbb, e, "")
	c.latestReturn = newInt(e.Value)
	c.latestReturnType = c.ddpinttyp
}
func (c *Compiler) VisitFloatLit(e *ast.FloatLit) {
	c.commentNode(c.cbb, e, "")
	c.latestReturn = constant.NewFloat(ddpfloat, e.Value)
	c.latestReturnType = c.ddpfloattyp
}
func (c *Compiler) VisitBoolLit(e *ast.BoolLit) {
	c.commentNode(c.cbb, e, "")
	c.latestReturn = constant.NewBool(e.Value)
	c.latestReturnType = c.ddpbooltyp
}
func (c *Compiler) VisitCharLit(e *ast.CharLit) {
	c.commentNode(c.cbb, e, "")
	c.latestReturn = newIntT(ddpchar, int64(e.Value))
	c.latestReturnType = c.ddpchartyp
}

// string literals are created by the runtime
// so we need to do some work here
func (c *Compiler) VisitStringLit(e *ast.StringLit) {
	constStr := c.mod.NewGlobalDef("", irutil.NewCString(e.Value))
	// call the ddp-runtime function to create the ddpstring
	c.commentNode(c.cbb, e, constStr.Name())
	dest := c.cbb.NewAlloca(c.ddpstring.typ)
	c.cbb.NewCall(c.ddpstring.fromConstantsIrFun, dest, c.cbb.NewBitCast(constStr, ptr(i8)))
	c.latestReturn = dest
	c.latestReturnType = c.ddpstring
}
func (c *Compiler) VisitListLit(e *ast.ListLit) {
	listType := c.toIrType(e.Type).(*ddpIrListType)
	list := c.cbb.NewAlloca(listType.IrType())

	// get the listLen as irValue
	var listLen value.Value = zero
	if e.Values != nil {
		listLen = newInt(int64(len(e.Values)))
	} else if e.Count != nil && e.Value != nil {
		listLen, _ = c.evaluate(e.Count)
	} else { // empty list
		c.cbb.NewStore(listType.DefaultValue(), list)
		c.latestReturn = list
		c.latestReturnType = listType
		return
	}

	// create a empty list of the correct length
	c.cbb.NewCall(listType.fromConstantsIrFun, list, listLen)

	listArr := c.loadStructField(list, arr_field_index) // load the array

	if e.Values != nil { // we got some values to copy
		// evaluate every value and copy it into the array
		for i, v := range e.Values {
			val, valTyp := c.evaluate(v)
			elementPtr := c.indexArray(listArr, newInt(int64(i)))
			if !valTyp.IsPrimitive() {
				val = c.cbb.NewLoad(valTyp.IrType(), val)
			}
			c.cbb.NewStore(val, elementPtr)
		}
	} else if e.Count != nil && e.Value != nil { // single Value multiple times
		Value, _ := c.evaluate(e.Value)

		c.createFor(zero, c.forDefaultCond(listLen), func(index value.Value) {
			elementPtr := c.indexArray(listArr, index)
			if listType.elementType.IsPrimitive() {
				c.cbb.NewStore(Value, elementPtr)
			} else {
				c.deepCopyInto(elementPtr, Value, listType.elementType)
			}
		})

		c.freeNonPrimitive(Value, listType.elementType)
	}
	c.latestReturn = list
	c.latestReturnType = listType
}
func (c *Compiler) VisitUnaryExpr(e *ast.UnaryExpr) {
	const all_ones int64 = ^0 // int with all bits set to 1

	rhs, typ := c.evaluate(e.Rhs) // compile the expression onto which the operator is applied
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
			err("invalid Parameter Type for BETRAG: %s", typ.Name())
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
			err("invalid Parameter Type for NEGATE: %s", typ.Name())
		}
	case ast.UN_NOT:
		c.latestReturn = c.cbb.NewXor(rhs, newInt(1))
		c.latestReturnType = c.ddpbooltyp
	case ast.UN_LOGIC_NOT:
		c.latestReturn = c.cbb.NewXor(rhs, newInt(all_ones))
		c.latestReturnType = c.ddpinttyp
	case ast.UN_LEN:
		switch typ {
		case c.ddpstring:
			c.latestReturn = c.cbb.NewCall(c.ddpstring.lengthIrFun, rhs)
		default:
			if _, isList := typ.(*ddpIrListType); isList {
				c.latestReturn = c.loadStructField(rhs, len_field_index)
			} else {
				err("invalid Parameter Type for LÄNGE: %s", typ.Name())
			}
		}
		c.latestReturnType = c.ddpinttyp
	case ast.UN_SIZE:
		switch typ {
		case c.ddpinttyp, c.ddpfloattyp:
			c.latestReturn = newInt(8)
		case c.ddpbooltyp:
			c.latestReturn = newInt(1)
		case c.ddpchartyp:
			c.latestReturn = newInt(4)
		case c.ddpstring:
			c.latestReturn = c.cbb.NewAdd(c.loadStructField(rhs, 1), c.sizeof(c.ddpstring.typ))
		default:
			if _, isList := typ.(*ddpIrListType); isList {
				cap := c.loadStructField(rhs, cap_field_index)
				cap_times_size := c.cbb.NewMul(cap, c.sizeof(typ.(*ddpIrListType).elementType.IrType()))
				c.latestReturn = c.cbb.NewAdd(cap_times_size, c.sizeof(typ.IrType()))
			} else {
				err("invalid Parameter Type for GRÖßE: %s", typ.Name())
			}
		}
		c.latestReturnType = c.ddpinttyp
	default:
		err("Unbekannter Operator '%s'", e.Operator)
	}

	c.freeNonPrimitive(rhs, typ)
}
func (c *Compiler) VisitBinaryExpr(e *ast.BinaryExpr) {
	// for UND and ODER both operands are booleans, so we don't need to worry about memory management
	switch e.Operator {
	case ast.BIN_AND:
		lhs, _ := c.evaluate(e.Lhs)
		startBlock, trueBlock, leaveBlock := c.cbb, c.cf.NewBlock(""), c.cf.NewBlock("")
		c.commentNode(c.cbb, e, e.Operator.String())
		c.cbb.NewCondBr(lhs, trueBlock, leaveBlock)

		c.cbb = trueBlock
		rhs, _ := c.evaluate(e.Rhs)
		c.commentNode(c.cbb, e, e.Operator.String())
		c.cbb.NewBr(leaveBlock)
		trueBlock = c.cbb

		c.cbb = leaveBlock
		c.commentNode(c.cbb, e, e.Operator.String())
		c.latestReturn = c.cbb.NewPhi(ir.NewIncoming(rhs, trueBlock), ir.NewIncoming(lhs, startBlock))
		c.latestReturnType = c.ddpbooltyp
		return
	case ast.BIN_OR:
		lhs, _ := c.evaluate(e.Lhs)
		startBlock, falseBlock, leaveBlock := c.cbb, c.cf.NewBlock(""), c.cf.NewBlock("")
		c.commentNode(c.cbb, e, e.Operator.String())
		c.cbb.NewCondBr(lhs, leaveBlock, falseBlock)

		c.cbb = falseBlock
		rhs, _ := c.evaluate(e.Rhs)
		c.commentNode(c.cbb, e, e.Operator.String())
		c.cbb.NewBr(leaveBlock)
		falseBlock = c.cbb // in case c.evaluate has multiple blocks

		c.cbb = leaveBlock
		c.commentNode(c.cbb, e, e.Operator.String())
		c.latestReturn = c.cbb.NewPhi(ir.NewIncoming(lhs, startBlock), ir.NewIncoming(rhs, falseBlock))
		c.latestReturnType = c.ddpbooltyp
		return
	}

	// compile the two expressions onto which the operator is applied
	lhs, lhsTyp := c.evaluate(e.Lhs)
	rhs, rhsTyp := c.evaluate(e.Rhs)
	// big switches on the different type combinations
	c.commentNode(c.cbb, e, e.Operator.String())
	switch e.Operator {
	case ast.BIN_CONCAT:
		var (
			result    value.Value
			resultTyp ddpIrType
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
		result = c.cbb.NewAlloca(resultTyp.IrType())

		// string concatenations
		var concat_func *ir.Func = nil
		if lhsTyp == c.ddpstring && rhsTyp == c.ddpstring {
			concat_func = c.ddpstring.str_str_concat_IrFunc
		} else if lhsTyp == c.ddpstring && rhsTyp == c.ddpchartyp {
			concat_func = c.ddpstring.str_char_concat_IrFunc
		} else if lhsTyp == c.ddpchartyp && rhsTyp == c.ddpstring {
			concat_func = c.ddpstring.char_str_concat_IrFunc
		}

		// list concatenations
		if concat_func == nil {
			if lhsIsList && rhsIsList {
				concat_func = lhsListTyp.list_list_concat_IrFunc
			} else if lhsIsList && !rhsIsList {
				concat_func = lhsListTyp.list_scalar_concat_IrFunc
			} else if !lhsIsList && !rhsIsList {
				concat_func = c.getListType(lhsTyp).scalar_scalar_concat_IrFunc
			} else if !lhsIsList && rhsIsList {
				concat_func = rhsListTyp.scalar_list_concat_IrFunc
			}
		}

		c.cbb.NewCall(concat_func, result, lhs, rhs)
		c.latestReturn = result
		c.latestReturnType = resultTyp
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
				err("invalid Parameter Types for PLUS (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		case c.ddpfloattyp:
			switch rhsTyp {
			case c.ddpinttyp:
				fp := c.cbb.NewSIToFP(rhs, ddpfloat)
				c.latestReturn = c.cbb.NewFAdd(lhs, fp)
			case c.ddpfloattyp:
				c.latestReturn = c.cbb.NewFAdd(lhs, rhs)
			default:
				err("invalid Parameter Types for PLUS (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
			c.latestReturnType = c.ddpfloattyp
		default:
			err("invalid Parameter Types for PLUS (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
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
				err("invalid Parameter Types for MINUS (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		case c.ddpfloattyp:
			switch rhsTyp {
			case c.ddpinttyp:
				fp := c.cbb.NewSIToFP(rhs, ddpfloat)
				c.latestReturn = c.cbb.NewFSub(lhs, fp)
			case c.ddpfloattyp:
				c.latestReturn = c.cbb.NewFSub(lhs, rhs)
			default:
				err("invalid Parameter Types for MINUS (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
			c.latestReturnType = c.ddpfloattyp
		default:
			err("invalid Parameter Types for MINUS (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
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
				err("invalid Parameter Types for MAL (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		case c.ddpfloattyp:
			switch rhsTyp {
			case c.ddpinttyp:
				fp := c.cbb.NewSIToFP(rhs, ddpfloat)
				c.latestReturn = c.cbb.NewFMul(lhs, fp)
			case c.ddpfloattyp:
				c.latestReturn = c.cbb.NewFMul(lhs, rhs)
			default:
				err("invalid Parameter Types for MAL (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
			c.latestReturnType = c.ddpfloattyp
		default:
			err("invalid Parameter Types for MAL (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
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
				err("invalid Parameter Types for DURCH (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		case c.ddpfloattyp:
			switch rhsTyp {
			case c.ddpinttyp:
				fp := c.cbb.NewSIToFP(rhs, ddpfloat)
				c.latestReturn = c.cbb.NewFDiv(lhs, fp)
			case c.ddpfloattyp:
				c.latestReturn = c.cbb.NewFDiv(lhs, rhs)
			default:
				err("invalid Parameter Types for DURCH (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		default:
			err("invalid Parameter Types for DURCH (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
		}
		c.latestReturnType = c.ddpfloattyp
	case ast.BIN_INDEX:
		switch lhsTyp {
		case c.ddpstring:
			c.latestReturn = c.cbb.NewCall(c.ddpstring.indexIrFun, lhs, rhs)
			c.latestReturnType = c.ddpchartyp
		default:
			if listType, isList := lhsTyp.(*ddpIrListType); isList {
				listLen := c.loadStructField(lhs, len_field_index)
				index := c.cbb.NewSub(rhs, newInt(1)) // ddp indices start at 1, so subtract 1
				// index bounds check
				cond := c.cbb.NewAnd(c.cbb.NewICmp(enum.IPredSLT, index, listLen), c.cbb.NewICmp(enum.IPredSGE, index, zero))
				c.createIfElese(cond, func() {
					listArr := c.loadStructField(lhs, arr_field_index)
					elementPtr := c.indexArray(listArr, index)
					if listType.elementType.IsPrimitive() { // primitives are simply loaded
						c.latestReturn = c.cbb.NewLoad(listType.elementType.IrType(), elementPtr)
					} else {
						dest := c.cbb.NewAlloca(listType.elementType.IrType())
						c.deepCopyInto(dest, elementPtr, listType.elementType)
						c.latestReturn = dest
					}
				}, func() { // runtime error
					c.out_of_bounds_error(rhs, listLen)
				})
				c.latestReturnType = listType.elementType
			} else {
				err("invalid Parameter Types for STELLE (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		}
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
				err("invalid Parameter Types for HOCH (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		case c.ddpfloattyp:
			switch rhsTyp {
			case c.ddpinttyp:
				rhs = c.cbb.NewSIToFP(rhs, ddpfloat)
			}
		default:
			err("invalid Parameter Types for HOCH (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
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
				err("invalid Parameter Types for LOGARITHMUS (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		case c.ddpfloattyp:
			switch rhsTyp {
			case c.ddpinttyp:
				rhs = c.cbb.NewSIToFP(rhs, ddpfloat)
			}
		default:
			err("invalid Parameter Types for LOGARITHMUS (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
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
		switch lhsTyp {
		case c.ddpinttyp, c.ddpbooltyp, c.ddpchartyp:
			c.latestReturn = c.cbb.NewICmp(enum.IPredEQ, lhs, rhs)
		case c.ddpfloattyp:
			c.latestReturn = c.cbb.NewFCmp(enum.FPredOEQ, lhs, rhs)
		default:
			c.latestReturn = c.cbb.NewCall(lhsTyp.EqualsFunc(), lhs, rhs)
		}
		c.latestReturnType = c.ddpbooltyp
	case ast.BIN_UNEQUAL:
		switch lhsTyp {
		case c.ddpinttyp, c.ddpbooltyp, c.ddpchartyp:
			c.latestReturn = c.cbb.NewICmp(enum.IPredNE, lhs, rhs)
		case c.ddpfloattyp:
			c.latestReturn = c.cbb.NewFCmp(enum.FPredONE, lhs, rhs)
		default:
			equal := c.cbb.NewCall(lhsTyp.EqualsFunc(), lhs, rhs)
			c.latestReturn = c.cbb.NewXor(equal, newInt(1))
		}
		c.latestReturnType = c.ddpbooltyp
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
				err("invalid Parameter Types for KLEINER (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		case c.ddpfloattyp:
			switch rhsTyp {
			case c.ddpinttyp:
				fp := c.cbb.NewSIToFP(rhs, ddpfloat)
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOLT, lhs, fp)
			case c.ddpfloattyp:
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOLT, lhs, rhs)
			default:
				err("invalid Parameter Types for KLEINER (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
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
				err("invalid Parameter Types for KLEINERODER (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		case c.ddpfloattyp:
			switch rhsTyp {
			case c.ddpinttyp:
				fp := c.cbb.NewSIToFP(rhs, ddpfloat)
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOLE, lhs, fp)
			case c.ddpfloattyp:
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOLE, lhs, rhs)
			default:
				err("invalid Parameter Types for KLEINERODER (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		default:
			err("invalid Parameter Types for KLEINERODER (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
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
				err("invalid Parameter Types for GRÖßER (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		case c.ddpfloattyp:
			switch rhsTyp {
			case c.ddpinttyp:
				fp := c.cbb.NewSIToFP(rhs, ddpfloat)
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOGT, lhs, fp)
			case c.ddpfloattyp:
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOGT, lhs, rhs)
			default:
				err("invalid Parameter Types for GRÖßER (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		default:
			err("invalid Parameter Types for GRÖßER (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
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
				err("invalid Parameter Types for GRÖßERODER (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		case c.ddpfloattyp:
			switch rhsTyp {
			case c.ddpinttyp:
				fp := c.cbb.NewSIToFP(rhs, ddpfloat)
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOGE, lhs, fp)
			case c.ddpfloattyp:
				c.latestReturn = c.cbb.NewFCmp(enum.FPredOGE, lhs, rhs)
			default:
				err("invalid Parameter Types for GRÖßERODER (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
			}
		default:
			err("invalid Parameter Types for GRÖßERODER (%s, %s)", lhsTyp.Name(), rhsTyp.Name())
		}
		c.latestReturnType = c.ddpbooltyp
	}
	c.freeNonPrimitive(lhs, lhsTyp)
	c.freeNonPrimitive(rhs, rhsTyp)
}
func (c *Compiler) VisitTernaryExpr(e *ast.TernaryExpr) {
	lhs, lhsTyp := c.evaluate(e.Lhs)
	mid, midTyp := c.evaluate(e.Mid)
	rhs, rhsTyp := c.evaluate(e.Rhs)

	switch e.Operator {
	case ast.TER_SLICE:
		dest := c.cbb.NewAlloca(lhsTyp.IrType())
		switch lhsTyp {
		case c.ddpstring:
			c.cbb.NewCall(c.ddpstring.sliceIrFun, dest, lhs, mid, rhs)
		default:
			if listTyp, isList := lhsTyp.(*ddpIrListType); isList {
				c.cbb.NewCall(listTyp.sliceIrFun, dest, lhs, mid, rhs)
			} else {
				err("invalid Parameter Types for VONBIS (%s, %s, %s)", lhsTyp.Name(), midTyp.Name(), rhsTyp.Name())
			}
		}
		c.latestReturn = dest
		c.latestReturnType = lhsTyp
	default:
		err("invalid Parameter Types for VONBIS (%s, %s, %s)", lhsTyp.Name(), midTyp.Name(), rhsTyp.Name())
	}

	c.freeNonPrimitive(lhs, lhsTyp)
	c.freeNonPrimitive(mid, midTyp)
	c.freeNonPrimitive(rhs, rhsTyp)
}
func (c *Compiler) VisitCastExpr(e *ast.CastExpr) {
	lhs, lhsTyp := c.evaluate(e.Lhs)
	if e.Type.IsList {
		listType := c.getListType(lhsTyp)
		list := c.cbb.NewAlloca(listType.typ)
		c.cbb.NewCall(listType.fromConstantsIrFun, list, newInt(1))
		elementPtr := c.indexArray(c.loadStructField(list, arr_field_index), zero)
		if !lhsTyp.IsPrimitive() {
			lhs = c.cbb.NewLoad(lhsTyp.IrType(), lhs)
		}
		c.cbb.NewStore(lhs, elementPtr)
		c.latestReturn = list
		c.latestReturnType = listType
		return // don't free lhs
	} else {
		switch e.Type.Primitive {
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
			default:
				err("invalid Parameter Type for ZAHL: %s", lhsTyp.Name())
			}
		case ddptypes.KOMMAZAHL:
			switch lhsTyp {
			case c.ddpinttyp:
				c.latestReturn = c.cbb.NewSIToFP(lhs, ddpfloat)
			case c.ddpfloattyp:
				c.latestReturn = lhs
			case c.ddpstring:
				c.latestReturn = c.cbb.NewCall(c.functions["ddp_string_to_float"].irFunc, lhs)
			default:
				err("invalid Parameter Type for KOMMAZAHL: %s", lhsTyp.Name())
			}
		case ddptypes.BOOLEAN:
			switch lhsTyp {
			case c.ddpinttyp:
				c.latestReturn = c.cbb.NewICmp(enum.IPredNE, lhs, zero)
			case c.ddpbooltyp:
				c.latestReturn = lhs
			default:
				err("invalid Parameter Type for BOOLEAN: %s", lhsTyp.Name())
			}
		case ddptypes.BUCHSTABE:
			switch lhsTyp {
			case c.ddpinttyp:
				c.latestReturn = c.cbb.NewTrunc(lhs, ddpchar)
			case c.ddpchartyp:
				c.latestReturn = lhs
			default:
				err("invalid Parameter Type for BUCHSTABE: %s", lhsTyp.Name())
			}
		case ddptypes.TEXT:
			if lhsTyp == c.ddpstring {
				c.latestReturn = lhs
				return // don't free lhs
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
				err("invalid Parameter Type for TEXT: %s", lhsTyp.Name())
			}
			dest := c.cbb.NewAlloca(c.ddpstring.typ)
			c.cbb.NewCall(to_string_func, dest, lhs)
			c.latestReturn = dest
		default:
			err("Invalide Typumwandlung zu %s", e.Type)
		}
	}
	c.latestReturnType = c.toIrType(e.Type)
	c.freeNonPrimitive(lhs, lhsTyp)
}
func (c *Compiler) VisitGrouping(e *ast.Grouping) {
	e.Expr.Accept(c) // visit like a normal expression, grouping is just precedence stuff which has already been parsed
}

// helper for VisitAssignStmt and VisitFuncCall
// if as_ref is true, the assignable is treated as a reference parameter and the third return value can be ignored
// if as_ref is false, the assignable is treated as the lhs in an AssignStmt and might be a string indexing
func (c *Compiler) evaluateAssignableOrReference(ass ast.Assigneable, as_ref bool) (value.Value, ddpIrType, *ast.Indexing) {
	switch assign := ass.(type) {
	case *ast.Ident:
		Var := c.scp.lookupVar(assign.Literal.Literal)
		return Var.val, Var.typ, nil
	case *ast.Indexing:
		lhs, lhsTyp, _ := c.evaluateAssignableOrReference(assign.Lhs, as_ref) // get the (possibly nested) assignable
		if listTyp, isList := lhsTyp.(*ddpIrListType); isList {
			index, _ := c.evaluate(assign.Index)
			index = c.cbb.NewSub(index, newInt(1)) // ddpindices start at 1
			listArr := c.loadStructField(lhs, arr_field_index)
			elementPtr := c.indexArray(listArr, index)
			return elementPtr, listTyp.elementType, nil
		} else if !as_ref && lhsTyp == c.ddpstring {
			return lhs, lhsTyp, assign
		} else {
			err("non-list/string type passed as assignable/reference")
		}
	}
	err("Invalid types in evaluateAssignableOrReference %s", ass)
	return nil, nil, nil
}
func (c *Compiler) VisitFuncCall(e *ast.FuncCall) {
	fun := c.functions[e.Name] // retreive the function (the resolver took care that it is present)
	args := make([]value.Value, 0, len(fun.funcDecl.ParamNames)+1)

	irReturnType := c.toIrType(fun.funcDecl.Type)
	var ret value.Value
	if !irReturnType.IsPrimitive() {
		ret = c.cbb.NewAlloca(irReturnType.IrType())
		args = append(args, ret)
	}

	for i, param := range fun.funcDecl.ParamNames {
		var val value.Value

		// differentiate between references and normal parameters
		if fun.funcDecl.ParamTypes[i].IsReference {
			if assign, ok := e.Args[param.Literal].(ast.Assigneable); ok {
				val, _, _ = c.evaluateAssignableOrReference(assign, true)
			} else {
				err("non-assignable passed as reference to %s", fun.funcDecl.Name)
			}
		} else {
			val, _ = c.evaluate(e.Args[param.Literal]) // compile each argument for the function
		}

		args = append(args, val) // add the value to the arguments
	}

	c.commentNode(c.cbb, e, "")
	// compile the actual function call
	if irReturnType.IsPrimitive() {
		c.latestReturn = c.cbb.NewCall(fun.irFunc, args...)
	} else {
		c.cbb.NewCall(fun.irFunc, args...)
		c.latestReturn = ret
	}
	c.latestReturnType = irReturnType

	// the arguments of external functions must be freed by the caller
	// normal functions free their parameters in their body
	if ast.IsExternFunc(fun.funcDecl) {
		for i := range fun.funcDecl.ParamNames {
			if !fun.funcDecl.ParamTypes[i].IsReference {
				c.freeNonPrimitive(args[i], c.toIrType(fun.funcDecl.ParamTypes[i].Type))
			}
		}
	}
}

// should have been filtered by the resolver/typechecker, so err
func (c *Compiler) VisitBadStmt(s *ast.BadStmt) {
	err("Es wurde eine invalide Aussage gefunden")
}
func (c *Compiler) VisitDeclStmt(s *ast.DeclStmt) {
	s.Decl.Accept(c)
}
func (c *Compiler) VisitExprStmt(s *ast.ExprStmt) {
	expr, exprTyp := c.evaluate(s.Expr)
	c.freeNonPrimitive(expr, exprTyp)
}
func (c *Compiler) VisitAssignStmt(s *ast.AssignStmt) {
	rhs, rhsTyp := c.evaluate(s.Rhs) // compile the expression

	lhs, lhsTyp, lhsStringIndexing := c.evaluateAssignableOrReference(s.Var, false)

	if lhsStringIndexing != nil {
		index, _ := c.evaluate(lhsStringIndexing.Index)
		c.cbb.NewCall(c.ddpstring.replaceCharIrFun, lhs, rhs, index)
	} else {
		c.freeNonPrimitive(lhs, lhsTyp) // free the old value in the variable/list
		if !lhsTyp.IsPrimitive() {
			rhs = c.cbb.NewLoad(rhsTyp.IrType(), rhs)
		}
		c.cbb.NewStore(rhs, lhs) // store the new value
	}
}
func (c *Compiler) VisitBlockStmt(s *ast.BlockStmt) {
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
}

// for info on how the generated ir works you might want to see https://llir.github.io/document/user-guide/control/#If
func (c *Compiler) VisitIfStmt(s *ast.IfStmt) {
	cond, _ := c.evaluate(s.Condition)
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
}

// for info on how the generated ir works you might want to see https://llir.github.io/document/user-guide/control/#Loop
func (c *Compiler) VisitWhileStmt(s *ast.WhileStmt) {
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

		c.cbb, c.scp = condBlock, c.exitScope(c.scp) // the condition is not in scope
		cond, _ := c.evaluate(s.Condition)
		leaveBlock := c.cf.NewBlock("")
		c.commentNode(c.cbb, s, "")
		c.cbb.NewCondBr(cond, body, leaveBlock)

		c.cbb = leaveBlock
	case token.WIEDERHOLE:
		counter := c.cf.Blocks[0].NewAlloca(ddpint)
		cond, _ := c.evaluate(s.Condition)
		c.cbb.NewStore(cond, counter)
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
}

// for info on how the generated ir works you might want to see https://llir.github.io/document/user-guide/control/#Loop
func (c *Compiler) VisitForStmt(s *ast.ForStmt) {
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
	indexVar := incrementBlock.NewLoad(Var.typ.IrType(), Var.val)
	var incrementer value.Value // Schrittgröße
	// if no stepsize was present it is 1
	if s.StepSize == nil {
		incrementer = newInt(1)
	} else { // stepsize was present, so compile it
		c.cbb = incrementBlock
		incrementer, _ = c.evaluate(s.StepSize)
	}

	c.cbb = incrementBlock
	// add the incrementer to the counter variable
	add := c.cbb.NewAdd(indexVar, incrementer)
	c.cbb.NewStore(add, c.scp.lookupVar(s.Initializer.Name.Literal).val)
	c.commentNode(c.cbb, s, "")
	c.cbb.NewBr(condBlock) // check the condition (loop)

	// finally compile the condition block(s)
	initGreaterTo := c.cf.NewBlock("")
	c.comment("initGreaterTo", initGreaterTo)
	initLessthenTo := c.cf.NewBlock("")
	c.comment("initLessthenTo", initLessthenTo)
	leaveBlock := c.cf.NewBlock("") // after the condition is false we jump to the leaveBlock
	c.comment("forLeaveBlock", leaveBlock)

	c.cbb = condBlock
	// we check the counter differently depending on wether or not we are looping up or down (positive vs negative stepsize)
	to, _ := c.evaluate(s.To)
	cond := c.cbb.NewICmp(enum.IPredSLE, initValue, to)
	c.commentNode(c.cbb, s, "")
	c.cbb.NewCondBr(cond, initLessthenTo, initGreaterTo)

	c.cbb = initLessthenTo
	// we are counting up, so compare less-or-equal
	to, _ = c.evaluate(s.To)
	cond = c.cbb.NewICmp(enum.IPredSLE, c.cbb.NewLoad(Var.typ.IrType(), Var.val), to)
	c.commentNode(c.cbb, s, "")
	c.cbb.NewCondBr(cond, forBody, leaveBlock)

	c.cbb = initGreaterTo
	// we are counting down, so compare greater-or-equal
	to, _ = c.evaluate(s.To)
	cond = c.cbb.NewICmp(enum.IPredSGE, c.cbb.NewLoad(Var.typ.IrType(), Var.val), to)
	c.commentNode(c.cbb, s, "")
	c.cbb.NewCondBr(cond, forBody, leaveBlock)

	c.cbb, c.scp = leaveBlock, c.exitScope(c.scp) // leave the scopee
}
func (c *Compiler) VisitForRangeStmt(s *ast.ForRangeStmt) {
	c.scp = newScope(c.scp)
	in, inTyp := c.evaluate(s.In)
	c.scp.addNonPrimitive(in, inTyp)

	var len value.Value
	if inTyp == c.ddpstring {
		len = c.cbb.NewCall(c.ddpstring.lengthIrFun, in)
	} else {
		len = c.loadStructField(in, len_field_index)
	}
	loopStart, condBlock, bodyBlock, incrementBlock, leaveBlock := c.cf.NewBlock(""), c.cf.NewBlock(""), c.cf.NewBlock(""), c.cf.NewBlock(""), c.cf.NewBlock("")
	c.cbb.NewCondBr(c.cbb.NewICmp(enum.IPredEQ, len, zero), leaveBlock, loopStart)

	c.cbb = loopStart
	index := c.cf.Blocks[0].NewAlloca(ddpint)
	c.cbb.NewStore(newInt(1), index)
	irType := c.toIrType(s.Initializer.Type)
	c.scp.addVar(s.Initializer.Name.Literal, c.cf.Blocks[0].NewAlloca(irType.IrType()), irType, false)
	c.cbb.NewBr(condBlock)

	c.cbb = condBlock
	c.cbb.NewCondBr(c.cbb.NewICmp(enum.IPredSLE, c.cbb.NewLoad(ddpint, index), len), bodyBlock, leaveBlock)

	c.cbb = bodyBlock
	loopVar := c.scp.lookupVar(s.Initializer.Name.Literal)
	if inTyp == c.ddpstring {
		char := c.cbb.NewCall(c.ddpstring.indexIrFun, in, c.cbb.NewLoad(ddpint, index))
		c.cbb.NewStore(char, loopVar.val)
	} else {
		arr := c.loadStructField(in, arr_field_index)
		ddpindex := c.cbb.NewSub(c.cbb.NewLoad(ddpint, index), newInt(1))
		elementPtr := c.indexArray(arr, ddpindex)
		inListTyp := inTyp.(*ddpIrListType)
		if inListTyp.elementType.IsPrimitive() {
			element := c.cbb.NewLoad(inListTyp.elementType.IrType(), elementPtr)
			c.cbb.NewStore(element, loopVar.val)
		} else {
			c.deepCopyInto(loopVar.val, elementPtr, inListTyp.elementType)
		}
	}
	c.visitNode(s.Body)
	if c.cbb.Term == nil {
		c.cbb.NewBr(incrementBlock)
	}

	c.cbb = incrementBlock
	c.cbb.NewStore(c.cbb.NewAdd(c.cbb.NewLoad(ddpint, index), newInt(1)), index)
	c.cbb.NewBr(condBlock)

	c.cbb, c.scp = leaveBlock, c.exitScope(c.scp)
	c.freeNonPrimitive(in, inTyp)
}
func (c *Compiler) VisitReturnStmt(s *ast.ReturnStmt) {
	if s.Value == nil {
		c.exitNestedScopes()
		c.commentNode(c.cbb, s, "")
		c.cbb.NewRet(nil)
		return
	}
	val, valTyp := c.evaluate(s.Value)
	c.exitNestedScopes()
	c.commentNode(c.cbb, s, "")
	if valTyp.IsPrimitive() {
		c.cbb.NewRet(val)
	} else {
		c.cbb.NewStore(c.cbb.NewLoad(valTyp.IrType(), val), c.cf.Params[0])
		c.cbb.NewRet(nil)
	}
}

func (c *Compiler) exitNestedScopes() {
	for scp := c.scp; scp != c.cfscp; scp = c.exitScope(scp) {
		for _, v := range scp.non_primitives {
			c.freeNonPrimitive(v.val, v.typ)
		}
	}
	c.exitScope(c.cfscp)
}
