# syntax=docker/dockerfile:1

FROM justb4/jmeter:5.5

RUN apk --no-cache add ca-certificates git

WORKDIR /root/

ENV ENTRYPOINT_CMD="/executor_entrypoint.sh"

COPY dist/runner /bin/runner
COPY scripts/entrypoint.sh /executor_entrypoint.sh
ADD plugins/ ${JMETER_CUSTOM_PLUGINS_FOLDER}
ADD lib/ ${JMETER_HOME}/lib/

ENTRYPOINT ["/bin/runner"]
