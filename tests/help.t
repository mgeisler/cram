Cram comes with builtin help:

  $ cram --help
  usage: cram [<flags>] <path>...
  
  Flags:
        --help         Show context-sensitive help (also try --help-long and
                       --help-man).
    -i, --interactive  interactively update test file on failure
    -v, --verbose      show names of test files
        --debug        output debug information
        --keep-tmp     keep temporary directory after executing tests
    -j, --jobs=\d+ +   number of tests to run in parallel (re)
        --version      Show application version.
  
  Args:
    <path>  test files
  
  [1]

The traditional --version flag also works:

  $ cram --version
  cram version 0.0.0
