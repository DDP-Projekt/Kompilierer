package compression

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func CompressFolder(from, to string) error {
	// create the .zip/.tar file
	f, err := os.Create(to)
	if err != nil {
		return err
	}
	defer f.Close()

	if runtime.GOOS == "windows" {
		writer := zip.NewWriter(f)
		defer writer.Close()

		// go through all the files of the source
		return filepath.Walk(from, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// create a local file header
			header, err := zip.FileInfoHeader(info)
			if err != nil {
				return err
			}

			// set compression
			header.Method = zip.Deflate

			// set relative path of a file as the header name
			header.Name = filepath.Join(filepath.Base(from), strings.TrimPrefix(path, from))
			if info.IsDir() {
				header.Name += "/"
			}

			// create writer for the file header and save content of the file
			headerWriter, err := writer.CreateHeader(header)
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()

			_, err = io.Copy(headerWriter, f)
			return err
		})
	} else if runtime.GOOS == "linux" {
		// tar > gzip > file
		zr := gzip.NewWriter(f)
		tw := tar.NewWriter(zr)

		// walk through every file in the folder
		filepath.Walk(from, func(file string, fi os.FileInfo, _ error) error {
			// generate tar header
			header, err := tar.FileInfoHeader(fi, file)
			if err != nil {
				return err
			}

			header.Name = filepath.Join(filepath.Base(from), strings.TrimPrefix(file, from))

			// write header
			if err := tw.WriteHeader(header); err != nil {
				return err
			}
			// if not a dir, write file content
			if !fi.IsDir() {
				data, err := os.Open(file)
				if err != nil {
					return err
				}
				if _, err := io.Copy(tw, data); err != nil {
					return err
				}
			}
			return nil
		})

		// produce tar
		if err := tw.Close(); err != nil {
			return err
		}
		// produce gzip
		if err := zr.Close(); err != nil {
			return err
		}
		//
		return nil
	} else {
		panic("invalid OS")
	}
}
