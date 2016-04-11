Cram shows the failed command with the actual and expected output:

  $ echo '  $ echo foo' >> test.t
  $ echo '  bar'        >> test.t
  $ cram test.t
  F
  When executing "echo foo", got
    foo
  but expected
    bar
  # Ran 1 tests (1 commands), 0 errors, 1 failures.
