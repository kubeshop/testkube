#syntax=docker/dockerfile:1
 FROM node:21.4.0-bookworm-slim 
 RUN apt-get update && apt-get install -y git && \
     npm install -g artillery@2.0.3 && \
     useradd -m -d /home/artillery -s /bin/bash -u 1001 -r -g root artillery


COPY artillery /bin/runner

USER 1001

EXPOSE 8082
WORKDIR /home/artillery

ENTRYPOINT ["/bin/runner"]
