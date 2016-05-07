package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/mgeisler/cram"
)

func booleanPrompt(prompt string) (bool, error) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(prompt, " ")
		answer, err := reader.ReadString('\n')
		if err != nil {
			return false, err
		}
		answer = strings.ToLower(strings.TrimSpace(answer))
		switch answer {
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		default:
			fmt.Println("Please answer 'yes' or 'no'")
		}
	}
}

func showFailures(tests []cram.ExecutedTest) {
	for _, test := range tests {
		for _, cmd := range test.Failures {
			fmt.Printf("When executing %+#v, got\n", cram.DropEol(cmd.CmdLine))
			if cmd.ActualExitCode != cmd.ExpectedExitCode {
				fmt.Printf("  exit code %d, but expected %d\n",
					cmd.ActualExitCode, cmd.ExpectedExitCode)
			} else {
				actual := cram.DropEol(strings.Join(cmd.ActualOutput, "  "))
				expected := cram.DropEol(strings.Join(cmd.ExpectedOutput, "  "))

				fmt.Println(" ", actual)
				fmt.Println("but expected")
				fmt.Println(" ", expected)
			}
		}
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
	failures := []cram.ExecutedTest{}

	for _, path := range ctx.Args() {
		result, err := cram.Process(tempdir, path)
		if ctx.GlobalBool("debug") {
			fmt.Fprintf(os.Stderr, "# %s\n", path)
			fmt.Fprintln(os.Stderr, result.Script)
		}

		cmdCount += len(result.Cmds)

		switch {
		case err != nil:
			fmt.Fprintln(os.Stderr, err)
			fmt.Print("E")
			errors++
		case len(result.Failures) > 0:
			fmt.Print("F")
			failures = append(failures, result)
		default:
			fmt.Print(".")
		}
	}
	fmt.Print("\n")

	showFailures(failures)

	fmt.Printf("# Ran %d tests (%d commands), %d errors, %d failures.\n",
		len(ctx.Args()), cmdCount, errors, len(failures))

	switch {
	case errors > 0:
		os.Exit(2)
	case len(failures) > 0:
		os.Exit(1)
	default:
		os.Exit(0)
	}
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
