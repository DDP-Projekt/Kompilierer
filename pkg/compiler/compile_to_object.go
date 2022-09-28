package compiler

import (
	"io"

	"github.com/bafto/Go-LLVM-Bindings/llvm"
)

// takes a .ll file and compiles it into the given OutputType
// and writes the result to to
// mainly copied from the llvm documentation
func compileToObject(inputFile string, outType OutputType, to io.Writer) (int, error) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()

	mod, err := llvm.ParseIRFile(inputFile, ctx)
	if err != nil {
		return 0, err
	}
	defer mod.Dispose()

	llvm.InitializeAllTargetInfos()
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllAsmParsers()
	llvm.InitializeAllAsmPrinters()

	target, err := llvm.GetTargetFromTriple(llvm.DefaultTargetTriple())
	if err != nil {
		return 0, err
	}

	targetMachine := target.CreateTargetMachine(
		llvm.DefaultTargetTriple(),
		"generic",
		"",
		llvm.CodeGenOptLevel(llvm.CodeGenLevelDefault),
		llvm.RelocMode(llvm.RelocDynamicNoPic), // TODO: find documentation about what this is
		llvm.CodeModel(llvm.CodeModelDefault),
	)
	defer targetMachine.Dispose()

	targetData := targetMachine.CreateTargetData()
	defer targetData.Dispose()

	mod.SetDataLayout(targetData.String())
	mod.SetTarget(targetMachine.Triple())

	pass := llvm.NewPassManager()
	defer pass.Dispose()

	// dunno if they do something, but who knows
	pass.AddInstructionCombiningPass()
	pass.AddLoopDeletionPass()
	pass.AddLoopUnrollPass()
	pass.AddStripDeadPrototypesPass()
	pass.AddPromoteMemoryToRegisterPass() // promote as many allocas as possible to registers
	targetMachine.AddAnalysisPasses(pass)
	pass.Run(mod)

	var fileType llvm.CodeGenFileType
	switch outType {
	case OutputAsm:
		fileType = llvm.CodeGenFileType(llvm.AssemblyFile)
	case OutputObj:
		fileType = llvm.CodeGenFileType(llvm.ObjectFile)
	}

	memBuffer, err := targetMachine.EmitToMemoryBuffer(mod, fileType)
	if err != nil {
		return 0, err
	}
	defer memBuffer.Dispose()

	return to.Write(memBuffer.Bytes())
}
