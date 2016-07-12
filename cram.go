// Copyright 2016 Martin Geisler <martin@geisler.net>
//
// Cram is licensed under the MIT license, see the LICENSE file.

package cram

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/satori/go.uuid"
)

const (
	commandPrefix      = "  $ "
	continuationPrefix = "  > "
	outputPrefix       = "  "

	reSuffix    = " (re)"
	globSuffix  = " (glob)"
	noEolSuffix = " (no-eol)"
	escSuffix   = " (esc)"
)

type Env map[string]string

type InvalidTestError struct {
	Path   string // Path to test file.
	Lineno int    // Line number of failure
	Msg    string // Error message
}

func (e *InvalidTestError) Error() string {
	return fmt.Sprintf("%s:%d: %s", e.Path, e.Lineno, e.Msg)
}

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

// matchEntireLine returns true exactly when pattern can be compiled
// and matches all of line.
func matchEntireLine(pattern, line string) bool {
	pattern = "^(?:" + pattern + ")$"
	matched, err := regexp.MatchString(pattern, line)
	return err == nil && matched
}

// globToRegexp translates a glob pattern into the corresponding
// regular expression. We cannnot simply use filepath.Match since we
// want "*" to match a sequence of any character instead of stopping
// at "/" (or "\\" on Windows). Also, filepath.Match has a quirk where
// there is no escaping on Windows.
func globToRegexp(pattern string) string {
	regexpMeta := `\.+*?()|[]{}^$`
	buf := make([]byte, 2*len(pattern))
	j := 0
Loop:
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '?':
			buf[j] = '.'
		case '*':
			buf[j], buf[j+1] = '.', '*'
			j++
		case '\\':
			// Skip over the backslash
			i++
			if i == len(pattern) {
				break Loop
			}
			// If we didn't break, add the next character to buf as a
			// literal.
			fallthrough
		default:
			// Escape character is necessary
			if strings.IndexByte(regexpMeta, pattern[i]) >= 0 {
				buf[j] = '\\'
				j++
			}
			buf[j] = pattern[i]
		}
		j++
	}
	return string(buf[:j])
}

// failed indicates if the actual exit code or output differed from
// what was expected.
func (cmd *ExecutedCommand) failed() bool {
	if cmd.ActualExitCode != cmd.ExpectedExitCode {
		return true
	}
	if len(cmd.ActualOutput) != len(cmd.ExpectedOutput) {
		return true
	}
	for i, actual := range cmd.ActualOutput {
		expected := cmd.ExpectedOutput[i]
		// Always accept an exact match, even if the line might end
		// with (re). This means that such lines need no escaping in
		// the test file and are quick to match.
		if actual == expected {
			continue
		}

		// The following tests ignore EOLs.
		actual = DropEol(actual)
		expected = DropEol(expected)

		switch {
		case strings.HasSuffix(expected, reSuffix):
			pattern := expected[:len(expected)-len(reSuffix)]
			if !matchEntireLine(pattern, actual) {
				return true
			}
		case strings.HasSuffix(expected, globSuffix):
			pattern := expected[:len(expected)-len(globSuffix)]
			if !matchEntireLine(globToRegexp(pattern), actual) {
				return true
			}
		case strings.HasSuffix(expected, escSuffix):
			// The same output can be escaped in multiple differnet
			// ways by the user: both "x (esc)" and "\x78 (esc)" are
			// ways of saying "x". We normalize the output by
			// unescaping and then escaping it. This ensures that the
			// escaped form is the same as what was applied to the
			// actual output in ParseOutput.
			expected, err := Unescape(expected)
			if err != nil || Escape(expected) != actual {
				return true
			}
		default:
			// No special suffix, not equal by the check above => we
			// found a change in the output.
			return true
		}
	}
	return false
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
		case strings.HasPrefix(line, continuationPrefix):
			if state != inCommand {
				err = &InvalidTestError{path, lineno,
					fmt.Sprintf("Continuation line %q has no command", line)}
				return
			}
			line = line[len(continuationPrefix):]
			cmd := &test.Cmds[len(test.Cmds)-1]
			cmd.CmdLine = cmd.CmdLine + line
			cmd.Lineno++
		case strings.HasPrefix(line, outputPrefix):
			if state == inCommentary {
				err = &InvalidTestError{path, lineno,
					fmt.Sprintf("Output line %q has no command", line)}
				return
			}
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
	return u.String() + " ---"
}

