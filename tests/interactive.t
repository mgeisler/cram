The --interactive flag does nothing if there are no test failures:

  $ touch empty.t
  $ cram --interactive empty.t
  .
  # Ran 1 tests (0 commands), 0 errors, 0 failures

When there are test failures, you're prompted to accept changes one at
a time:

  $ echo '  $ echo foo' >> test.t
  $ echo '  bar'        >> test.t

  $ echo y | cram --interactive test.t
  F
  When executing "echo foo", output changed:
  -bar
  +foo
  Accept changed output? Patched test.t
  # Ran 1 tests (1 commands), 0 errors, 1 failures
  [1]

Patching updates the original .t file:

  $ cat test.t
    $ echo foo
    foo

When there are multiple failures, you can update just some of them:

  $ echo '  $ echo foo' >> multiple.t
  $ echo '  first'      >> multiple.t
  $ echo '  $ echo bar' >> multiple.t
  $ echo '  second'     >> multiple.t
  $ echo '  $ echo baz' >> multiple.t
  $ echo '  third'      >> multiple.t

Here we accept the 'foo' and 'baz' outputs:

  $ echo "y\nn\ny" | cram --interactive multiple.t
  F
  When executing "echo foo", output changed:
  -first
  +foo
  Accept changed output? When executing "echo bar", output changed:
  -second
  +bar
  Accept changed output? When executing "echo baz", output changed:
  -third
  +baz
  Accept changed output? Patched multiple.t
  # Ran 1 tests (3 commands), 0 errors, 1 failures
  [1]

  $ cat multiple.t
    $ echo foo
    foo
    $ echo bar
    second
    $ echo baz
    baz

Not answering 'yes'/'no' or 'y'/'n' causes the prompt to be shown
again:

  $ echo something else | cram --interactive multiple.t
  F
  When executing "echo bar", output changed:
  -second
  +bar
  Accept changed output? Please answer 'yes' or 'no'
  Accept changed output? # Ran 1 tests (3 commands), 0 errors, 1 failures
  [1]

The file was not updated in this case:

  $ cat multiple.t
    $ echo foo
    foo
    $ echo bar
    second
    $ echo baz
    baz
