# syntax=docker/dockerfile:1
FROM golang:1.21
WORKDIR /build
COPY . .
ENV CGO_ENABLED=0 
ENV GOOS=linux

RUN go build -o /app main.go

ARG ALPINE_IMAGE
FROM ${ALPINE_IMAGE}
RUN apk --no-cache add ca-certificates libssl1.1
WORKDIR /root/
COPY --from=0 /app /bin/app
EXPOSE 8080
CMD ["/bin/app"]
