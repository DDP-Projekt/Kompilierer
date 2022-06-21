package tests

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestKDDP(t *testing.T) {
	err := filepath.WalkDir("./testdata", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			t.Logf("Error walking %s: %s", path, err)
			return fs.SkipDir
		}
		if !d.IsDir() {
			return nil
		}
		if d.Name() == "testdata" {
			return nil
		}

		t.Run(d.Name(), func(t *testing.T) {
			expected, err := os.ReadFile(filepath.Join(path, "expected.txt"))
			if err != nil {
				t.Errorf("Could not read expected output: %s", err.Error())
				return
			}
			filename := filepath.Join(path, filepath.Base(path))
			cmd := exec.Command("../build/kddp", "build", filename+".ddp", "-o", filename)
			if runtime.GOOS == "windows" {
				filename += ".exe"
			}
			if _, err := cmd.CombinedOutput(); err != nil {
				t.Errorf("compilation failed: %s", err)
				return
			} else {
				defer os.Remove(filename)
			}

			cmd = exec.Command(filename)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Errorf("error getting combined output: %s", err)
				return
			}
			if out, expected := string(out), string(expected); out != expected {
				t.Errorf("Test did not yield the expected output\nExpected:\n%s\nGot:\n%s", expected, out)
				return
			}
		})

		return nil
	})

	if err != nil {
		t.Logf("Error walking the test directory: %s", err)
	}
}
