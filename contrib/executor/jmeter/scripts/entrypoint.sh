#!/bin/bash

EXECUTOR_CUSTOM_PLUGINS_FOLDER="${RUNNER_DATADIR}/uploads/plugins"

if [ -d $EXECUTOR_CUSTOM_PLUGINS_FOLDER ];
then
    echo "Copying custom plugins from ${EXECUTOR_CUSTOM_PLUGINS_FOLDER} to ${JMETER_HOME}/lib/ext"
    for plugin in ${EXECUTOR_CUSTOM_PLUGINS_FOLDER}/*.jar; do
        echo "Copying plugin: $plugin"
        cp $plugin ${JMETER_HOME}/lib/ext
    done;
else
    echo "No custom plugins found at ${EXECUTOR_CUSTOM_PLUGINS_FOLDER}"
fi

if [ -f "/entrypoint.sh" ];
then
  echo "Executing custom entrypoint script at /entrypoint.sh"
  /entrypoint.sh $@
else
  echo "Executing JMeter command directly: jmeter $@"
  jmeter $@
fi

