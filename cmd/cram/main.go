package main

import (
	"fmt"
	"os"

	"github.com/codegangsta/cli"
	"github.com/mgeisler/cram"
)

func run(ctx *cli.Context) {
	errors, failures, cmdCount := 0, 0, 0
	for _, path := range ctx.Args() {
		commands, err := cram.Process(path)
		cmdCount += len(commands)
		// No tests are run yet, so we can only distinguish between
		// successes and errors, not test failures.
		if err == nil {
			fmt.Print(".")
		} else {
			fmt.Print("E")
			errors++
		}
	}
	fmt.Print("\n")

	fmt.Printf("# Ran %d tests (%d commands), %d errors, %d failures.\n",
		len(ctx.Args()), cmdCount, errors, failures)
}

func main() {
	app := cli.NewApp()
	app.Action = run
	app.Run(os.Args)
}
