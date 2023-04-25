# TODO add valid dockerfile with executor

## Base image consists of
    # FROM alpine
    # RUN apk --no-cache add ca-certificates && \
    #     apk --no-cache add curl git

FROM kubeshop/testkube-curl-executor:base
COPY curl /bin/runner

WORKDIR /root/
USER 1001
ENTRYPOINT ["/bin/runner"]
