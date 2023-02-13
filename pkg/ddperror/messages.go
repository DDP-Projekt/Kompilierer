package ddperror

import "fmt"

// generates a error message in the format
// 		Es wurde ..., ... oder ... erwartet aber ... gefunden
func MsgGotExpected(got any, expected ...any) string {
	switch len(expected) {
	case 0:
		panic("called MsgGotExpected without expected arg")
	case 1:
		return fmt.Sprintf("Es wurde %v erwartet aber %v gefunden", expected[0], got)
	default:
		msg := fmt.Sprintf("Es wurde %v", expected[0])
		for _, v := range expected[1 : len(expected)-1] {
			msg += fmt.Sprintf(", %v", v)
		}
		msg += fmt.Sprintf("oder %v erwartet aber %v gefunden", expected[len(expected)-1], got)
		return msg
	}
}

const (
	MSG_MISSING_RETURN         = "Am Ende einer Funktion, die etwas zurück gibt, muss eine Rückgabe Anweisung stehen"
	MSG_CHAR_LITERAL_TOO_LARGE = "Ein Buchstaben Literal darf nur einen Buchstaben enthalten"
	MSG_INVALID_UTF8           = "Der Quelltext entspricht nicht dem UTF-8 Standard"
	MSG_INVALID_FILE_EXTENSION = "Ungültiger Datei Typ (nicht .ddp)"
	MSG_GLOBAL_RETURN          = "Man kann nur aus Funktionen einen Wert zurückgeben"
)