// MakeScript produces a script ready to be sent to a shell. The
// banner should be a random string. It will be inserted in the output
// together with the exit status of each command.
func MakeScript(cmds []Command, banner string) (lines []string) {
	echo := fmt.Sprintf("echo \"--- CRAM $? %s\"\n", banner)
	for _, cmd := range cmds {
		lines = append(lines, cmd.CmdLine, echo)
	}
	return
}

// parseEnviron turns a slice of "key=value" pairs into a map from
// "key" to "value". The inverse is unparseEnviron.
func parseEnviron(pairs []string) Env {
	env := make(Env, len(pairs))
	for _, pair := range pairs {
		i := strings.IndexByte(pair, '=')
		if i < 0 {
			panic(fmt.Sprintf("parseEnviron: found no '=' in %q", pair))
		}
		env[pair[:i]] = pair[i+1:]
	}
	return env
}

// unparseEnviron turns a map of keys and values into a slice of
// "key=value" strings. It is the inverse of parseEnviron.
func unparseEnviron(env Env) []string {
	pairs := make([]string, len(env))
	i := 0
	for key, value := range env {
		pairs[i] = key + "=" + value
		i++
	}
	return pairs
}

// MakeEnvironment prepares the environment to be used when executing
// the test in the given path. It currently sets TESTDIR to the
// dirname of path.
func MakeEnvironment(path string) ([]string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	env := parseEnviron(os.Environ())
	// Test file directory
	env["TESTDIR"] = filepath.Dir(abs)
	// Reset locale variables
	env["LC_ALL"] = "C"
	env["LANG"] = "C"
	env["LANGUAGE"] = "C"
	return unparseEnviron(env), nil
}

