package ParseTest

import (
	"bytes"

	"github.com/mgeisler/cram"
)

func Fuzz(data []byte) int {
	path := "<fuzz>"
	r := bytes.NewReader(data)
	test, err := cram.ParseTest(r, path)
	if err != nil {
		return 0
	}
	if test.Path != path {
		panic("unexpected path: " + test.Path)
	}

	return 1
}
