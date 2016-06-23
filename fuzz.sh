#!/bin/sh

echo "Building go-fuzz"
go get -v github.com/dvyukov/go-fuzz/go-fuzz
go get -v github.com/dvyukov/go-fuzz/go-fuzz-build

# This test has no fixed corpus, we use our test cases.
cp -a tests fuzz/ParseTest/corpus

fuzz_test () {
    echo
    echo "Preparing $1 fuzz test"
    go-fuzz-build github.com/mgeisler/cram/fuzz/$1

    echo "Starting $1 fuzz test"
    timeout -s INT 25 go-fuzz -bin $1-fuzz.zip -workdir=fuzz/$1

    local return_value=0
    for path in fuzz/$1/crashers/*.quoted; do
        # This test is here to support dash
        if [ -e "$path" ]; then
            return_value=1
            echo "Found crasher:"
            cat "$path"
            echo "Output:"
            cat "${path%.quoted}.output"
            echo
        fi
    done
    return $return_value
}

exit_code=0
for t in ParseTest; do
    if ! fuzz_test $t; then
        exit_code=1
    fi
done

echo
echo "Exiting with exit code $exit_code"
exit $exit_code
