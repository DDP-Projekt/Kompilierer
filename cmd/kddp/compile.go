package main

import (
	"os"
	"path/filepath"

	"github.com/bafto/Go-LLVM-Bindings/llvm"
)

// takes a .ll file and compiles it into a .o .obj .s or .asm file depending on the outputFile extension
func compileToObject(inputFile, outputFile string) error {
	fileExtension := filepath.Ext(outputFile)

	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod, err := llvm.ParseIRFile(inputFile, ctx)
	if err != nil {
		return err
	}
	defer mod.Dispose()

	llvm.InitializeAllTargetInfos()
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllAsmParsers()
	llvm.InitializeAllAsmPrinters()

	target, err := llvm.GetTargetFromTriple(llvm.DefaultTargetTriple())
	if err != nil {
		return err
	}
	targetMachine := target.CreateTargetMachine(
		llvm.DefaultTargetTriple(),
		"generic",
		"",
		llvm.CodeGenOptLevel(llvm.CodeGenLevelDefault),
		llvm.RelocMode(llvm.RelocDefault),
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
	targetMachine.AddAnalysisPasses(pass)
	pass.Run(mod)

	var fileType llvm.CodeGenFileType
	switch fileExtension {
	case ".s", ".asm":
		fileType = llvm.CodeGenFileType(llvm.AssemblyFile)
	case ".o", ".obj":
		fileType = llvm.CodeGenFileType(llvm.ObjectFile)
	}

	memBuffer, err := targetMachine.EmitToMemoryBuffer(mod, fileType)
	if err != nil {
		return err
	}
	defer memBuffer.Dispose()

	return os.WriteFile(outputFile, memBuffer.Bytes(), os.ModePerm)
}
