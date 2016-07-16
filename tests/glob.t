Cram supports output lines with glob patterns:

  $ echo 'hello beautiful world!'
  hello * world? (glob)

The only two special characters are "*" and "?". The "*" character
matches any number of characters, including "/":

  $ echo foo/bar/baz
  foo*z (glob)

The "?" character matches a single character:

  $ echo x/y
  x?y (glob)

You can escape these characters with a backslash:

  $ echo '?foo*'
  \?foo\* (glob)

The matching of actual and expected output is done line-by-line, so
you cannot match a newline with either character:

  $ echo '  $ echo 'a'; echo b' >> newline.t
  $ echo '  a?b (glob)'         >> newline.t
  $ cram newline.t
  F
  newline.t:1: When executing "echo a; echo b":
  -a?b (glob)
  +a
  +b
  # Ran 1 tests (1 commands), 0 errors, 1 failures
  [1]
