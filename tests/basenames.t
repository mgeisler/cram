Test for issue #31 where Cram could not handle multiple test files
with the same base name:

  $ mkdir foo bar
  $ touch foo/empty.t bar/empty.t
  $ cram foo/empty.t bar/empty.t
  ..
  # Ran 2 tests (0 commands), 0 errors, 0 failures
