package tests

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/DDP-Projekt/Kompilierer/pkg/ddperror"
	"github.com/DDP-Projekt/Kompilierer/pkg/parser"
)

type BenchmarkFunction func(path string, d fs.DirEntry, b *testing.B) error

// helper
func walkSubDirs(root string, b *testing.B, benchFunc BenchmarkFunction) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			b.Errorf("Error walking %s: %s\nskipping this directory", path, err)
			return fs.SkipDir
		}
		if !d.IsDir() {
			return nil
		}
		if d.Name() == filepath.Base(root) {
			return nil
		}

		return benchFunc(path, d, b)
	})
}

func BenchmarkExecutables(b *testing.B) {
	err := walkSubDirs("./benchmarks/executables", b, func(path string, d fs.DirEntry, b *testing.B) error {
		ddpPath := filepath.Join(path, d.Name()) + ".ddp"
		exePath := changeExtension(ddpPath, ".exe")

		compileCmd := exec.Command("../build/DDP/bin/kddp", "kompiliere", ddpPath)
		if out, err := compileCmd.CombinedOutput(); err != nil {
			b.Errorf("Error compiling %s: %s\noutput:\n%s", ddpPath, err, out)
			return err
		}
		defer os.Remove(exePath)

		b.Run(d.Name(), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				benchmarkCmd := exec.Command(exePath)
				if out, err := benchmarkCmd.CombinedOutput(); err != nil {
					b.Errorf("Error running %s: %s\noutput: %s\n", exePath, err, out)
				}
			}
		})

		return nil
	})

	if err != nil {
		b.Errorf("Error walking the test directory: %s", err)
	}
}

func BenchmarkParser(b *testing.B) {
	err := walkSubDirs("./benchmarks/parser", b, func(path string, d fs.DirEntry, b *testing.B) error {
		ddpPath := filepath.Join(path, d.Name()) + ".ddp"

		b.Run(d.Name(), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				Ast, err := parser.ParseFile(ddpPath, func(err ddperror.Error) {
					b.Errorf("Parser error: [%s %d:%d] %s", err.File(), err.GetRange().Start.Line, err.GetRange().Start.Column, err.Msg())
				})
				if err != nil {
					b.Errorf("Parser returned an error: %s", err)
				}
				if Ast.Faulty {
					b.Error("Faulty Ast")
				}
			}
		})

		return nil
	})

	if err != nil {
		b.Errorf("Error walking the test directory: %s", err)
	}
}
