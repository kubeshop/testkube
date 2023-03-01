# syntax=docker/dockerfile:1
FROM golang:1.18
WORKDIR /build
COPY . .
ENV CGO_ENABLED=0 
ENV GOOS=linux

RUN cd cmd/agent;go build -o /runner -mod mod -a .

FROM node:17

RUN apt-get update && apt-get install -y git && \
    npm install -g artillery@2.0.0-27 && \
    useradd -m -d /home/artillery -s /bin/bash -u 1001 -r -g root artillery
USER 1001

COPY --from=0 /runner /bin/runner
EXPOSE 8082
WORKDIR /home/artillery

ENTRYPOINT ["/bin/runner"]
