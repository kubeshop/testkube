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

BASE_PROPERTIES_FILE=${JMETER_PARENT_TEST_FOLDER}/user.properties
if [ -f "${BASE_PROPERTIES_FILE}" ]
then
  echo "Copying user properties file from ${BASE_PROPERTIES_FILE}"
  cp ${BASE_PROPERTIES_FILE} ${JMETER_HOME}/bin/
else
  echo "File user.properties not present in ${JMETER_PARENT_TEST_FOLDER}"
fi
echo

NESTED_PROPERTIES_FILE=${JMETER_PARENT_TEST_FOLDER}/properties/user.properties
if [ -f "${NESTED_PROPERTIES_FILE}" ]
then
  echo "Copying user properties file from ${NESTED_PROPERTIES_FILE}"
  cp ${NESTED_PROPERTIES_FILE} ${JMETER_HOME}/bin/
else
  echo "File user.properties not present in ${JMETER_PARENT_TEST_FOLDER}/properties"
fi
echo

echo "********************************************************"
echo "*            Initializing JMeter Master                *"
echo "********************************************************"
echo

freeMem=$(awk '/MemAvailable/ { print int($2/1024) }' /proc/meminfo)

[[ -z ${JVM_XMN} ]] && JVM_XMN=$(($freeMem*2/10))
[[ -z ${JVM_XMS} ]] && JVM_XMS=$(($freeMem*8/10))
[[ -z ${JVM_XMX} ]] && JVM_XMX=$(($freeMem*8/10))

echo "Setting dynamically heap size based on available resources JVM_ARGS=-Xmn${JVM_XMN}m -Xms${JVM_XMS}m -Xmx${JVM_XMX}m"
export JVM_ARGS="-Xmn${JVM_XMN}m -Xms${JVM_XMS}m -Xmx${JVM_XMX}m"

if [ -n "$MASTER_OVERRIDE_JVM_ARGS" ]; then
  echo "Overriding JVM_ARGS=${MASTER_OVERRIDE_JVM_ARGS}"
  export JVM_ARGS="${MASTER_OVERRIDE_JVM_ARGS}"
fi

if [ -n "$MASTER_ADDITIONAL_JVM_ARGS" ]; then
  echo "Appending additional JVM args: ${MASTER_ADDITIONAL_JVM_ARGS}"
  export JVM_ARGS="${JVM_ARGS} ${MASTER_ADDITIONAL_JVM_ARGS}"
fi

echo "Available memory: ${freeMem} MB"
echo "Configured JVM_ARGS=${JVM_ARGS}"
echo

echo "********************************************************"
echo "*                Executing JMeter tests                *"
echo "********************************************************"
echo

if [ -z "$SSL_DISABLED" ]; then
    SSL_DISABLED=true
fi

CONN_ARGS="-Jserver.rmi.ssl.disable=${SSL_DISABLED}"
echo "Executing command: jmeter $@ ${CONN_ARGS} "
echo
echo "Started CMD"
jmeter $@ ${CONN_ARGS}

echo "END Finished JMeter test on $(date) for test ${file}"
echo

echo "********************************************************"
echo "*           JMeter test executions finished            *"
echo "********************************************************"
echo
