package cram

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
		assert.Equal(Command{"touch foo", nil}, cmds[0])
		assert.Equal(Command{"touch bar", nil}, cmds[1])
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
		}, cmds[0])
		assert.Equal(Command{
			"echo goodbye",
			[]string{"goodbye"},
		}, cmds[1])
	}
}
