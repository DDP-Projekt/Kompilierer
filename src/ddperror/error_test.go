package ddperror_test

import (
	"testing"

	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/scanner"
	"github.com/DDP-Projekt/Kompilierer/src/token"
)

func TestErrors(t *testing.T) {
	testScannerError(t,
		`2.
		die Zahl v ist 54.`, ddperror.EXPECTED_CAPITAL.ErrorMessage(), false)

	testScannerError(t, `"\d"`, ddperror.UNKNOWN_ESCAPE_SEQ.ErrorMessage("d"), false)
	testScannerError(t, `"\	"`, ddperror.UNKNOWN_ESCAPE_SEQ.ErrorMessage("<U+0009>"), false)
	testScannerError(t, `"\'"`, ddperror.UNKNOWN_ESCAPE_SEQ.ErrorMessage("'"), false)
	testScannerError(t, `'\d'`, ddperror.UNKNOWN_ESCAPE_SEQ.ErrorMessage("d"), false)
	testScannerError(t, `'\	'`, ddperror.UNKNOWN_ESCAPE_SEQ.ErrorMessage("<U+0009>"), false)
	testScannerError(t, `'\"`, ddperror.UNKNOWN_ESCAPE_SEQ.ErrorMessage("\""), false)
	testScannerError(t, "a\xc5z", ddperror.INVALID_UTF8.ErrorMessage(), true)
	testScannerError2(t, scanner.Options{FileName: "test.txt", ScannerMode: scanner.ModeStrictCapitalization}, ddperror.INVALID_FILE_TYPE.ErrorMessage())
	testScannerError(t, `'hi'`, ddperror.CHAR_TOO_LONG.ErrorMessage(), false)
	testScannerError(t, `'\ti'`, ddperror.CHAR_TOO_LONG.ErrorMessage(), false)
	testScannerAliasError(t, `<2>`, ddperror.INVALID_PARAMETER_NAME.ErrorMessage())
	testScannerAliasError(t, `<a€>`, ddperror.INVALID_PARAMETER_NAME.ErrorMessage())
	testScannerAliasError(t, `<abc`, ddperror.OPEN_PARAMETER.ErrorMessage())
	testScannerAliasError(t, `<>`, ddperror.EMPTY_PARAMETER.ErrorMessage())
	testScannerAliasError(t, `<alle>`, ddperror.ALIAS_EXPECTED_NAME.ErrorMessage())
}

func testScannerError2(t *testing.T, opts scanner.Options, msg string) {
	result, err := scanner.Scan(opts)
	if err == nil {
		t.Errorf("Unexpected scan success: %v", result)
	} else if err.Error() != msg {
		t.Errorf("Scan Error didn't match `%s` != `%s`", err.Error(), msg)
	}
}

func testScannerError(t *testing.T, src string, msg string, scanError bool) {
	didError := false
	result, err := scanner.Scan(scanner.Options{
		FileName:     "test.ddp",
		ScannerMode:  scanner.ModeStrictCapitalization,
		ErrorHandler: testHandler(t, msg, &didError),
		Source:       []byte(src),
	})

	if scanError {
		if err == nil {
			t.Errorf("Unexpected scan success: %v", result)
		} else if err.Error() != msg {
			t.Errorf("Scan Error didn't match `%s` != `%s`", err.Error(), msg)
		}
		return
	}

	if !didError { // not using Scan's err return because some errors don't return an err
		t.Errorf("Unexpected success: %v", result)
		return
	}
}

func testScannerAliasError(t *testing.T, src string, msg string) {
	didError := false
	result, _ := scanner.ScanAlias(token.Token{Literal: src}, testHandler(t, ddperror.ALIAS_ERROR.ErrorMessage(src, msg), &didError))

	if !didError { // not using Scan's err return because some errors don't return an err
		t.Errorf("Unexpected success: %v", result)
		return
	}
}

func testHandler(t *testing.T, msg string, didError *bool) ddperror.Handler {
	return func(err ddperror.Message) {
		*didError = true
		if err.Error() != msg {
			t.Errorf("Error didn't match: `%s` != `%s`", err.Error(), msg)
			return
		}
	}
}
