package tests

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestKDDP(t *testing.T) {
	err := filepath.WalkDir("./testdata/kddp", func(path string, d fs.DirEntry, err error) error {
		return runTests(t, "kddp", path, d, err)
	})

	if err != nil {
		t.Errorf("Error walking the test directory: %s", err)
	}
}

func TestStdlib(t *testing.T) {
	err := filepath.WalkDir("./testdata/stdlib", func(path string, d fs.DirEntry, err error) error {
		return runTests(t, "stdlib", path, d, err)
	})

	if err != nil {
		t.Errorf("Error walking the test directory: %s", err)
	}
}

func runTests(t *testing.T, ignoreFile string, path string, d fs.DirEntry, err error) error {
	if err != nil {
		t.Errorf("Error walking %s: %s\nskipping this directory", path, err)
		return fs.SkipDir
	}

	if !d.IsDir() {
		return nil
	}

	if d.Name() == ignoreFile {
		return nil
	}

	t.Run(d.Name(), func(t *testing.T) {
		t.Parallel()
		expected, err := os.ReadFile(filepath.Join(path, "expected.txt"))
		if err != nil {
			t.Errorf("Could not read expected output: %s", err.Error())
			return
		}

		filename := filepath.Join(path, filepath.Base(path)) + ".ddp"
		cmd := exec.Command("../build/kddp", "build", changeExtension(filename, ".ddp"), "-o", changeExtension(filename, ".exe"), "--verbose")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("compilation failed: %s\ncompiler output: %s", err, string(out))
			return
		} else {
			defer os.Remove(changeExtension(filename, ".exe"))
		}

		cmd = exec.Command(changeExtension(filename, ".exe"))
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Errorf("\nerror getting combined output: %s\noutput:\n%s", err, out)
			return
		}

		if out, expected := string(out), string(expected); out != expected {
			t.Errorf("Test did not yield the expected output\n"+
				"\x1b[1;32mExpected:\x1b[0m\n%s\n"+
				"\x1b[1;31mGot:\x1b[0m\n%s", expected, out)
			return
		}
	})

	return nil
}

// helper function
// returns path with the specified extension
func changeExtension(path, ext string) string {
	return path[:len(path)-len(filepath.Ext(path))] + ext
}
