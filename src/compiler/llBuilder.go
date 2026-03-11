package compiler

import (
	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/compiler/llvm"
)

type funcParam struct {
	name string
	typ  llvm.Type
	val  llvm.Value
}

// holds variables to build a single function
type llBuilder struct {
	llvm.Builder
	fnName   string
	c        *compiler
	llFnType llvm.Type       // the llvm type of the function
	llFn     llvm.Value      // the llvm function value
	ddpDecl  *ast.FuncDecl   // the ddp decl for the current function
	cb       llvm.BasicBlock // current block
	params   []funcParam

	scp     *scope // current scope in the ast (not in the ir)
	fnScope *scope

	latestReturn ddpValue // return of the latest evaluated expression (in the ir)
	currentNode  ast.Node // used for error reporting

	curLeaveBlock    llvm.BasicBlock // leave block of the current loop
	curContinueBlock llvm.BasicBlock // block where a continue should jump to
	curLoopScope     *scope          // scope of the current loop for break/continue to free to
}

func (b *llBuilder) newBlock() llvm.BasicBlock {
	return b.c.llctx.AddBasicBlock(b.llFn, "")
}

func (b *llBuilder) setBlock(bb llvm.BasicBlock) {
	b.cb = bb
	b.SetInsertPointAtEnd(bb)
}

func (b *llBuilder) isDDPMain() bool {
	return !b.llFn.IsNil()
}

func (b *llBuilder) withBlock(block llvm.BasicBlock, do func()) {
	cb := b.cb
	b.setBlock(block)
	do()
	b.setBlock(cb)
}

// calculates all needed gc-live values and their re-store locations
func (b *llBuilder) getLiveValues() ([]llvm.Value, []llvm.Value) {
	// TODO: maybe remove this check and ensure a scope is always correct
	// here, because when calling the module dispose function, the scp is nil
	if b.scp == nil {
		return nil, nil
	}

	liveValues := make([]llvm.Value, 0, len(b.scp.temporaries)+len(b.scp.variables))
	restoreLocations := make([]llvm.Value, 0, len(b.scp.temporaries)+len(b.scp.variables))

	getLiveValuesForScope := func(scp *scope) {
		if scp == nil {
			return
		}

		// globals are already tracked
		if !scp.isGlobalScope() {
			for _, Var := range scp.variables {
				live, restores := Var.typ.LoadLivesAndRestores(b.c, Var.val)
				liveValues = append(liveValues, live...)
				restoreLocations = append(restoreLocations, restores...)
			}
		}

		for _, temp := range scp.temporaries {
			if temp.typ == nil {
				liveValues = append(liveValues, temp.val)
				restoreLocations = append(restoreLocations, temp.restoreLoc)
			} else {
				live, restores := temp.typ.LoadLivesAndRestores(b.c, temp.val)
				liveValues = append(liveValues, live...)
				restoreLocations = append(restoreLocations, restores...)
			}
		}
	}

	for scp := b.scp; scp != b.fnScope; scp = scp.enclosing {
		getLiveValuesForScope(scp)
	}
	getLiveValuesForScope(b.fnScope)

	return liveValues, restoreLocations
}

func (b *llBuilder) createCall(fn llvm.Value, args ...llvm.Value) llvm.Value {
	if fn.GC() == DDP_GC_STRATEGY_NAME {
		liveValues, restores := b.getLiveValues()
		gcLive := llvm.CreateOperandBundle("gc-live", liveValues)
		defer gcLive.Dispose()

		// TODO: re-store or otherwise use the relocated values
		call := b.createCallWithOperandBundles(fn, []llvm.OperandBundle{gcLive}, args...)

		for i, relocated := range liveValues {
			if restores[i] != (llvm.Value{}) {
				b.CreateStore(relocated, restores[i])
			}
		}

		return call
	}

	return b.CreateCall(fn.GlobalValueType(), fn, args, "")
}

func (b *llBuilder) createCallWithOperandBundles(fn llvm.Value, operandBundles []llvm.OperandBundle, args ...llvm.Value) llvm.Value {
	return b.CreateCallWithOperandBundle(fn.GlobalValueType(), fn, args, operandBundles, "")
}

const DDP_GC_STRATEGY_NAME = "ddp-gc"

func (c *compiler) createBuilder(funcName string, funcType llvm.Type, funcAttributes []llvm.Attribute, paramNames []string, paramAttributes [][]llvm.Attribute, scp *scope, isGC bool, declarationOnly bool) *llBuilder {
	if scp == nil {
		scp = newScope(nil) // separate "global" scope
	}

	builder := &llBuilder{
		fnName:  funcName,
		c:       c,
		Builder: c.llctx.NewBuilder(),
		scp:     scp,
		fnScope: scp,
	}

	builder.llFnType = funcType
	builder.llFn = llvm.AddFunction(c.llmod, funcName, builder.llFnType)
	builder.llFn.SetFunctionCallConv(llvm.CCallConv) // every function is called with the c calling convention to make interaction with inbuilt stuff easier
	// builder.llFn.AddFunctionAttr(c.attr_nounwind)

	if isGC {
		builder.llFn.SetGC(DDP_GC_STRATEGY_NAME)
	}

	for _, attr := range funcAttributes {
		builder.llFn.AddFunctionAttr(attr)
	}

	for i, attrs := range paramAttributes {
		for _, attr := range attrs {
			builder.llFn.AddAttributeAtIndex(i+1, attr)
		}
	}

	if !declarationOnly {
		builder.cb = builder.newBlock()
		builder.SetInsertPointAtEnd(builder.cb)
	}
	for i, param := range builder.llFn.Params() {
		builder.params = append(builder.params, funcParam{name: paramNames[i], typ: param.Type(), val: param})
	}

	return builder
}

func (c *compiler) pushNewBuilder(funcName string, funcType llvm.Type, funcAttributes []llvm.Attribute, paramNames []string, paramAttributes [][]llvm.Attribute, scp *scope, isGC bool, declarationOnly bool) *llBuilder {
	builder := c.createBuilder(funcName, funcType, funcAttributes, paramNames, paramAttributes, scp, isGC, declarationOnly)
	c.builderStack = append(c.builderStack, builder)
	return builder
}

func (c *compiler) pushBuilder(b *llBuilder) *llBuilder {
	c.builderStack = append(c.builderStack, b)
	return b
}

func (c *compiler) disposeAndPop() {
	c.popBuilder().Dispose()
}

func (c *compiler) popBuilder() (builder *llBuilder) {
	if len(c.builderStack) > 0 {
		builder = c.builder()
		c.builderStack = c.builderStack[:len(c.builderStack)-1]
	}
	return
}

func (c *compiler) builder() *llBuilder {
	return c.builderStack[len(c.builderStack)-1]
}
