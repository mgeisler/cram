package cram

import (
	"bufio"
	"io"
	"os"
	"strings"
)

const (
	commandPrefix = "  $ "
	outputPrefix  = "  "
)

type Command struct {
	CmdLine        string   // Command line as it will be passed to the shell.
	ExpectedOutput []string // Expected output including any newlines.
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

// Process parses a .t file, executes the test commands and compares
// the actual output to the expected output.
func Process(path string) (commands []Command, err error) {
	fp, err := os.Open(path)
	if err != nil {
		return
	}
	defer fp.Close()
	commands, err = ParseTest(fp)
	return
}
