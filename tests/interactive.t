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
  When executing "echo foo":
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

  $ echo '  $ echo foo' >> multiple.t
  $ echo '  first'      >> multiple.t
  $ echo '  $ echo bar' >> multiple.t
  $ echo '  second'     >> multiple.t
  $ echo '  $ echo baz' >> multiple.t
  $ echo '  third'      >> multiple.t

Here we accept the 'foo' and 'baz' outputs:

  $ echo "y\nn\ny" | cram --interactive multiple.t
  F
  When executing "echo foo":
  -first
  +foo
  Accept this change? When executing "echo bar":
  -second
  +bar
  Accept this change? When executing "echo baz":
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
  When executing "echo bar":
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

  $ echo 'Wrong exit code'                  >> exit-code.t
  $ echo '  $ (exit 7)'                     >> exit-code.t
  $ echo '  [10]'                           >> exit-code.t
  $ echo 'Missing non-zero exit code:'      >> exit-code.t
  $ echo '  $ false'                        >> exit-code.t
  $ echo 'White-space after the exit code:' >> exit-code.t
  $ echo '  $ true'                         >> exit-code.t
  $ echo '  [42]   '                        >> exit-code.t
  $ yes | cram --interactive exit-code.t
  F
  When executing "(exit 7)":
  -[10]
  +[7]
  Accept this change? When executing "false":
  +[1]
  Accept this change? When executing "true":
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
