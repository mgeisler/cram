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
	cmds, err := ParseTest(buf)

	assert.NoError(t, err)
	assert.Len(t, cmds, 0)
}

func TestParseCommentaryOnly(t *testing.T) {
	buf := strings.NewReader("This file only has\nsome commentary.\n")
	cmds, err := ParseTest(buf)

	assert.NoError(t, err)
	assert.Len(t, cmds, 0)
}

func TestParseNoOutput(t *testing.T) {
	assert := assert.New(t)
	buf := strings.NewReader(`
  $ touch foo
  $ touch bar
`)
	cmds, err := ParseTest(buf)
	assert.NoError(err)
	if assert.Len(cmds, 2) {
		assert.Equal(Command{"touch foo", nil, 0}, cmds[0])
		assert.Equal(Command{"touch bar", nil, 0}, cmds[1])
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
	cmds, err := ParseTest(buf)
	assert.NoError(err)

	if assert.Len(cmds, 2) {
		assert.Equal(Command{
			`echo "hello\nworld"`,
			[]string{"hello", "world"},
			0,
		}, cmds[0])
		assert.Equal(Command{
			"echo goodbye",
			[]string{"goodbye"},
			0,
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
	cmds, err := ParseTest(buf)
	assert.NoError(err)

	if assert.Len(cmds, 5) {
		assert.Equal(Command{"false", []string{}, 1}, cmds[0])
		assert.Equal(Command{"echo hello; false", []string{"hello"}, 1}, cmds[1])
		assert.Equal(Command{"false", []string{}, 1}, cmds[2])
		assert.Equal(Command{"true", nil, 0}, cmds[3])
		assert.Equal(Command{"echo hello; false", []string{"hello"}, 1}, cmds[4])
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
	cmds := []Command{{"ls", nil, 0}, {"touch foo.txt", nil, 0}}
	lines := MakeScript(cmds, MakeBanner(u))
	banner := `echo "--- CRAM 12345678-1234-abcd-1234-123412345678 --- $?"`
	if assert.Len(t, lines, 4) {
		assert.Equal(t, "ls", lines[0])
		assert.Equal(t, banner, lines[1])
		assert.Equal(t, "touch foo.txt", lines[2])
		assert.Equal(t, banner, lines[3])
	}
}

func TestParseOutputEmpty(t *testing.T) {
	cmds := []Command{
		{"touch foo", nil, 0},
		{"touch bar", nil, 0},
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
		{"echo foo", []string{"foo"}, 0},
		{"echo bar", []string{"bar"}, 0},
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
		assert.Equal(t, []string{"foo"}, executed[0].ActualOutput)
		assert.Equal(t, 0, executed[0].ActualExitCode)
		assert.Equal(t, []string{"bar"}, executed[1].ActualOutput)
		assert.Equal(t, 1, executed[1].ActualExitCode)
	}
}
