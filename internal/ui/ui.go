package ui

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

const (
	Green  = "\033[32m"
	Red    = "\033[31m"
	Yellow = "\033[33m"
	Cyan   = "\033[36m"
	Dim    = "\033[2m"
	Bold   = "\033[1m"
	Reset  = "\033[0m"
)

func IsTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func TermWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return 80
	}
	return w
}

func Header(msg string) {
	if !IsTTY() {
		return
	}
	fmt.Printf("%s%s⚡ %s%s\n", Cyan, Bold, msg, Reset)
}

func Success(msg string) {
	fmt.Printf("%s✓ %s%s\n", Green, msg, Reset)
}

func Error(msg string) {
	fmt.Fprintf(os.Stderr, "%s✗ %s%s\n", Red, msg, Reset)
}

func Status(msg string) {
	fmt.Printf("%s→ %s%s\n", Dim, msg, Reset)
}

func Dimf(format string, a ...any) {
	fmt.Printf("%s"+format+"%s\n", append([]any{Dim}, append(a, Reset)...)...)
}

func Fatalf(format string, a ...any) {
	Error(fmt.Sprintf(format, a...))
	os.Exit(1)
}
