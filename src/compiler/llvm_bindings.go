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
	context       llvm.Context
}

func newllvmContext() (llctx *llvmContext, err error) {
	llctx = &llvmContext{}

	llctx.context = llvm.NewContext()

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
	llctx.context.Dispose()
	llctx.targetMachine.Dispose()
	llctx.targetData.Dispose()
	llctx.passManager.Dispose()
}

// parses a .ll file and returns the module in with llctx.Context as c ontext
// the module needs to be disposed
func (llctx *llvmContext) parseIR(llvm_ir []byte) (llvm.Module, error) {
	buf := llvm.NewMemoryBufferFromRangeCopy(llvm_ir)

	mod, err := llvm.ParseIRFromMemoryBuffer(buf, llctx.context)
	if err != nil {
		return llvm.Module{}, err
	}

	mod.SetDataLayout(llctx.targetData.String())
	mod.SetTarget(llctx.targetMachine.Triple())

	return mod, nil
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

// links all sources into dest, destroying them
// in case of failure, all sources that were not used yet are disposed
// and dest should not be used
func llvmLinkAllModules(dest llvm.Module, sources []llvm.Module) error {
	for i, src := range sources {
		if err := llvm.LinkModules(dest, src); err != nil {
			for _, mod := range sources[i+1:] {
				mod.Dispose()
			}
			return err
		}
	}
	return nil
}
