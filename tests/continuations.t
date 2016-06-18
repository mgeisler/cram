Commands start with "$" and can be continued on subsequent lines that
start with ">":

  $ hello () {
  >   echo "This is the hello function"
  >   echo "Hello $1, nice to meet you!"
  >   return 42
  > }

We can now use the function just defined

  $ hello Martin
  This is the hello function
  Hello Martin, nice to meet you!
  [42]

Changes in output are shown normally:

  $ cat > cont.t << EOM
  > Some text before the command
  >   $ echo foo
  >   > echo bar
  >   > echo baz
  > Some text after the command
  > EOM
  $ cram cont.t
  F
  When executing "echo foo\necho bar\necho baz":
  +foo
  +bar
  +baz
  # Ran 1 tests (1 commands), 0 errors, 1 failures
  [1]

You can patch such a command:

  $ yes | cram --interactive cont.t
  F
  When executing "echo foo\necho bar\necho baz":
  +foo
  +bar
  +baz
  Accept this change? Patched cont.t
  # Ran 1 tests (1 commands), 0 errors, 1 failures
  [1]

  $ cat cont.t
  Some text before the command
    $ echo foo
    > echo bar
    > echo baz
    foo
    bar
    baz
  Some text after the command

  $ cram cont.t
  .
  # Ran 1 tests (1 commands), 0 errors, 0 failures
