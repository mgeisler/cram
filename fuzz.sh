#!/bin/sh

echo "Building go-fuzz"
go get -v github.com/dvyukov/go-fuzz/go-fuzz
go get -v github.com/dvyukov/go-fuzz/go-fuzz-build

echo "Instrumenting Cram"
go-fuzz-build github.com/mgeisler/cram/fuzz/ParseTest

cp -a tests fuzz/ParseTest/corpus

echo "Starting fuzz test"
timeout -s INT 25 go-fuzz -bin ParseTest-fuzz.zip -workdir=fuzz/ParseTest

exit_code=0
for path in fuzz/ParseTest/crashers/*.quoted; do
    # This test is here to support dash
    if [ -e "$path" ]; then
        exit_code=1
        echo "Found crasher:"
        cat "$path"
        echo "Output:"
        cat "${path%.quoted}.output"
        echo
    fi
done

echo "Exiting with exit code $exit_code"
exit $exit_code
