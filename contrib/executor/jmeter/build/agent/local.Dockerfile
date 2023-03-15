# syntax=docker/dockerfile:1

FROM justb4/jmeter

RUN apk --no-cache add ca-certificates git

WORKDIR /root/

COPY dist/runner /bin/runner

ENTRYPOINT ["/bin/runner"]
