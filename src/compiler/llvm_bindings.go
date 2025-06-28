package compiler

import (
	"fmt"
	"io"
	"os"

	"github.com/DDP-Projekt/Kompilierer/src/compiler/llvm"
	"github.com/DDP-Projekt/Kompilierer/src/ddppath"
)

func init() {
	llvm.InitializeAllTargetInfos()
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllAsmParsers()
	llvm.InitializeAllAsmPrinters()
}

type llTarget struct {
	llTargetMachine llvm.TargetMachine
	llTargetData    llvm.TargetData
}

func newllvmTarget() (*llTarget, error) {
	newTarget, err := llvm.GetTargetFromTriple(llvm.DefaultTargetTriple())
	if err != nil {
		return nil, fmt.Errorf("could not create llvm target: %w", err)
	}

	targetMachine := newTarget.CreateTargetMachine(
		llvm.DefaultTargetTriple(),
		"generic",
		"",
		llvm.CodeGenOptLevel(llvm.CodeGenLevelDefault),
		llvm.RelocMode(llvm.RelocDynamicNoPic), // TODO: find documentation about what this is
		llvm.CodeModel(llvm.CodeModelDefault),
	)

	targetData := targetMachine.CreateTargetData()

	return &llTarget{
		llTargetMachine: targetMachine,
		llTargetData:    targetData,
	}, nil
}

func (target *llTarget) Dispose() {
	target.llTargetMachine.Dispose()
	target.llTargetData.Dispose()
}

// context which contains
// all variables needed
// to use llvm
// needs to be disposed after use
type llvmTargetContext struct {
	llTarget
	llctx llvm.Context
}

func (llctx *llvmTargetContext) newModule(name string) llvm.Module {
	mod := llctx.llctx.NewModule(name)
	mod.SetDataLayout(llctx.llTargetData.String())
	mod.SetTarget(llctx.llTargetMachine.Triple())
	return mod
}

func newllvmModuleContext(moduleName string) (llctx llvmTargetContext, err error) {
	target, err := newllvmTarget()
	if err != nil {
		return llvmTargetContext{}, err
	}

	llctx = llvmTargetContext{}
	llctx.llTarget = *target

	llctx.llctx = llvm.NewContext()

	return llctx, nil
}

func (llctx *llvmTargetContext) parseIR(llvm_ir []byte) (llvm.Module, error) {
	buf := llvm.NewMemoryBufferFromRangeCopy(llvm_ir)

	mod, err := llctx.llctx.ParseIR(buf)
	if err != nil {
		return llvm.Module{}, fmt.Errorf("could not parse llvm ir: %w", err)
	}

	mod.SetDataLayout(llctx.llTargetData.String())
	mod.SetTarget(llctx.llTargetMachine.Triple())

	return mod, nil
}

func (llctx *llvmTargetContext) Dispose() {
	llctx.llTarget.Dispose()
	llctx.llctx.Dispose()
}

func parseListDefsIntoContext(llctx *llvmTargetContext) (llvm.Module, error) {
	list_defs_ir, err := os.ReadFile(ddppath.DDP_List_Types_Defs_LL)
	if err != nil {
		return llvm.Module{}, fmt.Errorf("could not read list_defs: %w", err)
	}
	return llctx.parseIR(list_defs_ir)
}

// optimizes the given module and returns any error
func (llctx *llvmTargetContext) optimizeModule(mod llvm.Module) error {
	options := llvm.NewPassBuilderOptions()
	options.SetCallGraphProfile(true)
	options.SetMergeFunctions(true)
	options.SetLoopUnrolling(true)
	options.SetLoopInterleaving(true)
	options.SetLoopVectorization(true)
	options.SetSLPVectorization(true)

	defer options.Dispose()
	// options.SetVerifyEach(true) // TODO: only do this in debug mode as it is expensive
	return mod.RunPasses("default<O2>", llctx.llTargetMachine, options)
}

// compiles the module to w and returns w.Write
func (llctx *llvmTargetContext) compileModule(mod llvm.Module, fileType llvm.CodeGenFileType, w io.Writer) (int, error) {
	memBuffer, err := llctx.llTargetMachine.EmitToMemoryBuffer(mod, fileType)
	if err != nil {
		return 0, fmt.Errorf("could not compile module to memory buffer: %w", err)
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
