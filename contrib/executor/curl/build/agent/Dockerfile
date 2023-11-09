# syntax=docker/dockerfile:1
FROM ubuntu:23.10
RUN apt update
RUN apt install -y ca-certificates && \
    apt install -y curl git
COPY curl /bin/runner

WORKDIR /root/
USER 1001
ENTRYPOINT ["/bin/runner"]
