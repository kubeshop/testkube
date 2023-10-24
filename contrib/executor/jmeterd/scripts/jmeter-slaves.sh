#!/bin/bash

echo "********************************************************"
echo "*              Installing JMeter Plugins               *"
echo "********************************************************"
echo

if [ -d ${JMETER_PARENT_TEST_FOLDER}/plugins ]
then
  echo "Installing user plugins from ${JMETER_PARENT_TEST_FOLDER}/plugins"
  for plugin in ${JMETER_PARENT_TEST_FOLDER}/plugins/*.jar; do
      echo "Copying plugin $plugin to ${JMETER_HOME}/lib/ext/"
      cp $plugin ${JMETER_HOME}/lib/ext
  done;
else
  echo "No user plugins provided as directory ${JMETER_PARENT_TEST_FOLDER}/plugins is not present"
fi
echo

echo "********************************************************"
echo "*            Initializing JMeter Master                *"
echo "********************************************************"
echo

freeMem=`awk '/MemAvailable/ { print int($2/1024) }' /proc/meminfo`

[[ -z ${JVM_XMN} ]] && JVM_XMN=$(($freeMem/10*2))
[[ -z ${JVM_XMS} ]] && JVM_XMS=$(($freeMem/10*8))
[[ -z ${JVM_XMX} ]] && JVM_XMX=$(($freeMem/10*8))

echo "Setting dynamically heap size based on available resources JVM_ARGS=-Xmn${JVM_XMN}m -Xms${JVM_XMS}m -Xmx${JVM_XMX}m"
export JVM_ARGS="-Xmn${JVM_XMN}m -Xms${JVM_XMS}m -Xmx${JVM_XMX}m"

if [ -n "$OVERRIDE_JVM_ARGS" ]; then
  echo "Overriding JVM_ARGS=${OVERRIDE_JVM_ARGS}"
  export JVM_ARGS="${OVERRIDE_JVM_ARGS}"
fi

if [ -n "$ADDITIONAL_JVM_ARGS" ]; then
  echo "Appending additional JVM args: ${ADDITIONAL_JVM_ARGS}"
  export JVM_ARGS="${JVM_ARGS} ${ADDITIONAL_JVM_ARGS}"
fi

echo "Available memory: ${freeMem} MB"
echo "Configured JVM_ARGS=${JVM_ARGS}"
echo

echo "********************************************************"
echo "*              Starting JMeter Server                  *"
echo "********************************************************"
echo

SERVER_ARGS="-Dserver.rmi.localport=60001 -Dserver_port=1099 -Jserver.rmi.ssl.disable=${SSL_DISABLED}"
echo "Running command: jmeter-server ${SERVER_ARGS} ${SLAVES_ADDITIONAL_JMETER_ARGS}"
echo

jmeter-server ${SERVER_ARGS} ${SLAVES_ADDITIONAL_JMETER_ARGS}