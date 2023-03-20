# syntax=docker/dockerfile:1
FROM --platform=amd64 golang:1.18
WORKDIR contrib/executor/init/build
COPY . .
ENV CGO_ENABLED=0 
ENV GOOS=linux
ARG TARGETOS TARGETARCH

RUN cd contrib/executor/init/cmd/agent;GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /runner -mod mod -a .

FROM alpine
RUN apk --no-cache add ca-certificates git
WORKDIR /root/
RUN git config --global user.name "testkube"
RUN chmod a=rw /root/.gitconfig
COPY --from=0 /runner /bin/runner
ENTRYPOINT ["/bin/runner"]
