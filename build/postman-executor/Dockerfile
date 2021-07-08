# syntax=docker/dockerfile:1
FROM golang:1.16
WORKDIR /build
COPY . .
ENV GONOSUMDB=github.com/kubeshop/* 
ENV CGO_ENABLED=0 
ENV GOOS=linux
RUN cd cmd/postman-executor;go build -o /app -mod vendor -a .

FROM alpine:latest  
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=0 /app /bin/app
CMD ["/bin/app"]  