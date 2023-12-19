# syntax=docker/dockerfile:1
FROM kubeshop/testkube-jmeter-executor:jmeter5.6.2-jdk21-base
COPY jmeter /bin/runner

ENV ENTRYPOINT_CMD="/executor_entrypoint.sh"
WORKDIR /root/

COPY ./contrib/executor/jmeter/scripts/entrypoint.sh /executor_entrypoint.sh
ADD ./contrib/executor/jmeter/plugins/ ${JMETER_CUSTOM_PLUGINS_FOLDER}
ADD ./contrib/executor/jmeter/lib ${JMETER_HOME}/lib/

ENTRYPOINT ["/bin/runner"]
