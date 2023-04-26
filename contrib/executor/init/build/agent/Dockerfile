# syntax=docker/dockerfile:1
FROM alpine
COPY init /bin/runner
RUN apk --no-cache add ca-certificates git
WORKDIR /root/
RUN git config --global user.name "testkube"
RUN chmod a=rw /root/.gitconfig
ENTRYPOINT ["/bin/runner"]
