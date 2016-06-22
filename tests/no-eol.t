An output line with no final newline must be marked with (no-eol) for
it to be considered a match:

  $ echo -n hey
  hey (no-eol)

Changes in output are indicated using (no-eol) marker the diff output:

  $ cat > diff.t << EOM
  >   $ echo -n hello
  >   hello
  >   $ echo world
  >   world (no-eol)
  > EOM
  $ cram diff.t
  F
  When executing "echo -n hello":
  -hello
  +hello (no-eol)
  When executing "echo world":
  -world (no-eol)
  +world
  # Ran 1 tests (2 commands), 0 errors, 1 failures
  [1]

When patching, the (no-eol) markers are inserted and removed as
necessary:

  $ yes | cram --interactive diff.t
  F
  When executing "echo -n hello":
  -hello
  +hello (no-eol)
  Accept this change? When executing "echo world":
  -world (no-eol)
  +world
  Accept this change? Patched diff.t
  # Ran 1 tests (2 commands), 0 errors, 1 failures
  [1]

  $ cat diff.t
    $ echo -n hello
    hello (no-eol)
    $ echo world
    world
