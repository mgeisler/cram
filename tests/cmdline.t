Cram uses a traditional flexible GNU-style command line parser. This
means that you can specify flags anywhere on the command line:

  $ touch foo.t bar.t
  $ cram foo.t -j 3 bar.t
  ..
  # Ran 2 tests (0 commands), 0 errors, 0 failures

You can also use short options without a space:

  $ cram -ij 2 foo.t
  .
  # Ran 1 tests (0 commands), 0 errors, 0 failures
