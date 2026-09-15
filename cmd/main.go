package main

import (
	"fmt"
	"os"

	"github.com/lk16/led/internal/editor"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: led <file>")
		os.Exit(2)
	}
	if err := editor.Run(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, "led:", err)
		os.Exit(1)
	}
}
