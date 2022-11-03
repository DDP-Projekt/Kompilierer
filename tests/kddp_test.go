package tests

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"
)

func TestKDDP(t *testing.T) {
	err := filepath.WalkDir("./testdata/kddp", func(path string, d fs.DirEntry, err error) error {
		return runTests(t, "kddp", path, d, err, false)
	})

	if err != nil {
		t.Errorf("Error walking the test directory: %s", err)
	}
}

func TestStdlib(t *testing.T) {
	err := filepath.WalkDir("./testdata/stdlib", func(path string, d fs.DirEntry, err error) error {
		return runTests(t, "stdlib", path, d, err, false)
	})

	if err != nil {
		t.Errorf("Error walking the test directory: %s", err)
	}
}

func TestMemory(t *testing.T) {
	t.Run("KDDP", func(t *testing.T) {
		if err := filepath.WalkDir("./testdata/kddp", func(path string, d fs.DirEntry, err error) error {
			return runTests(t, "kddp", path, d, err, true)
		}); err != nil {
			t.Errorf("Error walking the test directory: %s", err)
		}
	})

	t.Run("Stdlib", func(t *testing.T) {
		if err := filepath.WalkDir("./testdata/stdlib", func(path string, d fs.DirEntry, err error) error {
			return runTests(t, "stdlib", path, d, err, true)
		}); err != nil {
			t.Errorf("Error walking the test directory: %s", err)
		}
	})
}

func runTests(t *testing.T, ignoreFile string, path string, d fs.DirEntry, err error, testMemory bool) error {
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

		// read expected.txt
		expected, err := os.ReadFile(filepath.Join(path, "expected.txt"))
		if err != nil {
			t.Errorf("Could not read expected output: %s", err.Error())
			return
		}

		// get ddp file path
		filename := filepath.Join(path, filepath.Base(path)) + ".ddp"

		// build dpp file
		cmd := exec.Command("../build/DDP/bin/kddp", "kompiliere", changeExtension(filename, ".ddp"), "-o", changeExtension(filename, ".exe"), "--wortreich")
		// get build output
		if out, err := cmd.CombinedOutput(); err != nil {
			// error if failed
			t.Errorf("compilation failed: %s\ncompiler output: %s", err, string(out))
			return
		} else {
			// remove exe if successful
			defer os.Remove(changeExtension(filename, ".exe"))
		}

		// run ddp executeable
		cmd = exec.Command(changeExtension(filename, ".exe"))

		// read input
		input, err := os.Open(filepath.Join(path, "input.txt"))
		// if input.txt exists
		if err == nil {
			// replace stdin with input.txt
			cmd.Stdin = input
		}

		// get output
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Errorf("\nerror getting combined output: %s\noutput:\n%s", err, out)
			return
		}

		input.Close() // close input file

		if testMemory {
			now_at_zero := regexp.MustCompile("freed 24 bytes, now at 0 bytesAllocated")
			if now_at_zero.Find(out) == nil {
				now_at_x := regexp.MustCompile("(now at (?P<num_bytes>[0-9]+) bytesAllocated)$")
				match := now_at_x.FindSubmatch(out)
				if match != nil {
					num_bytes := match[now_at_x.SubexpIndex("num_bytes")]
					t.Errorf("Program exited with %s bytes still allocated!", num_bytes)
				} else {
					t.Errorf("Could not find number of bytes allocated, something bad happened!")
				}
				if err := os.WriteFile(filepath.Join(path, "output.dump.txt"), out, os.ModePerm); err != nil {
					t.Errorf("Error dumping output: %s", err)
				}
			}
		} else {
			// error if 'out' was not the expected output
			if out, expected := string(out), string(expected); out != expected {
				t.Errorf("Test did not yield the expected output\n"+
					"\x1b[1;32mExpected:\x1b[0m\n%s\n"+
					"\x1b[1;31mGot:\x1b[0m\n%s", expected, out)
				return
			}
		}
	})

	return nil
}

// helper function
// returns path with the specified extension
func changeExtension(path, ext string) string {
	return path[:len(path)-len(filepath.Ext(path))] + ext
}
