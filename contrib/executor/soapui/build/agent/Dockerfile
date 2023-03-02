# syntax=docker/dockerfile:1
FROM golang:1.18 as builder
WORKDIR /build
COPY . .
ENV CGO_ENABLED=0
ENV GOOS=linux
RUN cd cmd/agent;go build -o /runner -mod mod -a .

FROM smartbear/soapuios-testrunner
COPY --from=builder /runner /bin/runner

RUN apt-get update && apt-get install -y git && \
    chmod 777 /usr/local/SmartBear && \
    useradd -m -d /home/soapui -s /bin/bash -u 1001 -r -g root soapui
USER 1001

ENTRYPOINT ["/bin/runner"]
