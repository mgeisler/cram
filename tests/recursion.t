Cram can recursively find test files inside a directory:

  $ mkdir -p foo/x foo/y
  $ touch foo/a.t foo/x/b.t foo/y/c.t
  $ cram -v -j 1 foo
  . foo/a.t: 0 commands passed
  . foo/x/b.t: 0 commands passed
  . foo/y/c.t: 0 commands passed
  
  # Ran 3 tests (0 commands), 0 errors, 0 failures

It is only .t files that are found:

  $ touch foo/bar.go
  $ cram -v -j 1 foo
  . foo/a.t: 0 commands passed
  . foo/x/b.t: 0 commands passed
  . foo/y/c.t: 0 commands passed
  
  # Ran 3 tests (0 commands), 0 errors, 0 failures

Explicit paths on the command line can have any extension:

  $ touch README.md tests.txt
  $ cram -v -j 1 README.md tests.txt
  . README.md: 0 commands passed
  . tests.txt: 0 commands passed
  
  # Ran 2 tests (0 commands), 0 errors, 0 failures
