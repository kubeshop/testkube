# syntax=docker/dockerfile:1
FROM justb4/jmeter:5.5
COPY jmeter /bin/runner

RUN apk --no-cache add ca-certificates git openssh sshpass
ENV ENTRYPOINT_CMD="/executor_entrypoint.sh"
WORKDIR /root/

COPY ./contrib/executor/jmeter/scripts/entrypoint.sh /executor_entrypoint.sh
ADD ./contrib/executor/jmeter/plugins/ ${JMETER_CUSTOM_PLUGINS_FOLDER}
ADD ./contrib/executor/jmeter/lib ${JMETER_HOME}/lib/

ENTRYPOINT ["/bin/runner"]
