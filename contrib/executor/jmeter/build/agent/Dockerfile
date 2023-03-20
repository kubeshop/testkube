# syntax=docker/dockerfile:1
FROM golang:1.19

WORKDIR contrib/executor/jmeter/build

COPY . .

ENV CGO_ENABLED=0
ENV GOOS=linux

RUN cd contrib/executor/jmeter/cmd/agent;go build -o /runner -mod mod -a .

FROM justb4/jmeter:5.5

ENV ENTRYPOINT_CMD="/executor_entrypoint.sh"

RUN apk --no-cache add ca-certificates git

WORKDIR /root/

COPY --from=0 /runner /bin/runner
COPY scripts/entrypoint.sh /executor_entrypoint.sh
ADD plugins/ ${JMETER_CUSTOM_PLUGINS_FOLDER}
ADD lib/ ${JMETER_HOME}/lib/

ENTRYPOINT ["/bin/runner"]
