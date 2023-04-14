FROM alpine
COPY scraper /bin/runner
RUN apk --no-cache add ca-certificates git
WORKDIR /root/
ENTRYPOINT ["/bin/runner"]
