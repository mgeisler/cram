package cram

import (
	"fmt"
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

	assert.EqualError(t, err, `<string>:2: Output line "  \n" has not command.`)
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
	u, err := uuid.FromString("123456781234abcd1234123412345678")
	assert.NoError(t, err)
	cmds := []Command{}
	lines := MakeScript(cmds, MakeBanner(u))
	assert.Len(t, lines, 0)
}

func TestMakeScript(t *testing.T) {
	u, err := uuid.FromString("123456781234abcd1234123412345678")
	assert.NoError(t, err)
	cmds := []Command{
		{"ls", nil, 0, 0},
		{"touch foo.txt", nil, 0, 0},
	}
	lines := MakeScript(cmds, MakeBanner(u))
	banner := "echo \"--- CRAM 12345678-1234-abcd-1234-123412345678 --- $?\"\n"
	if assert.Len(t, lines, 4) {
		assert.Equal(t, "ls", lines[0])
		assert.Equal(t, banner, lines[1])
		assert.Equal(t, "touch foo.txt", lines[2])
		assert.Equal(t, banner, lines[3])
	}
}

func TestParseOutputEmpty(t *testing.T) {
	cmds := []Command{
		{"touch foo", nil, 0, 0},
		{"touch bar", nil, 0, 0},
	}
	banner := "--- CRAM 12345678-1234-abcd-1234-123412345678 ---"
	output := []byte(`--- CRAM 12345678-1234-abcd-1234-123412345678 --- 0
--- CRAM 12345678-1234-abcd-1234-123412345678 --- 1
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
	banner := "--- CRAM 12345678-1234-abcd-1234-123412345678 ---"
	output := []byte(`foo
--- CRAM 12345678-1234-abcd-1234-123412345678 --- 0
bar
--- CRAM 12345678-1234-abcd-1234-123412345678 --- 1
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
