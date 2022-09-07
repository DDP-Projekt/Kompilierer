package tests

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func BenchmarkAll(b *testing.B) {
	err := filepath.WalkDir("./benchmarks", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			b.Errorf("Error walking %s: %s\nskipping this directory", path, err)
			return fs.SkipDir
		}
		if !d.IsDir() {
			return nil
		}
		if d.Name() == "benchmarks" {
			return nil
		}

		return runBenchmark(path, d, b)
	})

	if err != nil {
		b.Errorf("Error walking the test directory: %s", err)
	}
}

func runBenchmark(path string, d fs.DirEntry, b *testing.B) error {
	ddpPath := filepath.Join(path, d.Name()) + ".ddp"
	exePath := changeExtension(ddpPath, ".exe")

	compileCmd := exec.Command("../build/kddp", "build", ddpPath)
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
}
