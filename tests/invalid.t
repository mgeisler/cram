Parsing a file with an output line before any command lines:

  $ echo '  This is an output line' > test.t
  $ cram test.t
  test.t:0: Output line "  This is an output line\n" has no command.
  E
  # Ran 1 tests (0 commands), 1 errors, 0 failures.
  [2]

Parsing a file with a command and with an output line immediated after
a commentary line:

  $ echo '  $ echo hello'  > test.t
  $ echo '  hello'        >> test.t
  $ echo 'Commentary'     >> test.t
  $ echo '  Output line'  >> test.t

  $ cram test.t
  test.t:3: Output line "  Output line\n" has no command.
  E
  # Ran 1 tests (0 commands), 1 errors, 0 failures.
  [2]

Parse invalid file with no final newline:

  $ echo -n '  ' > test.t
  $ cram test.t
  test.t:0: Output line "  " has no command.
  E
  # Ran 1 tests (0 commands), 1 errors, 0 failures.
  [2]
