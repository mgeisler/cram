// Copyright 2016 Martin Geisler <martin@geisler.net>
//
// Cram is licensed under the MIT license, see the LICENSE file.

package cram

// Copies of standard library functions for compatibility with earlier
// versions of Go.

//// Go version 1.5 ////

// LastIndexByte returns the index of the last instance of c in s, or
// -1 if c is not present in s.
func stringsLastIndexByte(s string, c byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			return i
		}
	}
	return -1
}
