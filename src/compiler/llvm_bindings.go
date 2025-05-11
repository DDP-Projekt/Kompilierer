package compiler

import (
	"fmt"
	"io"
	"os"

	"github.com/DDP-Projekt/Kompilierer/src/ddppath"
	"github.com/DDP-Projekt/Kompilierer/src/compiler/llvm"
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
type llvmModuleContext struct {
	llTarget
	llctx llvm.Context
	llmod llvm.Module
}

func newllvmModuleContext(moduleName string) (llctx llvmModuleContext, err error) {
	target, err := newllvmTarget()
	if err != nil {
		return llvmModuleContext{}, err
	}

	llctx = llvmModuleContext{}
	llctx.llTarget = *target

	llctx.llctx = llvm.NewContext()
	llctx.llmod = llctx.llctx.NewModule(moduleName)
	llctx.llmod.SetDataLayout(llctx.llTargetData.String())
	llctx.llmod.SetTarget(llctx.llTargetMachine.Triple())

	return llctx, nil
}

func newllvmModuleContextFromIR(name string, llvm_ir []byte) (llvmModuleContext, error) {
	target, err := newllvmTarget()
	if err != nil {
		return llvmModuleContext{}, err
	}

	llctx := llvmModuleContext{}
	llctx.llTarget = *target

	llctx.llctx = llvm.NewContext()

	buf := llvm.NewMemoryBufferFromRangeCopy(llvm_ir)

	if llctx.llmod, err = llctx.llctx.ParseIR(buf); err != nil {
		return llctx, fmt.Errorf("could not parse llvm ir: %w", err)
	}

	llctx.llmod.SetDataLayout(llctx.llTargetData.String())
	llctx.llmod.SetTarget(llctx.llTargetMachine.Triple())

	return llctx, nil
}

func (llctx *llvmModuleContext) Dispose() {
	llctx.llTarget.Dispose()
	llctx.llmod.Dispose()
	llctx.llctx.Dispose()
}

func newllvmModuleContextFromListDefs() (llvmModuleContext, error) {
	list_defs_ir, err := os.ReadFile(ddppath.DDP_List_Types_Defs_LL)
	if err != nil {
		return llvmModuleContext{}, fmt.Errorf("could not read list_defs: %w", err)
	}
	return newllvmModuleContextFromIR(ddppath.DDP_List_Types_Defs_LL, list_defs_ir)
}

// optimizes the given module and returns any error
func (llctx *llvmModuleContext) optimizeModule(mod llvm.Module) error {
	return llctx.llmod.RunPasses("default<O2>", llctx.llTargetMachine, llvm.NewPassBuilderOptions())
}

// compiles the module to w and returns w.Write
func (llctx *llvmModuleContext) compileModule(fileType llvm.CodeGenFileType, w io.Writer) (int, error) {
	memBuffer, err := llctx.llTargetMachine.EmitToMemoryBuffer(llctx.llmod, fileType)
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
