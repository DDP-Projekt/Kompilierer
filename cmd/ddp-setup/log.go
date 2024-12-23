package main

import (
	"fmt"

	supportscolor "github.com/jwalton/go-supportscolor"
)

const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Cyan   = "\033[36m"
)

func InfoF(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

func DoneF(format string, args ...any) {
	fmt.Printf(ColorString("[\u221A] "+format+"\n", Green), args...)
}

func WarnF(format string, args ...any) {
	fmt.Printf(ColorString("[!] "+format+"\n", Yellow), args...)
}

var errored = false

func ErrorF(format string, args ...any) {
	errored = true
	fmt.Printf(ColorString("[X] "+format+"\n", Red), args...)
}

func ColorString(str, colour string) string {
	if supportscolor.Stdout().SupportsColor {
		return colour + str + Reset
	}
	return str
}
