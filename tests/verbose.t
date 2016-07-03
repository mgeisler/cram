The -v or --verbose flag can be used to make Cram output the names of
the test files as they execute. The order depends on the scheduling of
the goroutines, so we add -j 1 here to make the order consistent:

  $ touch foo.t bar.t
  $ cram -v -j 1 foo.t bar.t
  . foo.t: 0 commands passed
  . bar.t: 0 commands passed
  
  # Ran 2 tests (0 commands), 0 errors, 0 failures

The number of failed commands is shown for failed test files:

  $ cat > failure.t << EOM
  >   $ false
  >   $ true
  > EOM
  $ cram -v failure.t
  F failure.t: 1 of 2 commands failed
  
  When executing "false":
  +[1]
  # Ran 1 tests (2 commands), 0 errors, 1 failures
  [1]

Tests with errors have the errors shown:

  $ echo "  > bad" > error.t
  $ cram -v error.t
  E error.t:0: Continuation line "  > bad\n" has no command
  
  # Ran 1 tests (0 commands), 1 errors, 0 failures
  [2]
