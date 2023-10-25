#!/bin/bash

if [ -f "/executor_entrypoint_master.sh" ];
then
  echo "Executing custom entrypoint script at /entrypoint.sh"
  /executor_entrypoint_master.sh $@
else
  echo "Executing JMeter command directly: jmeter $@"
  jmeter $@
fi

