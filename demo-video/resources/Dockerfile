# syntax=docker/dockerfile:1
FROM golang:1.18
WORKDIR /svc/
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -o app .

FROM alpine:latest  
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=0 /svc/app ./
CMD ["./app"]  