# syntax=docker/dockerfile:1
FROM alpine
COPY example /bin/runner

RUN apk --no-cache add ca-certificates git
WORKDIR /root/
ENTRYPOINT ["/bin/runner"]
