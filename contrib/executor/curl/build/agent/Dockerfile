# TODO add valid dockerfile with executor

## Base image consists of
#  FROM alpine:3.18.0
#  RUN apk --no-cache add ca-certificates && \
#      apk --no-cache add curl git

# syntax=docker/dockerfile:1
FROM kubeshop/testkube-curl-executor:base
COPY curl /bin/runner

WORKDIR /root/
USER 1001
ENTRYPOINT ["/bin/runner"]