// Escape quotes non-printable characters in s with a backslash. If
// anything was changed, " (esc)" is added to the result. A final
// newline in s (if any) is kept unescaped.
func Escape(s string) string {
	trimmed := DropEol(s)
	quoted := strconv.Quote(trimmed)
	inner := quoted[1 : len(quoted)-1]

	// strconv.Quote changed `"` to `\"` and `\` to `\\` since it
	// returns a double quoted string. We want to undo this since both
	// " and \ are printable and need no escaping in our test files.
	//
	// Replacing `\"` with `"` is reversable in Unescape since there
	// can be no lone occurance of `"` in quoted.
	cleaned := strings.Replace(inner, `\"`, `"`, -1)

	// We cannot do the same for backslash since we won't be able to
	// reverse this in Unescape (there will most likely be other
	// occurances of `\` in quoted). However, we simply need to
	// determine if replacing `\\` with `\` would give us the string
	// we started with. If it does, we return s unchanged. If not, the
	// we leave the `\\` as they are since we will be returning a
	// string marked with "(esc)" in that case.
	if strings.Replace(cleaned, `\\`, `\`, -1) == trimmed {
		return s
	}

	// Add the " (esc)" suffix followed by the EOL (if any) from the
	// input string.
	return cleaned + escSuffix + s[len(trimmed):]
}

// Unescape is the inverse of Escape. It decodes escaped characters
// such as \t, \x01, etc in s if s ends with escSuffix.
func Unescape(s string) (string, error) {
	trimmed := DropEol(s)
	if !strings.HasSuffix(trimmed, escSuffix) {
		return s, nil
	}
	withoutMarker := trimmed[:len(trimmed)-len(escSuffix)]
	escaped := strings.Replace(withoutMarker, `"`, `\"`, -1)
	quoted := `"` + escaped + `"`
	unquoted, err := strconv.Unquote(quoted)
	if err != nil {
		return "", err
	}
	return unquoted + s[len(trimmed):], nil
}

// ParseOutput finds the actual output and exit codes for a slice of
// commands. The result is a slice of executed commands. The actual
// output is normalized, meaning that a missing final EOL in the
// output is represented as noEolSuffix.
func ParseOutput(cmds []Command, output []byte, banner string) (
	executed []ExecutedCommand, err error) {
	r := bytes.NewReader(output)
	reader := bufio.NewReader(r)

	banner = banner + "\n"
	i := 0
	actualOutput := []string{}
	line := ""
	for err == nil {
		line, err = reader.ReadString('\n')
		if strings.HasSuffix(line, banner) {
			// Cut off space, banner, and final newline. The line then
			// looks like "...--- CRAM NN", where "..." can be empty.
			line = line[:len(line)-len(banner)-1]
			// Find space between "--- CRAM" part and exit code.
			lastSpace := stringsLastIndexByte(line, ' ')

			prefix := line[:lastSpace-len("--- CRAM")]
			if len(prefix) > 0 {
				line := prefix + noEolSuffix + "\n"
				actualOutput = append(actualOutput, Escape(line))
			}

			number := line[lastSpace+1:]
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
			actualOutput = append(actualOutput, Escape(line))
		}
	}
	if err == io.EOF {
		err = nil
	}
	return
}

// Execute a script in the specified working directory.
func ExecuteScript(workdir string, env []string, lines []string) ([]byte, error) {
	script := strings.Join(lines, "")
	cmd := exec.Command("/bin/sh", "-")
	cmd.Dir = workdir
	cmd.Env = env
	cmd.Stdin = strings.NewReader(script)
	return cmd.CombinedOutput()
}

func filterFailures(executed []ExecutedCommand) (failures []ExecutedCommand) {
	for _, cmd := range executed {
		if cmd.failed() {
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
		for _, outputLine := range cmd.ActualOutput {
			output = append(output, "  "+outputLine)
		}
		lastLine := output[len(output)-1]
		if lastLine[len(lastLine)-1] != '\n' {
			output[len(output)-1] = lastLine + noEolSuffix + "\n"
		}
		lastLineno = cmd.Lineno + len(cmd.ExpectedOutput)

		if cmd.ActualExitCode != cmd.ExpectedExitCode {
			// Add a line with the actual exit code, but only if it is
			// non-zero since [0] is implied.
			if cmd.ActualExitCode != 0 {
				line := fmt.Sprintf("  [%d]\n", cmd.ActualExitCode)
				output = append(output, line)
			}

			// Figure out if we should skip a line in the input file.
			// We should only skip a line if we are overwriting the
			// expected exit code -- we should not skip a line if we
			// merely added the actual exit code.
			if lastLineno < len(lines) {
				lastLine := lines[lastLineno]
				line := fmt.Sprintf("  [%d]", cmd.ExpectedExitCode)
				// Extra whitespace on the exit code line is okay.
				if strings.HasPrefix(lastLine, line) {
					lastLineno++
				}
			}
		}
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
// the actual output to the expected output. The idx passed is used to
// make the working directory unique inside tempdir and must be
// different for each test file.
func Process(tempdir, path string, idx int) (result ExecutedTest, err error) {
	// Make sure Path is set, even if we fail later.
	result.Path = path
	fp, err := os.Open(path)
	if err != nil {
		return
	}
	defer fp.Close()
	test, err := ParseTest(fp, path)
	if err != nil {
		return
	}

	// Create unique base inside the tempdir
	base := fmt.Sprintf("%03d-%s", idx, filepath.Base(path))
	// Remove file extension (often, but not necessarily ".t")
	base = base[:len(base)-len(filepath.Ext(base))]
	workdir := filepath.Join(tempdir, base)
	err = os.Mkdir(workdir, 0700)
	if err != nil {
		return
	}

	u := uuid.NewV4()
	banner := MakeBanner(u)
	lines := MakeScript(test.Cmds, banner)
	env, err := MakeEnvironment(path)
	if err != nil {
		return
	}

	output, err := ExecuteScript(workdir, env, lines)
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
