# syntax=docker/dockerfile:1
# TODO add valid dockerfile with executor
FROM alpine
COPY curl /bin/runner

RUN apk --no-cache add ca-certificates && \
    apk --no-cache add curl git
WORKDIR /root/
USER 1001
ENTRYPOINT ["/bin/runner"]
