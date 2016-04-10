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
	CmdLine        string   // Command line as it will be passed to the shell.
	ExpectedOutput []string // Expected output including any newlines.
}

type ExecutedCommand struct {
	*Command              // Command responsible for the output.
	ActualOutput [][]byte // Actual output read from stdout and stderr.
	ExitCode     int      // Exit code.
}

type Result struct {
	Commands []ExecutedCommand // The executed commands.
	Script   string            // The script passed to the shell.
}

// Parse splits an input test file into Commands.
func ParseTest(r io.Reader) (cmds []Command, err error) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, commandPrefix):
			line = line[len(commandPrefix):]
			cmds = append(cmds, Command{CmdLine: line})
		case strings.HasPrefix(line, outputPrefix):
			line = line[len(outputPrefix):]
			cmd := &cmds[len(cmds)-1]
			cmd.ExpectedOutput = append(cmd.ExpectedOutput, line)
		}
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

	byteBanner := []byte(banner)

	i := 0
	actualOutput := [][]byte{}
	for scanner.Scan() {
		line := scanner.Bytes()
		if bytes.HasPrefix(line, byteBanner) {
			number := string(line[len(byteBanner)+1:])
			exitCode, e := strconv.Atoi(number)
			if e != nil {
				err = e
				return
			}
			executed = append(executed, ExecutedCommand{
				Command:      &cmds[i],
				ExitCode:     exitCode,
				ActualOutput: actualOutput,
			})
			actualOutput = nil
			i++
		} else {
			// Copy line since subsequent calls to Scanner.Scan may
			// overwrite the underlying array of line
			c := make([]byte, len(line))
			copy(c, line)
			actualOutput = append(actualOutput, c)
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
	result = Result{executed, strings.Join(lines, "\n")}
	return
}
