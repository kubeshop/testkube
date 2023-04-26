# Base image consists of:

# FROM smartbear/soapuios-testrunner
# RUN apt-get update && apt-get install -y git && \
   #  chmod 777 /usr/local/SmartBear && \
   #  useradd -m -d /home/soapui -s /bin/bash -u 1001 -r -g root soapui

# syntax=docker/dockerfile:1
FROM kubeshop/testkube-soapui-executor:base
COPY soapui /bin/runner
USER 1001

ENTRYPOINT ["/bin/runner"]
