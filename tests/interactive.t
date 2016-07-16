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
  test.t:1: When executing "echo foo":
  -bar
  +foo
  Accept this change? Patched test.t
  # Ran 1 tests (1 commands), 0 errors, 1 failures
  [1]

Patching updates the original .t file:

  $ cat test.t
    $ echo foo
    foo

When there are multiple failures, you can update just some of them:

  $ cat > multiple.t << EOM
  >   $ echo foo
  >   first
  >   $ echo bar
  >   second
  >   $ echo baz
  >   third
  > EOM

Here we accept the 'foo' and 'baz' outputs:

  $ echo "y\nn\ny" | cram --interactive multiple.t
  F
  multiple.t:1: When executing "echo foo":
  -first
  +foo
  Accept this change? multiple.t:3: When executing "echo bar":
  -second
  +bar
  Accept this change? multiple.t:5: When executing "echo baz":
  -third
  +baz
  Accept this change? Patched multiple.t
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
  multiple.t:3: When executing "echo bar":
  -second
  +bar
  Accept this change? Please answer 'yes' or 'no'
  Accept this change? # Ran 1 tests (3 commands), 0 errors, 1 failures
  [1]

The file was not updated in this case:

  $ cat multiple.t
    $ echo foo
    foo
    $ echo bar
    second
    $ echo baz
    baz

You will also be prompted to update the exit code:

  $ cat > exit-code.t << EOM
  > Wrong exit code
  >   $ (exit 7)
  >   [10]
  > Missing non-zero exit code:
  >   $ false
  > White-space after the exit code:
  >   $ true
  >   [42]   
  > EOM
  $ yes | cram --interactive exit-code.t
  F
  exit-code.t:2: When executing "(exit 7)":
  -[10]
  +[7]
  Accept this change? exit-code.t:5: When executing "false":
  +[1]
  Accept this change? exit-code.t:7: When executing "true":
  -[42]   
  Accept this change? Patched exit-code.t
  # Ran 1 tests (3 commands), 0 errors, 1 failures
  [1]

  $ cat exit-code.t
  Wrong exit code
    $ (exit 7)
    [7]
  Missing non-zero exit code:
    $ false
    [1]
  White-space after the exit code:
    $ true
