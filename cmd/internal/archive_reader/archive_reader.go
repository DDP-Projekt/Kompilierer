package archive_reader

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type tarFile struct {
	hdr *tar.Header
	buf *bytes.Buffer
}

// ArchiveReader provides a way to only read some selected files
// from either a .zip or .tar.gz archive
type ArchiveReader struct {
	zipArchive *zip.ReadCloser
	tarArchive *tar.Reader
	tarFiles   map[string]tarFile
	close_fn   func() error
}

// create a new archiveReader from the given archive (either a .zip or .tar.gz file)
func New(archive string) (*ArchiveReader, error) {
	switch filepath.Ext(archive) {
	case ".zip":
		zr, err := zip.OpenReader(archive)
		if err != nil {
			return nil, fmt.Errorf("failed to open zip archive: %w", err)
		}
		return &ArchiveReader{
			zipArchive: zr,
			close_fn: func() error {
				return zr.Close()
			},
		}, nil
	case ".gz":
		f, err := os.Open(archive)
		if err != nil {
			return nil, fmt.Errorf("failed to open tar archive: %w", err)
		}
		gzr, err := gzip.NewReader(f)
		if err != nil {
			f.Close()
			return nil, fmt.Errorf("failed to open gz compressed archive: %w", err)
		}
		tr := tar.NewReader(gzr)
		return &ArchiveReader{
			tarArchive: tr,
			tarFiles:   make(map[string]tarFile),
			close_fn: func() error {
				gzr.Close()
				return f.Close()
			},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported archive type: %s", filepath.Ext(archive))
	}
}

// close the archiveReader and all it's resources
func (ar *ArchiveReader) Close() error {
	return ar.close_fn()
}

// returns a file-like reader, wether the file is a directory the size of that file for the first path that satisfies fn
func (ar *ArchiveReader) GetElementFunc(fn func(string) bool) (io.ReadCloser, bool, uint64, error) {
	// zip case is simple as it already provides a file-like interface
	if ar.zipArchive != nil {
		for _, f := range ar.zipArchive.File {
			if fn(f.Name) {
				rc, err := f.Open()
				return rc, f.FileInfo().IsDir(), f.UncompressedSize64, err
			}
		}
	}

	// tar case is more complicated as we have to read the whole archive
	// check if we already have the file in memory
	for name, f := range ar.tarFiles {
		if fn(name) {
			return io.NopCloser(f.buf), f.hdr.FileInfo().IsDir(), uint64(f.buf.Len()), nil
		}
	}

	// otherwise iterate over the tar archive
	for hdr, err := ar.tarArchive.Next(); err != io.EOF; hdr, err = ar.tarArchive.Next() {
		if err != nil {
			return nil, false, 0, fmt.Errorf("failed to get next element from tar archive: %w", err)
		}
		if fn(hdr.Name) {
			return io.NopCloser(ar.tarArchive), hdr.FileInfo().IsDir(), uint64(hdr.Size), nil
		} else {
			ar.tarFiles[hdr.Name] = tarFile{hdr: hdr, buf: &bytes.Buffer{}}
			_, err := io.Copy(ar.tarFiles[hdr.Name].buf, ar.tarArchive)
			if err != nil {
				return nil, false, 0, fmt.Errorf("failed to read element (%s) from tar archive: %w", hdr.Name, err)
			}
		}
	}
	return nil, false, 0, fmt.Errorf("element not found")
}

// allows iteration over all elements in the archive
// fn is called on each element and should and error occur that error is returned
func (ar *ArchiveReader) IterateElementsFunc(fn func(string, bool, io.Reader, uint64) error) error {
	// zip case is simple as it already provides a file-like interface
	if ar.zipArchive != nil {
		for _, f := range ar.zipArchive.File {
			rc, err := f.Open()
			if err != nil {
				return fmt.Errorf("failed to open file (%s) in zip archive: %w", f.Name, err)
			}
			defer rc.Close()
			if err := fn(f.Name, f.FileInfo().IsDir(), rc, f.UncompressedSize64); err != nil {
				return err
			}
		}
		return nil
	}

	// read the whole tar archive into memory
	for hdr, err := ar.tarArchive.Next(); err != io.EOF; hdr, err = ar.tarArchive.Next() {
		if err != nil {
			return fmt.Errorf("failed to get next element from tar archive: %w", err)
		}
		ar.tarFiles[hdr.Name] = tarFile{hdr: hdr, buf: &bytes.Buffer{}}
		_, err := io.Copy(ar.tarFiles[hdr.Name].buf, ar.tarArchive)
		if err != nil {
			return fmt.Errorf("failed to read element (%s) from tar archive: %w", hdr.Name, err)
		}
	}

	// iterate over the archive
	for name, f := range ar.tarFiles {
		if err := fn(name, f.hdr.FileInfo().IsDir(), f.buf, uint64(f.buf.Len())); err != nil {
			return err
		}
	}
	return nil
}
