package compiler

import (
	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/compiler/llvm"
)

// holds variables to build a single function
type llBuilder struct {
	llvm.Builder
	fnName   string
	c        *compiler
	llFnType llvm.Type       // the llvm type of the function
	llFn     llvm.Value      // the llvm function value
	ddpDecl  *ast.FuncDecl   // the ddp decl for the current function
	cb       llvm.BasicBlock // current block
	scp      *scope          // current scope in the ast (not in the ir)
	fnScope  *scope

	latestReturn     llvm.Value // return of the latest evaluated expression (in the ir)
	latestReturnType ddpIrType  // the type of latestReturn
	latestIsTemp     bool       // ewther the latestReturn is a temporary or not
	currentNode      ast.Node   // used for error reporting

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

func (c *compiler) newBuilder(funcName string, funcType llvm.Type) *llBuilder {
	builder := &llBuilder{
		fnName:  funcName,
		c:       c,
		Builder: c.llctx.NewBuilder(),
		scp:     newScope(nil),
	}

	builder.llFnType = funcType
	builder.llFn = llvm.AddFunction(c.llmod, funcName, builder.llFnType)
	builder.llFn.SetFunctionCallConv(llvm.CCallConv)
	builder.cb = builder.newBlock()
	builder.SetInsertPointAtEnd(builder.cb)

	c.builderStack = append(c.builderStack, builder)
	return builder
}

func (c *compiler) pushBuilder(b *llBuilder) *llBuilder {
	c.builderStack = append(c.builderStack, b)
	return b
}

func (c *compiler) disposeAndPop() {
	c.builderStack[len(c.builderStack)].Dispose()
	c.builderStack = c.builderStack[:len(c.builderStack)-1]
}

func (c *compiler) builder() *llBuilder {
	return c.builderStack[len(c.builderStack)-1]
}
