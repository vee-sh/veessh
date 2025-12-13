package main

import (
	"context"
	"errors"
	"fmt"
	"os"

    "github.com/vee-sh/veessh/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		if errors.Is(err, context.Canceled) {
			fmt.Fprintln(os.Stderr, "ok. exiting")
			os.Exit(130)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
