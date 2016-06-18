Cram will run the commands in a temporary directory. The location of
the directory can be specified using the normal $TMPDIR environment
varible.

This test will record the working directory in a file in our $PWD:

  $ echo "  $ pwd > $PWD/pwd.txt" > record-pwd.t
  $ mkdir custom-tmp
  $ TMPDIR=custom-tmp cram record-pwd.t
  .
  # Ran 1 tests (1 commands), 0 errors, 0 failures

The working directory recorded is indeed inside our $PWD:

  $ sed -e "s|^$PWD|<PWD>|" < pwd.txt
  <PWD>/custom-tmp/cram-*/000-record-pwd (glob)

The directory is normally deleted after the test has executed:

  $ ls custom-tmp

You can use the --keep-tmp flag to prevent this if you want to inspect
the files created. The temporary directory is printed in that case:

  $ TMPDIR=custom-tmp cram --keep-tmp record-pwd.t
  # Temporary directory: custom-tmp/cram-* (glob)
  .
  # Ran 1 tests (1 commands), 0 errors, 0 failures

  $ ls custom-tmp
  cram-* (glob)
