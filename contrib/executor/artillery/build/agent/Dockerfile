FROM node:17
COPY artillery /bin/runner

RUN apt-get update && apt-get install -y git && \
    npm install -g artillery@2.0.0-31 && \
    useradd -m -d /home/artillery -s /bin/bash -u 1001 -r -g root artillery
USER 1001

EXPOSE 8082
WORKDIR /home/artillery

ENTRYPOINT ["/bin/runner"]
