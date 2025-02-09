package ddperror

import (
	"fmt"

	"github.com/DDP-Projekt/Kompilierer/src/token"
)

type WarnCode int

const (
	TODO_STMT_FOUND WarnCode = iota
)

func NewWarning(code WarnCode, Range token.Range, file string, a ...any) Message {
	return Message{
		Code:  int(code),
		Range: Range,
		File:  file,
		Msg:   code.WarnMessage(a),
	}
}

func (c WarnCode) WarnMessage(a ...any) string {
	return fmt.Sprintf([]string{
		"Für diesen Teil des Programms fehlt eine Implementierung und es wird ein Laufzeitfehler ausgelöst.",
	}[c], a...)
}
