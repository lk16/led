package main

import (
	"fmt"
	"os"

	"github.com/lk16/led/internal/editor"
)

func main() {
	path, err := editor.ParseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if err := editor.Run(path); err != nil {
		fmt.Fprintln(os.Stderr, "led:", err)
		os.Exit(1)
	}
}
