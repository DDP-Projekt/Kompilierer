package format

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/parser"
)

func FormatBytes(src []byte, to io.Writer) error {
	return FormatReader(bytes.NewReader(src), to)
}

func FormatFile(path string, to io.Writer) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return FormatReader(file, to)
}

func FormatReader(from io.Reader, to io.Writer) error {
	src, err := io.ReadAll(from)
	if err != nil {
		return err
	}

	mod, err := parser.Parse(parser.Options{
		Source: src,
	})
	if err != nil {
		return err
	}
	return Format(mod, to)
}

func Format(mod *ast.Module, to io.Writer) (err error) {
	defer panic_wrapper(&err)

	var formatter formatter
	ast.VisitModule(mod, &formatter)
	if formatter.err != nil {
		return err
	}

	_, err = io.Copy(to, &formatter.result)
	return err
}

func panic_wrapper(err *error) {
	if recovered := recover(); recovered != nil {
		recovered_err, ok := recovered.(error)
		if !ok {
			recovered_err = fmt.Errorf("%v", recovered)
		}

		*err = errors.Join(*err, recovered_err)
	}
}
