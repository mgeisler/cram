package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/kylelemons/godebug/diff"
	"github.com/mgeisler/cram"
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

	// Number of goroutines to process the test files. We default to 2
	// times the number of cores.
	parallelism := ctx.GlobalInt("jobs")
	if parallelism <= 0 {
		parallelism = 2 * runtime.NumCPU()
	}
	if parallelism > len(ctx.Args()) {
		parallelism = len(ctx.Args())
	}

	// Input and result channels with space for a few items before we
	// block.
	paths := make(chan string, 8)
	results := make(chan processResult, 8)

	for i := 0; i < parallelism; i++ {
		go func() {
			for path := range paths {
				result, err := cram.Process(tempdir, path)
				results <- processResult{result, err}
			}
		}()
	}

	go func() {
		for _, path := range ctx.Args() {
			paths <- path
		}
		close(paths)
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

	fmt.Printf("# Ran %d tests (%d commands), %d errors, %d failures\n",
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
