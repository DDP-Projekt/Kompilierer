package compiler

import (
	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/compiler/llvm"
)

// holds variables to build a single function
type llBuilder struct {
	*compiler
	llvm.Builder
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
