# syntax=docker/dockerfile:1
FROM ubuntu
COPY init /bin/runner
RUN apt-get update
RUN apt-get install ca-certificates git curl -y
WORKDIR /root/
RUN git config --global user.name "testkube"
RUN chmod a=rw /root/.gitconfig
ENTRYPOINT ["/bin/runner"]
