package ddperror_test

import (
	"testing"

	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/scanner"
)

func TestErrors(t *testing.T) {
	testScannerError(t,
		`2.
		die Zahl v ist 54.`, ddperror.EXPECTED_CAPITAL.ErrorMessage())

	testScannerError(t, `"\d"`, ddperror.UNKNOWN_ESCAPE_SEQ.ErrorMessage("d"))
	testScannerError(t, `"\	"`, ddperror.UNKNOWN_ESCAPE_SEQ.ErrorMessage("<U+0009>"))
	testScannerError(t, `"\'"`, ddperror.UNKNOWN_ESCAPE_SEQ.ErrorMessage("'"))
	testScannerError(t, `'\d'`, ddperror.UNKNOWN_ESCAPE_SEQ.ErrorMessage("d"))
	testScannerError(t, `'\	'`, ddperror.UNKNOWN_ESCAPE_SEQ.ErrorMessage("<U+0009>"))
	testScannerError(t, `'\"`, ddperror.UNKNOWN_ESCAPE_SEQ.ErrorMessage("\""))
	testScannerError(t, `"hi`, "")
}

func testScannerError(t *testing.T, src string, msg string) {
	didError := false
	result, _ := scanner.Scan(scanner.Options{
		FileName:     "test.ddp",
		ScannerMode:  scanner.ModeStrictCapitalization,
		ErrorHandler: testHandler(t, msg, &didError),
		Source:       []byte(src),
	})

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
