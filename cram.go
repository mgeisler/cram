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

type Test struct {
	Path string    // Path to test file.
	Cmds []Command // Commands.
}

type Command struct {
	CmdLine          string   // Command line passed to the shell.
	ExpectedOutput   []string // Expected output lines.
	ExpectedExitCode int      // Expected exit code.
	Lineno           int      // Line number of first output line.
}

type ExecutedCommand struct {
	*Command                // Command responsible for the output.
	ActualOutput   []string // Actual output read from stdout and stderr.
	ActualExitCode int      // Exit code.
}

// ExecutedTest captures the executed commands, the script sent to the
// shell and the failed commands. The Test struct is embedded as a
// value instead of pointer to make the zero value for ExecutedTest
// immediatedly useful.
type ExecutedTest struct {
	Test
	ExecutedCmds []ExecutedCommand // All executed commands.
	Script       string            // The script passed to the shell.
	Failures     []ExecutedCommand // Failed commands.
}

// DropEol removes a final end-of-line from s. It removes both Unix ("\n")
// and DOS ("\r\n") end-of-line characters.
func DropEol(s string) string {
	l := len(s)
	if l == 0 || s[l-1] != '\n' {
		return s
	}
	drop := 1
	if l > 1 && s[l-2] == '\r' {
		drop = 2
	}
	return s[:l-drop]
}

// updateExitCode looks at the last line of output and updates exit
// code if it is of the form [n]. The exit code line is only required
// for non-zero exit codes.
func updateExitCode(cmd *Command) {
	lines := len(cmd.ExpectedOutput)
	if lines == 0 {
		return
	}
	line := DropEol(cmd.ExpectedOutput[lines-1])

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
func ParseTest(r io.Reader, path string) (test Test, err error) {
	const (
		inCommentary = iota
		inCommand
		inOutput
	)

	test.Path = path
	reader := bufio.NewReader(r)
	state := inCommentary
	lineno := 0
	line := ""
	for err == nil {
		line, err = reader.ReadString('\n')
		switch {
		case strings.HasPrefix(line, commandPrefix):
			if state == inOutput {
				updateExitCode(&test.Cmds[len(test.Cmds)-1])
			}
			line = line[len(commandPrefix):]
			cmd := Command{
				CmdLine: line,
				Lineno:  lineno + 1,
			}
			test.Cmds = append(test.Cmds, cmd)
			state = inCommand
		case strings.HasPrefix(line, outputPrefix):
			line = line[len(outputPrefix):]
			cmd := &test.Cmds[len(test.Cmds)-1]
			cmd.ExpectedOutput = append(cmd.ExpectedOutput, line)
			state = inOutput
		default:
			if state == inOutput {
				updateExitCode(&test.Cmds[len(test.Cmds)-1])
			}
			state = inCommentary
		}
		lineno++
	}
	if state == inOutput {
		updateExitCode(&test.Cmds[len(test.Cmds)-1])
	}
	if err == io.EOF {
		err = nil
	}
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
	echo := fmt.Sprintf("echo \"%s $?\"\n", banner)
	for _, cmd := range cmds {
		lines = append(lines, cmd.CmdLine, echo)
	}
	return
}

func ParseOutput(cmds []Command, output []byte, banner string) (
	executed []ExecutedCommand, err error) {
	r := bytes.NewReader(output)
	reader := bufio.NewReader(r)

	i := 0
	actualOutput := []string{}
	line := ""
	for err == nil {
		line, err = reader.ReadString('\n')
		if strings.HasPrefix(line, banner) {
			number := DropEol(line[len(banner)+1:])
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
	if err == io.EOF {
		err = nil
	}
	return
}

// Execute a script in the specified working directory.
func ExecuteScript(workdir string, lines []string) ([]byte, error) {
	script := strings.Join(lines, "")
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
			actual := strings.Join(cmd.ActualOutput, "")
			expected := strings.Join(cmd.ExpectedOutput, "")
			err = actual != expected
		}
		if err {
			failures = append(failures, cmd)
		}
	}
	return
}

// Patch takes an ExecutedTest, a slice ExecutedCommands and returns
// the patched output where ActualOutput from each ExecutedCommand
// replaces the ExpectedOutput.
func Patch(r io.Reader, w io.Writer, cmds []ExecutedCommand) (err error) {
	reader := bufio.NewReader(r)
	writer := bufio.NewWriter(w)

	lines := []string{}
	line := ""
	for err == nil {
		line, err = reader.ReadString('\n')
		lines = append(lines, line)
	}
	if err == io.EOF {
		err = nil
	}

	output := []string{}
	lastLineno := 0

	for _, cmd := range cmds {
		pre := lines[lastLineno:cmd.Lineno]
		output = append(output, pre...)
		output = append(output, "  "+strings.Join(cmd.ActualOutput, "  "))
		lastLineno = cmd.Lineno + len(cmd.ExpectedOutput)
	}
	post := lines[lastLineno:]
	output = append(output, post...)

	for _, line := range output {
		_, err = writer.WriteString(line)
	}
	err = writer.Flush()
	return
}

// Process parses a .t file, executes the test commands and compares
// the actual output to the expected output.
func Process(tempdir, path string) (result ExecutedTest, err error) {
	fp, err := os.Open(path)
	if err != nil {
		return
	}
	defer fp.Close()
	test, err := ParseTest(fp, path)
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
	lines := MakeScript(test.Cmds, banner)

	output, err := ExecuteScript(workdir, lines)
	if err != nil {
		return
	}

	executed, err := ParseOutput(test.Cmds, output, banner)
	if err != nil {
		return
	}

	failures := filterFailures(executed)
	result = ExecutedTest{test, executed, strings.Join(lines, ""),
		failures}
	return
}
