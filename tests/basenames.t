Test for issue #31 where Cram could not handle multiple test files
with the same base name:

  $ mkdir tmp foo bar
  $ touch foo/empty.t bar/empty.t
  $ TMPDIR=tmp cram --keep-tmp foo/empty.t bar/empty.t
  # Temporary directory: tmp/cram-* (glob)
  ..
  # Ran 2 tests (0 commands), 0 errors, 0 failures

The tests are put into nicely numbered directories:

  $ ls tmp/cram-*
  000-empty
  001-empty
