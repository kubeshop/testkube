# syntax=docker/dockerfile:1

FROM  kubeshop/jmeter:5.5

RUN microdnf update -y && microdnf install -y ca-certificates git && microdnf clean all

WORKDIR /root/

ENV ENTRYPOINT_CMD="/executor_entrypoint.sh"

COPY dist/runner /bin/runner
COPY scripts/entrypoint.sh /executor_entrypoint.sh
COPY scripts/jmeter-master.sh /executor_entrypoint_master.sh

ENTRYPOINT ["/bin/runner"]

