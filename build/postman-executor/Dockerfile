# syntax=docker/dockerfile:1
FROM golang:1.16
WORKDIR /build
COPY . .
ENV GONOSUMDB=github.com/kubeshop/* 
ENV CGO_ENABLED=0 
ENV GOOS=linux

# TODO remove after going public - used for private operator repo deps 
ARG TOKEN
RUN git config --global url."https://fog1985:${TOKEN}@github.com".insteadOf "https://github.com"

RUN cd cmd/postman-executor;go build -o /app -mod mod -a .

FROM postman/newman
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=0 /app /bin/app
EXPOSE 8082
ENTRYPOINT ["/bin/app"]
