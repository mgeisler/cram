package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/mgeisler/cram"
)

func showFailures(failures []cram.ExecutedCommand) {
	for _, cmd := range failures {
		actual := strings.Join(cmd.ActualOutput, "\n  ")
		expected := strings.Join(cmd.ExpectedOutput, "\n  ")

		fmt.Printf("When executing %+#v, got\n", cmd.CmdLine)
		fmt.Println(" ", actual)
		fmt.Println("but expected")
		fmt.Println(" ", expected)
	}
}

func run(ctx *cli.Context) {
	tempdir, err := ioutil.TempDir("", "cram-")
	if err != nil {
		return
	}
	if ctx.GlobalBool("keep-tmp") {
		fmt.Println("# Temporary directory:", tempdir)
	} else {
		defer os.RemoveAll(tempdir)
	}

	errors, cmdCount := 0, 0
	failures := []cram.ExecutedCommand{}

	for _, path := range ctx.Args() {
		result, err := cram.Process(tempdir, path)
		if ctx.GlobalBool("debug") {
			fmt.Fprintf(os.Stderr, "# %s\n", path)
			fmt.Fprintln(os.Stderr, result.Script)
		}

		cmdCount += len(result.Commands)

		switch {
		case err != nil:
			fmt.Fprintln(os.Stderr, err)
			fmt.Print("E")
			errors++
		case len(result.Failures) > 0:
			fmt.Print("F")
			failures = append(failures, result.Failures...)
		default:
			fmt.Print(".")
		}
	}
	fmt.Print("\n")

	showFailures(failures)

	fmt.Printf("# Ran %d tests (%d commands), %d errors, %d failures.\n",
		len(ctx.Args()), cmdCount, errors, len(failures))
}

func main() {
	app := cli.NewApp()
	app.Action = run
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug",
			Usage: "output debug information",
		},
		cli.BoolFlag{
			Name:  "keep-tmp",
			Usage: "keep temporary directory after executing tests",
		},
	}
	app.Run(os.Args)
}
