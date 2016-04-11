Cram will normally exit with a status of 0 to indicate success:

  $ touch empty.t
  $ cram empty.t; echo "Cram exit code: $?"
  .
  # Ran 1 tests (0 commands), 0 errors, 0 failures.
  Cram exit code: 0

Test failures set the exit code to 1:

  $ echo '  $ echo foo' >> extra-output.t
  $ cram *.t; echo "Cram exit code: $?"
  .F
  When executing "echo foo", got
    foo
  but expected
    
  # Ran 2 tests (1 commands), 0 errors, 1 failures.
  Cram exit code: 1

If an error occurs, the error is shown, the error count incremented,
and the exit code is set to 2:

  $ cram does-not-exist.t *.t; echo "Cram exit code: $?"
  open does-not-exist.t: no such file or directory
  E.F
  When executing "echo foo", got
    foo
  but expected
    
  # Ran 3 tests (1 commands), 1 errors, 1 failures.
  Cram exit code: 2
