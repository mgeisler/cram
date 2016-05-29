Cram comes with builtin help:

  $ cram --help
  NAME:
     cram - A new cli application
  
  USAGE:
     cram [global options] command [command options] [arguments...]
     
  VERSION:
     0.0.0
     
  COMMANDS:
  GLOBAL OPTIONS:
     --interactive  interactively update test file on failure
     --debug        output debug information
     --keep-tmp     keep temporary directory after executing tests
     --help, -h     show help
     --version, -v  print the version
     
The traditional --version flag also works:

  $ cram --version
  cram version 0.0.0
