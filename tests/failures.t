Cram shows the failed command with a diff between the expected and
actual output:

  $ echo '  $ echo foo' >> test.t
  $ echo '  bar'        >> test.t
  $ cram test.t
  F
  When executing "echo foo", output changed:
  -bar
  +foo
  # Ran 1 tests (1 commands), 0 errors, 1 failures
  [1]

Unexpected output:

  $ echo '  $ echo foo' > unexpected.t
  $ cram unexpected.t
  F
  When executing "echo foo", output changed:
  +foo
  # Ran 1 tests (1 commands), 0 errors, 1 failures
  [1]

Missing output:

  $ echo '  $ true'  >> missing.t
  $ echo '  missing' >> missing.t
  $ cram missing.t
  F
  When executing "true", output changed:
  -missing
  # Ran 1 tests (1 commands), 0 errors, 1 failures
  [1]
