package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strings"

	"github.com/kylelemons/godebug/diff"
	"github.com/mgeisler/cram"
	"github.com/urfave/cli"
)

// We use a single, shared reader of os.Stdin to avoid losing data due
// to buffering. If we create a new reader every time we need to read
// a line of text, the reader will read "too much" (it buffers) and we
// will lose the buffered data when we create a new reader. This is
// visible when trying to answer two prompts using
//
//   $ echo "x\ny" | cram --interactive
var stdinReader = bufio.NewReader(os.Stdin)

func booleanPrompt(prompt string) (bool, error) {
	for {
		fmt.Print(prompt, " ")
		answer, err := stdinReader.ReadString('\n')
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

func processFailures(tests []cram.ExecutedTest, interactive bool) (
	err error) {

	for _, test := range tests {
		var needPatching []cram.ExecutedCommand

		for _, cmd := range test.Failures {
			fmt.Printf("When executing %+#v:\n", cram.DropEol(cmd.CmdLine))

			expected := cmd.ExpectedOutput
			actual := cmd.ActualOutput

			if cmd.ActualExitCode != 0 {
				line := fmt.Sprintf("[%d]\n", cmd.ActualExitCode)
				actual = append(actual, line)
			}
			if cmd.ExpectedExitCode != 0 {
				line := fmt.Sprintf("[%d]\n", cmd.ExpectedExitCode)
				expected = append(expected, line)
			}

			chunks := diff.DiffChunks(expected, actual)
			for _, chunk := range chunks {
				for _, line := range chunk.Added {
					fmt.Printf("+%s", line)
				}
				for _, line := range chunk.Deleted {
					fmt.Printf("-%s", line)
				}
				for _, line := range chunk.Equal {
					fmt.Printf(" %s", line)
				}
			}

			if interactive {
				accept, e := booleanPrompt("Accept this change?")
				if e != nil {
					err = e
					return
				}
				if accept {
					needPatching = append(needPatching, cmd)
				}
			}
		}

		if needPatching != nil {
			input, e := os.Open(test.Path)
			if err = e; err != nil {
				return
			}

			outPath := test.Path + ".patched"
			output, e := os.Create(outPath)
			if err = e; err != nil {
				return
			}
			err = cram.Patch(input, output, needPatching)
			if err != nil {
				return
			}
			err = os.Rename(outPath, test.Path)
			if err != nil {
				return
			}
			fmt.Println("Patched", test.Path)
		}

	}
	return
}

// Wrapper for the return type of cram.Process.
type processResult struct {
	Test cram.ExecutedTest
	Err  error
}

func run(ctx *cli.Context) error {
	tempdir, err := ioutil.TempDir("", "cram-")
	if err != nil {
		msg := "Could not create temp directory: " + err.Error()
		return cli.NewExitError(msg, 2)
	}
	if ctx.GlobalBool("keep-tmp") {
		fmt.Println("# Temporary directory:", tempdir)
	} else {
		defer os.RemoveAll(tempdir)
	}

	errors, cmdCount := 0, 0
	failures := []cram.ExecutedTest{}

	// Number of goroutines to process the test files. We default to 2
	// times the number of cores in the main function below.
	parallelism := ctx.GlobalInt("jobs")
	if parallelism < 1 {
		parallelism = 1
	}
	if parallelism > len(ctx.Args()) {
		parallelism = len(ctx.Args())
	}

	// Input and result channels with space for a few items before we
	// block.
	paths := ctx.Args()
	indexes := make(chan int, 8)
	results := make(chan processResult, 8)

	for i := 0; i < parallelism; i++ {
		go func() {
			for i := range indexes {
				result, err := cram.Process(tempdir, paths[i], i)
				results <- processResult{result, err}
			}
		}()
	}

	go func() {
		for i := range paths {
			indexes <- i
		}
		close(indexes)
	}()

	for i := 0; i < len(ctx.Args()); i++ {
		result := <-results
		test := result.Test
		err := result.Err

		if ctx.GlobalBool("debug") {
			fmt.Fprintf(os.Stderr, "# %s\n", test.Path)
			fmt.Fprintln(os.Stderr, test.Script)
		}

		cmdCount += len(test.Cmds)

		switch {
		case err != nil:
			fmt.Fprintln(os.Stderr, err)
			fmt.Print("E")
			errors++
		case len(test.Failures) > 0:
			fmt.Print("F")
			failures = append(failures, test)
		default:
			fmt.Print(".")
		}
	}
	fmt.Print("\n")

	processFailures(failures, ctx.GlobalBool("interactive"))

	msg := fmt.Sprintf("# Ran %d tests (%d commands), %d errors, %d failures",
		len(ctx.Args()), cmdCount, errors, len(failures))

	exitCode := 0
	if errors > 0 {
		exitCode = 2
	} else if len(failures) > 0 {
		exitCode = 1
	}
	return cli.NewExitError(msg, exitCode)
}

func main() {
	app := cli.NewApp()
	app.Usage = "command line test framework"
	app.Action = run
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "interactive",
			Usage: "interactively update test file on failure",
		},
		cli.BoolFlag{
			Name:  "debug",
			Usage: "output debug information",
		},
		cli.BoolFlag{
			Name:  "keep-tmp",
			Usage: "keep temporary directory after executing tests",
		},
		cli.IntFlag{
			Name:  "jobs, j",
			Usage: "number of tests to run in parallel",
			Value: 2 * runtime.NumCPU(),
		},
	}
	app.Run(os.Args)
}
