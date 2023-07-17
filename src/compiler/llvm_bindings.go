package compiler

import (
	"io"

	"github.com/bafto/Go-LLVM-Bindings/llvm"
)

func init() {
	llvm.InitializeAllTargetInfos()
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllAsmParsers()
	llvm.InitializeAllAsmPrinters()
}

// context which contains
// all variables needed
// to use llvm
// needs to be disposed after use
type llvmContext struct {
	targetMachine llvm.TargetMachine
	targetData    llvm.TargetData
	passManager   llvm.PassManager
}

func newllvmContext() (llctx *llvmContext, err error) {
	llctx = &llvmContext{}
	target, err := llvm.GetTargetFromTriple(llvm.DefaultTargetTriple())
	if err != nil {
		return nil, err
	}

	llctx.targetMachine = target.CreateTargetMachine(
		llvm.DefaultTargetTriple(),
		"generic",
		"",
		llvm.CodeGenOptLevel(llvm.CodeGenLevelDefault),
		llvm.RelocMode(llvm.RelocDynamicNoPic), // TODO: find documentation about what this is
		llvm.CodeModel(llvm.CodeModelDefault),
	)

	llctx.targetData = llctx.targetMachine.CreateTargetData()

	llctx.passManager = llvm.NewPassManager()
	llctx.passManager.AddInstructionCombiningPass()
	llctx.passManager.AddLoopDeletionPass()
	llctx.passManager.AddLoopUnrollPass()
	llctx.passManager.AddStripDeadPrototypesPass()
	llctx.passManager.AddPromoteMemoryToRegisterPass()
	llctx.passManager.AddAggressiveDCEPass()
	llctx.passManager.AddArgumentPromotionPass()
	llctx.passManager.AddCFGSimplificationPass()
	llctx.passManager.AddConstantMergePass()
	llctx.passManager.AddDeadArgEliminationPass()
	llctx.passManager.AddDeadStoreEliminationPass()
	llctx.passManager.AddFunctionInliningPass()
	llctx.passManager.AddFunctionAttrsPass()
	llctx.passManager.AddGlobalDCEPass()
	llctx.passManager.AddGlobalOptimizerPass()
	llctx.passManager.AddIndVarSimplifyPass()

	llctx.targetMachine.AddAnalysisPasses(llctx.passManager)

	return llctx, nil
}

func (llctx *llvmContext) Dispose() {
	llctx.targetMachine.Dispose()
	llctx.targetData.Dispose()
	llctx.passManager.Dispose()
}

// parses a .ll file and returns the module and context
// both need to be disposed in case of success
func (llctx *llvmContext) parseLLFile(path string) (llvm.Module, llvm.Context, error) {
	ctx := llvm.NewContext()

	mod, err := llvm.ParseIRFile(path, ctx)
	if err != nil {
		ctx.Dispose()
		return llvm.Module{}, llvm.Context{}, err
	}

	mod.SetDataLayout(llctx.targetData.String())
	mod.SetTarget(llctx.targetMachine.Triple())

	return mod, ctx, nil
}

// optimizes the given module and returns wether it was modified
func (llctx *llvmContext) optimizeModule(mod llvm.Module) bool {
	return llctx.passManager.Run(mod)
}

// compiles the module to w and returns w.Write
func (llctx *llvmContext) compileModule(mod llvm.Module, fileType llvm.CodeGenFileType, w io.Writer) (int, error) {
	memBuffer, err := llctx.targetMachine.EmitToMemoryBuffer(mod, fileType)
	if err != nil {
		return 0, err
	}
	defer memBuffer.Dispose()

	return w.Write(memBuffer.Bytes())
}

// in case of failure, dest should not be used
func llvmLinkAllModules(dest llvm.Module, sources []llvm.Module) error {
	for _, src := range sources {
		if err := llvm.LinkModules(dest, src); err != nil {
			return err
		}
	}
	return nil
}
