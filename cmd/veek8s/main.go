package main

import (
	"fmt"
	"os"

	"github.com/vee-sh/veessh/internal/veek8s"
)

func main() {
	msg, code := veek8s.Status(veek8s.Run(os.Args[1:]))
	if code == 0 {
		return
	}
	if msg != "" {
		fmt.Fprintln(os.Stderr, msg)
	}
	os.Exit(code)
}
