When Cram executes a test file, $TESTDIR will point to the directory
where the test file was.

  $ cat > x.t << EOM
  >   $ echo \$TESTDIR
  >   $PWD
  > EOM
  $ cram
  .
  # Ran 1 tests (1 commands), 0 errors, 0 failures

This can differ from one test file to another:

  $ mkdir -p foo
  $ cat > foo/y.t << EOM
  >   $ echo \$TESTDIR
  >   $PWD/foo
  > EOM

  $ cd foo
  $ cram -j 1 ../
  ..
  # Ran 2 tests (2 commands), 0 errors, 0 failures

Environment variables related to the locale are also reset:

  $ echo $LC_ALL, $LANG, $LANGUAGE
  C, C, C

The timezone is reset to GMT:

  $ echo $TZ
  GMT

The terminal is always an xterm with 80 columns:

  $ echo $TERM $COLUMNS
  xterm 80

The CDPATH and GREP_OPTIONS environment variables are removed from the
environment:

  $ cat > reset.t << EOM
  >   $ echo \$CDPATH
  >   
  >   $ echo \$GREP_OPTIONS
  >   
  > EOM
  $ CDPATH=foo GREP_OPTIONS=bar cram reset.t
  .
  # Ran 1 tests (2 commands), 0 errors, 0 failures
