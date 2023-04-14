FROM smartbear/soapuios-testrunner
COPY soapui /bin/runner

RUN apt-get update && apt-get install -y git && \
    chmod 777 /usr/local/SmartBear && \
    useradd -m -d /home/soapui -s /bin/bash -u 1001 -r -g root soapui
USER 1001

ENTRYPOINT ["/bin/runner"]
