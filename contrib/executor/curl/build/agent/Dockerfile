# syntax=docker/dockerfile:1
FROM alpine:3.18.0
RUN apk --no-cache add ca-certificates && \
    apk --no-cache add curl git
COPY curl /bin/runner

WORKDIR /root/
USER 1001
ENTRYPOINT ["/bin/runner"]
