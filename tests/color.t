Cram will use ANSI color codes in the output if it detects that stdout
is connected to a (pseudo) terminal. Commands in a test file are run
without a pseudo terminal, so Cram will not use colors:

  $ echo '  $ echo hello' > hello.t
  $ cram hello.t
  F
  hello.t:1: When executing "echo hello":
  +hello
  # Ran 1 tests (1 commands), 0 errors, 1 failures
  [1]

We can force colors:

  $ cram --color always hello.t
  F
  hello.t:1: When executing "echo hello":
  \x1b[32m+hello (esc)
  \x1b[0m# Ran 1 tests (1 commands), 0 errors, 1 failures (esc)
  [1]

We can also disable colors with "--color never", but since colors are
already disabled here, we cannot demonstrate it without starting a
pseudo terminal.
