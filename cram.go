package cram

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/satori/go.uuid"
)

const (
	commandPrefix = "  $ "
	outputPrefix  = "  "
)

type Command struct {
	CmdLine          string   // Command line passed to the shell.
	ExpectedOutput   []string // Expected output lines.
	ExpectedExitCode int      // Expected exit code.
}

type ExecutedCommand struct {
	*Command                // Command responsible for the output.
	ActualOutput   []string // Actual output read from stdout and stderr.
	ActualExitCode int      // Exit code.
}

type Result struct {
	Commands []ExecutedCommand // The executed commands.
	Script   string            // The script passed to the shell.
	Failures []ExecutedCommand // Failed commands.
}

// updateExitCode looks at the last line of output and updates exit
// code if it is of the form [n]. The exit code line is only required
// for non-zero exit codes.
func updateExitCode(cmd *Command) {
	lines := len(cmd.ExpectedOutput)
	if lines == 0 {
		return
	}
	line := cmd.ExpectedOutput[lines-1]

	l := len(line)
	if l == 0 || line[0] != '[' || line[l-1] != ']' {
		return
	}

	exitCode, err := strconv.Atoi(line[1 : l-1])
	if err != nil {
		// Not an exit code, just normal output.
		return
	}
	cmd.ExpectedOutput = cmd.ExpectedOutput[:lines-1]
	cmd.ExpectedExitCode = exitCode
}

// Parse splits an input test file into Commands.
func ParseTest(r io.Reader) (cmds []Command, err error) {
	const (
		inCommentary = iota
		inCommand
		inOutput
	)

	scanner := bufio.NewScanner(r)
	state := inCommentary
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, commandPrefix):
			if state == inOutput {
				updateExitCode(&cmds[len(cmds)-1])
			}
			line = line[len(commandPrefix):]
			cmds = append(cmds, Command{CmdLine: line})
			state = inCommand
		case strings.HasPrefix(line, outputPrefix):
			line = line[len(outputPrefix):]
			cmd := &cmds[len(cmds)-1]
			cmd.ExpectedOutput = append(cmd.ExpectedOutput, line)
			state = inOutput
		default:
			if state == inOutput {
				updateExitCode(&cmds[len(cmds)-1])
			}
			state = inCommentary
		}
	}
	if state == inOutput {
		updateExitCode(&cmds[len(cmds)-1])
	}
	err = scanner.Err()
	return
}

// MakeBanner turns a UUID into a nice banner we can recognize later
// in the output.
func MakeBanner(u uuid.UUID) string {
	return "--- CRAM " + u.String() + " ---"
}

// MakeScript produces a script ready to be sent to a shell. The
// banner should be a random string. It will be inserted in the output
// together with the exit status of each command.
func MakeScript(cmds []Command, banner string) (lines []string) {
	echo := fmt.Sprintf("echo \"%s $?\"", banner)
	for _, cmd := range cmds {
		lines = append(lines, cmd.CmdLine, echo)
	}
	return
}

func ParseOutput(cmds []Command, output []byte, banner string) (
	executed []ExecutedCommand, err error) {
	r := bytes.NewReader(output)
	scanner := bufio.NewScanner(r)

	i := 0
	actualOutput := []string{}
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, banner) {
			number := line[len(banner)+1:]
			exitCode, e := strconv.Atoi(number)
			if e != nil {
				err = e
				return
			}
			executed = append(executed, ExecutedCommand{
				Command:        &cmds[i],
				ActualExitCode: exitCode,
				ActualOutput:   actualOutput,
			})
			actualOutput = nil
			i++
		} else {
			actualOutput = append(actualOutput, line)
		}
	}
	return executed, scanner.Err()
}

// Execute a script in the specified working directory.
func ExecuteScript(workdir string, lines []string) ([]byte, error) {
	script := strings.Join(lines, "\n")
	cmd := exec.Command("/bin/sh", "-")
	cmd.Dir = workdir
	cmd.Stdin = strings.NewReader(script)
	return cmd.CombinedOutput()
}

func filterFailures(executed []ExecutedCommand) (failures []ExecutedCommand) {
	for _, cmd := range executed {
		// Quick check first
		err := cmd.ActualExitCode != cmd.ExpectedExitCode
		// More expensive check next
		if !err {
			actual := strings.Join(cmd.ActualOutput, "\n")
			expected := strings.Join(cmd.ExpectedOutput, "\n")
			err = actual != expected
		}
		if err {
			failures = append(failures, cmd)
		}
	}
	return
}

// Process parses a .t file, executes the test commands and compares
// the actual output to the expected output.
func Process(tempdir, path string) (result Result, err error) {
	fp, err := os.Open(path)
	if err != nil {
		return
	}
	defer fp.Close()
	commands, err := ParseTest(fp)
	if err != nil {
		return
	}

	workdir := filepath.Join(tempdir, filepath.Base(path))
	err = os.Mkdir(workdir, 0700)
	if err != nil {
		return
	}

	u := uuid.NewV4()
	banner := MakeBanner(u)
	lines := MakeScript(commands, banner)

	output, err := ExecuteScript(workdir, lines)
	if err != nil {
		return
	}

	executed, err := ParseOutput(commands, output, banner)
	if err != nil {
		return
	}

	failures := filterFailures(executed)
	result = Result{executed, strings.Join(lines, "\n"),
		failures}
	return
}
