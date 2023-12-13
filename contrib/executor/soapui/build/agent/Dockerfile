# syntax=docker/dockerfile:1
FROM smartbear/soapuios-testrunner:5.7.2
RUN apt-get update && apt-get install -y git \
   curl && \
   chmod 777 /usr/local/SmartBear && \
   useradd -m -d /home/soapui -s /bin/bash -u 1001 -r -g root soapui

COPY soapui /bin/runner
USER 1001

ENTRYPOINT ["/bin/runner"]
