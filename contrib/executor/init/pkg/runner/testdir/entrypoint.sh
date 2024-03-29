#!/bin/sh
"testdir/prerun.sh"
prerun_exit_code=$?
if [ $prerun_exit_code -ne 0 ]; then
  exit $prerun_exit_code
fi
"testdir/command.sh" $@
command_exit_code=$?
"testdir/postrun.sh"
postrun_exit_code=$?
if [ $command_exit_code -ne 0 ]; then
  exit $command_exit_code
fi
exit $postrun_exit_code
