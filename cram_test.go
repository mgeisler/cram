package cram

import (
	"strings"
	"testing"
)

func TestParseEmpty(t *testing.T) {
	buf := strings.NewReader("")
	cmds, err := ParseTest(buf)

	if err != nil {
		t.Error("Expected no error, got", err)
	}

	if len(cmds) > 0 {
		t.Error("Expected no commands, got", cmds)
	}
}

func TestParseCommentaryOnly(t *testing.T) {
	buf := strings.NewReader("This file only has\nsome commentary.\n")
	cmds, err := ParseTest(buf)

	if err != nil {
		t.Error("Expected no error, got", err)
	}

	if len(cmds) > 0 {
		t.Error("Expected no commands, got", cmds)
	}
}

func TestParseNoOutput(t *testing.T) {
	buf := strings.NewReader(`
  $ touch foo
  $ touch bar
`)
	cmds, err := ParseTest(buf)

	if err != nil {
		t.Error("Expected no error, got", err)
	}

	if len(cmds) == 2 {
		if cmds[0].CmdLine != "touch foo" {
			t.Error("Expected touch foo, got", cmds[0])
		}
		if len(cmds[0].ExpectedOutput) > 0 {
			t.Error("Expected no output, got", cmds[0].ExpectedOutput)
		}
		if cmds[1].CmdLine != "touch bar" {
			t.Error("Expected touch bar, got", cmds[1])
		}
		if len(cmds[1].ExpectedOutput) > 0 {
			t.Error("Expected no output, got", cmds[1].ExpectedOutput)
		}
	} else {
		t.Error("Expected two commands, got", cmds)
	}
}

func TestParseCommands(t *testing.T) {
	buf := strings.NewReader(`
  $ echo "hello\nworld"
  hello
  world
  $ echo goodbye
  goodbye
`)
	cmds, err := ParseTest(buf)

	if err != nil {
		t.Error("Expected no error, got", err)
	}

	if len(cmds) == 2 {
		if cmds[0].CmdLine != `echo "hello\nworld"` {
			t.Error("Expected echo hello\\nworld, got", cmds[0])
		}
		if len(cmds[0].ExpectedOutput) == 2 {
			if cmds[0].ExpectedOutput[0] != "hello" {
				t.Error("Expected hello, got", cmds[0].ExpectedOutput[0])
			}
			if cmds[0].ExpectedOutput[1] != "world" {
				t.Error("Expected world, got", cmds[0].ExpectedOutput[1])
			}
		} else {
			t.Error("Expected two output lines, got", cmds[0].ExpectedOutput)
		}
		if cmds[1].CmdLine != "echo goodbye" {
			t.Error("Expected echo goodbye, got", cmds[1])
		}
		if len(cmds[1].ExpectedOutput) == 1 {
			if cmds[1].ExpectedOutput[0] != "goodbye" {
				t.Error("Expected goodbye, got", cmds[1].ExpectedOutput[1])
			}
		} else {
			t.Error("Expected one output line, got", cmds[1].ExpectedOutput)
		}
	} else {
		t.Error("Expected two commands, got", cmds)
	}
}
