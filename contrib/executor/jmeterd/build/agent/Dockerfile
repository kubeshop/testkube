# syntax=docker/dockerfile:1
FROM kubeshop/jmeter:5.5
COPY jmeterd /bin/runner

RUN microdnf update -y && microdnf install -y ca-certificates git && microdnf clean all
ENV ENTRYPOINT_CMD="/executor_entrypoint.sh"
WORKDIR /root/

COPY ./contrib/executor/jmeterd/scripts/entrypoint.sh /executor_entrypoint.sh
COPY scripts/jmeter-master.sh /executor_entrypoint_master.sh
ADD ./contrib/executor/jmeterd/plugins/ ${JMETER_CUSTOM_PLUGINS_FOLDER}
ADD ./contrib/executor/jmeterd/lib ${JMETER_HOME}/lib/

ENTRYPOINT ["/bin/runner"]
