package compression

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func DecompressFolder(from, to string) error {
	if err := os.MkdirAll(to, 0o755); err != nil {
		return err
	}

	if runtime.GOOS == "windows" {
		r, err := zip.OpenReader(from)
		if err != nil {
			return err
		}
		defer func() {
			if err := r.Close(); err != nil {
				panic(err)
			}
		}()

		// Closure to address file descriptors issue with all the deferred .Close() methods
		extractAndWriteFile := func(f *zip.File) error {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer func() {
				if err := rc.Close(); err != nil {
					panic(err)
				}
			}()

			path := filepath.Clean(f.Name)

			// Check for ZipSlip (Directory traversal)
			if !strings.HasPrefix(path, filepath.Clean(to)) {
				return fmt.Errorf("illegal file path: %s", path)
			}

			if f.FileInfo().IsDir() {
				os.MkdirAll(path, f.Mode())
			} else {
				os.MkdirAll(filepath.Dir(path), f.Mode())
				f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
				if err != nil {
					return err
				}
				defer func() {
					if err := f.Close(); err != nil {
						panic(err)
					}
				}()

				_, err = io.Copy(f, rc)
				if err != nil {
					return err
				}
			}
			return nil
		}

		for _, f := range r.File {
			err := extractAndWriteFile(f)
			if err != nil {
				return err
			}
		}

		return nil
	} else if runtime.GOOS == "linux" {
		f, err := os.Open(from)
		if err != nil {
			return err
		}
		defer f.Close()

		zr, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		tr := tar.NewReader(zr)

		for header, err := tr.Next(); err != io.EOF; header, err = tr.Next() {
			if err != nil {
				return err
			}

			if strings.Contains(header.Name, "..") {
				return fmt.Errorf("archive contains '..' in path Name: %s", header.Name)
			}

			path := filepath.Join(to, header.Name)

			switch header.Typeflag {
			case tar.TypeDir:
				os.MkdirAll(path, 0o755)
			case tar.TypeReg:
				outFile, err := os.Create(path)
				if err != nil {
					return err
				}
				if _, err := io.Copy(outFile, tr); err != nil {
					outFile.Close()
					return err
				}
				outFile.Close()
			default:
				return fmt.Errorf("unknown type in %s while extracting .tar.gz: %d", header.Name, header.Typeflag)
			}
		}

		return zr.Close()
	} else {
		panic("invalid OS")
	}
}
