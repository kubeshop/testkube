# syntax=docker/dockerfile:1
FROM golang:1.18
WORKDIR /build
COPY . .
ENV CGO_ENABLED=0 
ENV GOOS=linux

RUN cd cmd/agent;go build -o /runner -mod mod -a .

FROM node:17
RUN apt-get update && apt-get install -y git
RUN npm install -g artillery@latest
WORKDIR /root/
COPY --from=0 /runner /bin/runner
EXPOSE 8082
ENTRYPOINT ["/bin/runner"]
