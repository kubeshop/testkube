# syntax=docker/dockerfile:1
FROM golang:1.18
WORKDIR contrib/executor/kubepug/build
COPY . .
ENV CGO_ENABLED=0 
ENV GOOS=linux

RUN cd contrib/executor/kubepug/cmd/agent;go build -o /runner -mod mod -a .

FROM bitnami/kubectl
USER root
RUN apt-get update && apt-get install -y wget
RUN mkdir -p $HOME/.kubepug  && \
    cd $HOME/.kubepug && \
    wget https://github.com/rikatz/kubepug/releases/download/v1.1.3/kubepug_linux_amd64.tar.gz && \
    tar -zxvf kubepug_linux_amd64.tar.gz && \
    rm -f kubepug_linux_amd64.tar.gz
ENV PATH="$HOME/.kubepug/:${PATH}"
WORKDIR /root/
COPY --from=0 /runner /bin/runner
USER 1001
ENTRYPOINT ["/bin/runner"]
