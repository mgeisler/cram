// Copyright 2016 Martin Geisler <martin@geisler.net>
//
// Cram is licensed under the MIT license, see the LICENSE file.

package cram

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestDropEol(t *testing.T) {
	var tests = []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"\n", ""},
		{"\r\n", ""},
		{"foo", "foo"},
		{"foo\n", "foo"},
		{"foo\r\n", "foo"},
		{"foo\nbar", "foo\nbar"},
		{"foo\r\nbar", "foo\r\nbar"},
	}

	for _, test := range tests {
		actual := DropEol(test.input)
		assert.Equal(t, test.expected, actual,
			fmt.Sprintf("DropEol(%#v)", test.input))
	}
}

func TestGlobToRegexp(t *testing.T) {
	var tests = []struct {
		input    string
		expected string
	}{
		{``, ``},
		{`foo`, `foo`},
		{`foo*`, `foo.*`},
		{`foo?`, `foo.`},
		{`a.b`, `a\.b`},
		{`a..b`, `a\.\.b`},
		{`a\*b`, `a\*b`},
		{`a\?b`, `a\?b`},
		{`a\\*b`, `a\\.*b`},
		{`x\`, `x`},
		{`x\x`, `xx`},
		{`x\n`, `xn`},
	}

	for _, test := range tests {
		actual := globToRegexp(test.input)
		assert.Equal(t, test.expected, actual,
			fmt.Sprintf("globToRegexp(%#v)", test.input))
	}
}

func TestEscape(t *testing.T) {
	var tests = []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"\n", "\n"},
		{"foo", "foo"},
		{"foo\n", "foo\n"},
		{"café", "café"},
		{`x " y \ z`, `x " y \ z`},
		{"\x01\x02\x03", `\x01\x02\x03 (esc)`},
		{"\x01\x02\x03\n", "\\x01\\x02\\x03 (esc)\n"},
	}

	for _, test := range tests {
		actual := Escape(test.input)
		assert.Equal(t, test.expected, actual,
			fmt.Sprintf("Escape(%#v)", test.input))
	}
}

func TestUnescape(t *testing.T) {
	var tests = []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"\n", "\n"},
		{"foo", "foo"},
		{"foo\n", "foo\n"},
		{"café", "café"},
		{`x " y \ z`, `x " y \ z`},
		{`\x01\x02\x03 (esc)`, "\x01\x02\x03"},
		{"\\x01\\x02\\x03 (esc)\n", "\x01\x02\x03\n"},
	}

	for _, test := range tests {
		actual, err := Unescape(test.input)
		assert.NoError(t, err)
		assert.Equal(t, test.expected, actual,
			fmt.Sprintf("Unescape(%#v)", test.input))
	}
}

func TestUnescapeError(t *testing.T) {
	unescaped, err := Unescape(`foo \ bar (esc)`)
	assert.Error(t, err)
	assert.Equal(t, "", unescaped)
}

func TestExecutedCommandFailed(t *testing.T) {
	cmd := Command{
		CmdLine:          "ls",
		ExpectedOutput:   []string{"bar\n", "foo\n"},
		ExpectedExitCode: 0,
		Lineno:           1,
	}
	re := Command{
		CmdLine:          "cat foo.txt",
		ExpectedOutput:   []string{"hello +world (re)\n"},
		ExpectedExitCode: 0,
		Lineno:           1,
	}
	badPattern := Command{
		CmdLine:          "cat foo.txt",
		ExpectedOutput:   []string{"* (re)\n"},
		ExpectedExitCode: 0,
		Lineno:           1,
	}

	glob := Command{
		CmdLine:          "ls",
		ExpectedOutput:   []string{"*.jpg (glob)\n"},
		ExpectedExitCode: 0,
		Lineno:           1,
	}

	var tests = []struct {
		cmd      ExecutedCommand
		expected bool
	}{
		// Simple output lines.
		{ExecutedCommand{&cmd, []string{"bar\n", "foo\n"}, 0}, false},
		{ExecutedCommand{&cmd, []string{"bar\n", "foo\n"}, 42}, true},
		{ExecutedCommand{&cmd, []string{"new", "output"}, 0}, true},
		{ExecutedCommand{&cmd, []string{"more", "lines"}, 0}, true},

		// Regular expressions.
		{ExecutedCommand{&re, []string{"hello +world (re)\n"}, 0}, false},
		{ExecutedCommand{&re, []string{"hello world\n"}, 0}, false},
		{ExecutedCommand{&re, []string{"hello world"}, 0}, false},
		{ExecutedCommand{&re, []string{"hello   world"}, 0}, false},
		{ExecutedCommand{&re, []string{"hello_world"}, 0}, true},
		{ExecutedCommand{&re, []string{"!hello world"}, 0}, true},
		{ExecutedCommand{&re, []string{"hello world!"}, 0}, true},
		{ExecutedCommand{&re, []string{"hello +world"}, 0}, true},
		{ExecutedCommand{&badPattern, []string{"..."}, 0}, true},

		// Glob patterns.
		{ExecutedCommand{&glob, []string{"*.jpg (glob)\n"}, 0}, false},
		{ExecutedCommand{&glob, []string{"foo.jpg\n"}, 0}, false},
		{ExecutedCommand{&glob, []string{"foo.jpg"}, 0}, false},
		{ExecutedCommand{&glob, []string{"foo.jpg  "}, 0}, true},
		{ExecutedCommand{&glob, []string{"quuz.png"}, 0}, true},
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, test.cmd.failed(),
			fmt.Sprintf("output: %q, exit code: %d",
				test.cmd.ActualOutput, test.cmd.ActualExitCode))
	}
}

func TestParseEmpty(t *testing.T) {
	buf := strings.NewReader("")
	test, err := ParseTest(buf, "<string>")

	assert.NoError(t, err)
	assert.Equal(t, test.Path, "<string>")
	assert.Len(t, test.Cmds, 0)
}

func TestParseOutputOnly(t *testing.T) {
	buf := strings.NewReader("\n\n  \n")
	test, err := ParseTest(buf, "<string>")

	assert.EqualError(t, err, `<string>:2: Output line "  \n" has no command`)
	assert.Equal(t, test.Path, "<string>")
	assert.Len(t, test.Cmds, 0)
}

func TestParseCommentaryOnly(t *testing.T) {
	buf := strings.NewReader("This file only has\nsome commentary.\n")
	test, err := ParseTest(buf, "<string>")

	assert.NoError(t, err)
	assert.Equal(t, test.Path, "<string>")
	assert.Len(t, test.Cmds, 0)
}

func TestParseNoOutput(t *testing.T) {
	assert := assert.New(t)
	buf := strings.NewReader(`
  $ touch foo
  $ touch bar
`)
	test, err := ParseTest(buf, "<string>")
	cmds := test.Cmds
	assert.NoError(err)
	if assert.Len(cmds, 2) {
		assert.Equal(Command{"touch foo\n", nil, 0, 2}, cmds[0])
		assert.Equal(Command{"touch bar\n", nil, 0, 3}, cmds[1])
	}
}

func TestParseCommands(t *testing.T) {
	assert := assert.New(t)
	buf := strings.NewReader(`
  $ echo "hello\nworld"
  hello
  world
  $ echo goodbye
  goodbye
`)
	test, err := ParseTest(buf, "<string>")
	assert.NoError(err)

	cmds := test.Cmds
	if assert.Len(cmds, 2) {
		assert.Equal(Command{
			"echo \"hello\\nworld\"\n",
			[]string{"hello\n", "world\n"},
			0, 2,
		}, cmds[0])
		assert.Equal(Command{
			"echo goodbye\n",
			[]string{"goodbye\n"},
			0, 5,
		}, cmds[1])
	}
}

func TestParseExitCodes(t *testing.T) {
	assert := assert.New(t)
	buf := strings.NewReader(`
Command with exit code but no output:

  $ false
  [1]

Commandline with exit code and output:

  $ echo hello; false
  hello
  [1]

Mixture of commands and output:

  $ false
  [1]
  $ true
  $ echo hello; false
  hello
  [1]
`)
	test, err := ParseTest(buf, "<string>")
	assert.NoError(err)

	cmds := test.Cmds
	if assert.Len(cmds, 5) {
		assert.Equal(Command{
			"false\n", []string{}, 1, 4},
			cmds[0])
		assert.Equal(Command{
			"echo hello; false\n", []string{"hello\n"}, 1, 9},
			cmds[1])
		assert.Equal(Command{
			"false\n", []string{}, 1, 15},
			cmds[2])
		assert.Equal(Command{
			"true\n", nil, 0, 17},
			cmds[3])
		assert.Equal(Command{
			"echo hello; false\n", []string{"hello\n"}, 1, 18},
			cmds[4])
	}
}

func TestMakeScriptEmpty(t *testing.T) {
	u, err := uuid.FromString("12345678-abcd-1234-abcd-123412345678")
	assert.NoError(t, err)
	cmds := []Command{}
	lines := MakeScript(cmds, MakeBanner(u))
	assert.Len(t, lines, 0)
}

func TestMakeScript(t *testing.T) {
	u, err := uuid.FromString("12345678-abcd-1234-abcd-123412345678")
	assert.NoError(t, err)
	cmds := []Command{
		{"ls", nil, 0, 0},
		{"touch foo.txt", nil, 0, 0},
	}
	lines := MakeScript(cmds, MakeBanner(u))
	banner := "echo \"--- CRAM $? 12345678-abcd-1234-abcd-123412345678 ---\"\n"
	if assert.Len(t, lines, 4) {
		assert.Equal(t, "ls", lines[0])
		assert.Equal(t, banner, lines[1])
		assert.Equal(t, "touch foo.txt", lines[2])
		assert.Equal(t, banner, lines[3])
	}
}

func TestParseEnviron(t *testing.T) {
	var tests = []struct {
		input    []string
		expected Env
	}{
		{[]string{}, Env{}},
		{[]string{"foo="}, Env{"foo": ""}},
		{[]string{"foo=x", "bar=y"}, Env{"foo": "x", "bar": "y"}},
		{[]string{"foo=x", "foo=y"}, Env{"foo": "y"}},
	}

	for _, test := range tests {
		actual := parseEnviron(test.input)
		assert.Equal(t, test.expected, actual,
			fmt.Sprintf("parseEnviron(%#v)", test.input))
	}
}

func TestParseEnvironError(t *testing.T) {
	assert.Panics(t, func() {
		parseEnviron([]string{"malformed entry"})
	})
}

func TestUnparseEnviron(t *testing.T) {
	var tests = []struct {
		input    Env
		expected []string
	}{
		{Env{}, []string{}},
		{Env{"foo": "x", "bar": "y"}, []string{"bar=y", "foo=x"}},
		{Env{"foo": ""}, []string{"foo="}},
	}

	for _, test := range tests {
		actual := unparseEnviron(test.input)
		sort.Strings(actual)
		assert.Equal(t, test.expected, actual,
			fmt.Sprintf("unparseEnviron(%#v)", test.input))
	}
}

func TestMakeEnvironment(t *testing.T) {
	pairs, err := MakeEnvironment("/foo/bar.t")
	assert.NoError(t, err)
	env := parseEnviron(pairs)
	assert.Equal(t, "/foo", env["TESTDIR"])
	assert.Equal(t, "C", env["LANG"])
	assert.Equal(t, "C", env["LC_ALL"])
	assert.Equal(t, "C", env["LANGUAGE"])
	assert.Equal(t, "GMT", env["TZ"])
	assert.Equal(t, "xterm", env["TERM"])
	assert.Equal(t, "80", env["COLUMNS"])
}

func TestParseOutputEmpty(t *testing.T) {
	cmds := []Command{
		{"touch foo", nil, 0, 0},
		{"touch bar", nil, 0, 0},
	}
	banner := "12345678-abcd-1234-abcd-123412345678 ---"
	output := []byte(`--- CRAM 0 12345678-abcd-1234-abcd-123412345678 ---
--- CRAM 1 12345678-abcd-1234-abcd-123412345678 ---
`)

	executed, err := ParseOutput(cmds, output, banner)
	assert.NoError(t, err)
	if assert.Len(t, executed, 2) {
		assert.Len(t, executed[0].ActualOutput, 0)
		assert.Equal(t, 0, executed[0].ActualExitCode)
		assert.Len(t, executed[1].ActualOutput, 0)
		assert.Equal(t, 1, executed[1].ActualExitCode)
	}
}

func TestParseOutput(t *testing.T) {
	cmds := []Command{
		{"echo foo", []string{"foo"}, 0, 0},
		{"echo bar", []string{"bar"}, 0, 0},
	}
	banner := "12345678-abcd-1234-abcd-123412345678 ---"
	output := []byte(`foo
--- CRAM 0 12345678-abcd-1234-abcd-123412345678 ---
bar
--- CRAM 1 12345678-abcd-1234-abcd-123412345678 ---
`)

	executed, err := ParseOutput(cmds, output, banner)
	assert.NoError(t, err)
	if assert.Len(t, executed, 2) {
		assert.Equal(t, []string{"foo\n"}, executed[0].ActualOutput)
		assert.Equal(t, 0, executed[0].ActualExitCode)
		assert.Equal(t, []string{"bar\n"}, executed[1].ActualOutput)
		assert.Equal(t, 1, executed[1].ActualExitCode)
	}
}

func TestParseOutputNoEol(t *testing.T) {
	cmds := []Command{
		{"echo -n foo", nil, 0, 0},
		{"echo -n bar", nil, 0, 0},
	}
	banner := "12345678-1234-abcd-1234-123412345678 ---"
	output := []byte(`foo--- CRAM 0 12345678-1234-abcd-1234-123412345678 ---
bar--- CRAM 1 12345678-1234-abcd-1234-123412345678 ---
`)

	executed, err := ParseOutput(cmds, output, banner)
	assert.NoError(t, err)
	if assert.Len(t, executed, 2) {
		assert.Equal(t, []string{"foo (no-eol)\n"}, executed[0].ActualOutput)
		assert.Equal(t, 0, executed[0].ActualExitCode)
		assert.Equal(t, []string{"bar (no-eol)\n"}, executed[1].ActualOutput)
		assert.Equal(t, 1, executed[1].ActualExitCode)
	}
}

func TestProcessInvalidPath(t *testing.T) {
	test, err := Process("/tmp", "no-such-file.t", 0)
	assert.Equal(t, test.Path, "no-such-file.t")
	assert.Error(t, err)
}
