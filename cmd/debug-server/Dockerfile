# syntax=docker/dockerfile:1
FROM golang:1.18
WORKDIR /build
COPY . .
ENV CGO_ENABLED=0 
ENV GOOS=linux

RUN go build -o /app main.go

FROM alpine:latest  
RUN apk --no-cache add ca-certificates libssl1.1
WORKDIR /root/
COPY --from=0 /app /bin/app
EXPOSE 8080
CMD ["/bin/app"]
