# syntax=docker/dockerfile:1
FROM justb4/jmeter:5.5
COPY jmeterd /bin/runner

RUN apk --no-cache add ca-certificates git
ENV ENTRYPOINT_CMD="/executor_entrypoint.sh"
WORKDIR /root/

COPY ./contrib/executor/jmeterd/scripts/entrypoint.sh /executor_entrypoint.sh
COPY scripts/jmeter-master.sh /executor_entrypoint_master.sh
ADD ./contrib/executor/jmeterd/plugins/ ${JMETER_CUSTOM_PLUGINS_FOLDER}
ADD ./contrib/executor/jmeterd/lib ${JMETER_HOME}/lib/

ENTRYPOINT ["/bin/runner"]
