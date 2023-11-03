# syntax=docker/dockerfile:1
FROM kubeshop/jmeter:5.5
COPY jmeterd /bin/runner

RUN microdnf update -y && microdnf install -y ca-certificates git sshpass  && microdnf clean all
ENV ENTRYPOINT_CMD="/executor_entrypoint.sh"
WORKDIR /root/
COPY ./contrib/executor/jmeterd/scripts/entrypoint.sh /executor_entrypoint.sh
COPY ./contrib/executor/jmeterd/scripts/jmeter-master.sh /executor_entrypoint_master.sh

ENTRYPOINT ["/bin/runner"]
