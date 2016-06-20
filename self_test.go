// Copyright 2016 Martin Geisler <martin@geisler.net>
//
// Cram is licensed under the MIT license, see the LICENSE file.

package cram

import (
	"os/exec"
	"path/filepath"
	"testing"
)

// TestSelf runs Cram on all .t files inside the tests directory.
func TestSelf(t *testing.T) {
	paths, err := filepath.Glob("tests/*.t")
	if err != nil {
		t.Fatal("Error while globbing:", err)
	}

	cmd := exec.Command("cram", paths...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Log("Cram failed:", err)
		t.Log(string(output))
		t.Fail()
	}
}
