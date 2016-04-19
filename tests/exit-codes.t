Cram will normally exit with a status of 0 to indicate success:

  $ touch empty.t
  $ cram empty.t
  .
  # Ran 1 tests (0 commands), 0 errors, 0 failures.

Test failures set the exit code to 1:

  $ echo '  $ echo foo' >> extra-output.t
  $ cram *.t
  .F
  When executing "echo foo", got
    foo
  but expected
    
  # Ran 2 tests (1 commands), 0 errors, 1 failures.
  [1]

If an error occurs, the error is shown, the error count incremented,
and the exit code is set to 2:

  $ cram does-not-exist.t *.t
  open does-not-exist.t: no such file or directory
  E.F
  When executing "echo foo", got
    foo
  but expected
    
  # Ran 3 tests (1 commands), 1 errors, 1 failures.
  [2]

A command with no output can also have a non-zero exit code:

  $ false
  [1]

Though it is redundant, a zero exit code can still be specified:

  $ true
  [0]

Mismatches in exit codes are shown in the Cram output:

  $ echo '  $ false' >> false.t
  $ cram false.t
  F
  When executing "false", got
    exit code 1, but expected 0
  # Ran 1 tests (1 commands), 0 errors, 1 failures.
  [1]
