Cram can be used to test non-printable or binary output from commands
by escaping the expected output lines:

  $ echo "foo\tbar"
  foo\tbar (esc)

You only need to escape non-printable characters. Accented letters or
non-ASCII characters are fine:

  $ echo "ε ≪ 1 ≪ 1000"
  ε ≪ 1 ≪ 1000

It is not a problem to have an unnecessary (esc) marker:

  $ echo foo bar
  foo bar (esc)

You can also escape printable characters if you really like:

  $ echo foo bar
  \x66\x6f\x6f bar (esc)

The backslash escape character can be escaped:

  $ echo 'foo \\ bar'
  foo \\ bar (esc)

Since we compare the actual and expected output directly first, you
only need to escape a backslash if you use (esc):

  $ echo 'foo \\ bar'
  foo \ bar

Changes in output are marked with (esc) as necessary:

  $ cat > diff.t << EOM
  >   $ printf 'foo\tbar\n'
  >   foo bar
  > EOM
  $ cram diff.t
  F
  diff.t:1: When executing "printf 'foo\\tbar\\n'":
  -foo bar
  +foo\tbar (esc)
  # Ran 1 tests (1 commands), 0 errors, 1 failures
  [1]

Patching works as expected:

  $ yes | cram --interactive diff.t
  F
  diff.t:1: When executing "printf 'foo\\tbar\\n'":
  -foo bar
  +foo\tbar (esc)
  Accept this change? Patched diff.t
  # Ran 1 tests (1 commands), 0 errors, 1 failures
  [1]

  $ cat diff.t
    $ printf 'foo\tbar\n'
    foo\tbar (esc)
