package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var (
	scanner    = bufio.NewScanner(os.Stdin)
	always_yes = false
)

func prompt(question string) bool {
	fmt.Print(ColorString(question+"? [j/n]: ", Cyan))
	if always_yes {
		fmt.Println("j")
		return true
	}
	scanner.Scan()
	answer := strings.ToLower(scanner.Text())
	return strings.ToLower(answer) == "j"
}
