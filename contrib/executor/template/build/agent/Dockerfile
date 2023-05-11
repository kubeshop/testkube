# syntax=docker/dockerfile:1
FROM alpine
COPY template /bin/runner
RUN apk --no-cache add ca-certificates git
WORKDIR /root/
ENTRYPOINT ["/bin/runner"]
