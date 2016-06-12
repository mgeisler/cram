Cram supports output lines with regular expressions:

  $ echo hello beautiful world
  hello .* world (re)

  $ echo hello beautiful world
  hello \w+ world (re)

The amount of whitespace before the "(re)" is important:

  $ echo 'hello world   '
  hello.world    (re)

When such a line fails, you get the diff as normal:

  $ echo '  $ echo foo42' >> failure.t
  $ echo '  [0-9]+ (re)'  >> failure.t
  $ cram failure.t
  F
  When executing "echo foo42":
  -[0-9]+ (re)
  +foo42
  # Ran 1 tests (1 commands), 0 errors, 1 failures
  [1]

The pattern is anchored at the beginning and the end of the line:

  $ cat > anchor.t << EOM
  >   $ echo foobar
  >   foo (re)
  >   $ echo foobar
  >   bar (re)
  > EOM
  $ cram anchor.t
  F
  When executing "echo foobar":
  -foo (re)
  +foobar
  When executing "echo foobar":
  -bar (re)
  +foobar
  # Ran 1 tests (2 commands), 0 errors, 1 failures
  [1]
