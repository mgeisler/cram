// Copyright 2016 Martin Geisler <martin@geisler.net>
//
// Cram is licensed under the MIT license, see the LICENSE file.

package Escape

import "github.com/mgeisler/cram"

func Fuzz(data []byte) int {
	line := string(data)
	escaped := cram.Escape(line)
	if escaped == line {
		return 0
	}
	unescaped, err := cram.Unescape(escaped)
	if err != nil {
		panic(err)
	}
	if unescaped != line {
		panic("change after unescaping")
	}

	return 0
}
