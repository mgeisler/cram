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
		result, err := cram.Process(path)
		if ctx.GlobalBool("debug") {
			fmt.Fprintf(os.Stderr, "# %s\n", path)
			fmt.Fprintln(os.Stderr, result.Script)
		}

		cmdCount += len(result.Commands)
		// No tests are run yet, so we can only distinguish between
		// successes and errors, not test failures.
		if err == nil {
			fmt.Print(".")
		} else {
			fmt.Fprintln(os.Stderr, err)
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
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug",
			Usage: "output debug information",
		},
	}
	app.Run(os.Args)
}
