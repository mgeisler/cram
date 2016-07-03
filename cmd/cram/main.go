// Copyright 2016 Martin Geisler <martin@geisler.net>
//
// Cram is licensed under the MIT license, see the LICENSE file.

package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/alecthomas/kingpin"
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

// Wrapper for a path and an index.
type pathIndex struct {
	Path string
	Idx  int
}

// Options describe the command line options. They are parsed in main
// and passed to run.
type Options struct {
	Jobs        int
	KeepTmp     bool
	Interactive bool
	Verbose     bool
	Debug       bool
}

// processPath runs cram.Process on the paths in the paths channel.
// The results (and any errors) are fed to the results channel.
func processPath(jobs *sync.WaitGroup, tempdir string,
	paths chan pathIndex, results chan processResult) {
	for pi := range paths {
		result, err := cram.Process(tempdir, pi.Path, pi.Idx)
		results <- processResult{result, err}
	}
	jobs.Done()
}

// expandArgs turns command line arguments into pathIndex elements.
// Directories are walked recursively and .t files found inside them
// are added to the paths channel. Files on the command line are added
// to paths directly.
func expandArgs(args []string, paths chan pathIndex) {
	// Index passed to cram.Process. Incremented when a pathIndex
	// is added to paths by the walker or the loop below.
	idx := 0

	walker := func(path string, info os.FileInfo, err error) error {
		// Add the path if there is an error (we want the error
		// from cram.Process) of if it is a .t file.
		if err != nil || !info.IsDir() && filepath.Ext(path) == ".t" {
			paths <- pathIndex{path, idx}
			idx++
		}
		return nil
	}

	for _, path := range args {
		// We want different behavior for files and directories
		// mentioned on the command line: files should be
		// processed regardless of their extension, directories
		// should be searched recursively for .t files. To
		// distinguish, we need to stat the path here instead of
		// simply calling filepath.Walk on it.
		info, err := os.Lstat(path)

		if err == nil && !info.IsDir() {
			// Add the path regardless of file extension.
			paths <- pathIndex{path, idx}
			idx++
		} else {
			// Let the walk function deal with the path.
			filepath.Walk(path, walker)
		}
	}
	close(paths)
}

func run(args []string, opts Options) (error, int) {
	tempdir, err := ioutil.TempDir("", "cram-")
	if err != nil {
		msg := "Could not create temp directory: " + err.Error()
		return errors.New(msg), 2
	}
	if opts.KeepTmp {
		fmt.Println("# Temporary directory:", tempdir)
	} else {
		defer os.RemoveAll(tempdir)
	}

	errCount, cmdCount, resultCount := 0, 0, 0
	failures := []cram.ExecutedTest{}

	// Number of goroutines to process the test files. We default to 2
	// times the number of cores in the main function below.
	if opts.Jobs < 1 {
		opts.Jobs = 1
	}

	// Input and result channels with space for a few items before we
	// block.
	paths := make(chan pathIndex, 8)
	results := make(chan processResult, 8)

	// Fan-in control that will let us close the results channel once
	// all jobs are done.
	var jobs sync.WaitGroup
	jobs.Add(opts.Jobs)

	// Expand the command line arguments into pathIndex elements.
	go expandArgs(args, paths)

	// Start the worker goroutines that will process the test files
	// found by expandArgs.
	for i := 0; i < opts.Jobs; i++ {
		go processPath(&jobs, tempdir, paths, results)
	}

	// Close the results channel when done.
	go func() {
		jobs.Wait()
		close(results)
	}()

	for result := range results {
		resultCount++
		test := result.Test
		err := result.Err

		if opts.Debug {
			fmt.Fprintf(os.Stderr, "# %s\n", test.Path)
			fmt.Fprintln(os.Stderr, test.Script)
		}

		cmdCount += len(test.Cmds)

		switch {
		case err != nil:
			if opts.Verbose {
				switch err := err.(type) {
				case *cram.InvalidTestError:
					fmt.Printf("E %s\n", err)
				default:
					fmt.Printf("E %s: %s\n", test.Path, err)
				}
			} else {
				fmt.Fprintln(os.Stderr, err)
				fmt.Print("E")
			}
			errCount++
		case len(test.Failures) > 0:
			if opts.Verbose {
				fmt.Printf("F %s: %d of %d commands failed\n",
					test.Path, len(test.Failures), len(test.Cmds))
			} else {
				fmt.Print("F")
			}
			failures = append(failures, test)
		default:
			if opts.Verbose {
				fmt.Printf(". %s: %d commands passed\n",
					test.Path, len(test.Cmds))
			} else {
				fmt.Print(".")
			}
		}
	}
	fmt.Print("\n")

	processFailures(failures, opts.Interactive)

	msg := fmt.Sprintf("# Ran %d tests (%d commands), %d errors, %d failures",
		resultCount, cmdCount, errCount, len(failures))

	exitCode := 0
	if errCount > 0 {
		exitCode = 2
	} else if len(failures) > 0 {
		exitCode = 1
	}
	return errors.New(msg), exitCode
}

func main() {
	interactive := kingpin.
		Flag("interactive", "interactively update test file on failure").
		Short('i').
		Bool()
	verbose := kingpin.
		Flag("verbose", "show names of test files").
		Short('v').
		Bool()
	debug := kingpin.
		Flag("debug", "output debug information").
		Bool()
	keepTmp := kingpin.
		Flag("keep-tmp", "keep temporary directory after executing tests").
		Bool()
	jobs := kingpin.
		Flag("jobs", "number of tests to run in parallel").
		Short('j').
		Default(strconv.Itoa(2 * runtime.NumCPU())).
		Int()
	paths := kingpin.
		Arg("path", "test files or directories").
		Default(".").
		Strings()

	kingpin.Version("cram version 0.0.0")
	kingpin.Parse()

	opts := Options{*jobs, *keepTmp, *interactive, *verbose, *debug}
	err, exitCode := run(*paths, opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitCode)
	}
}
