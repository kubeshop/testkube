FROM alpine:latest
USER root
# Install curl to download testkube
RUN apk --no-cache add curl jq && \
    # Install aws-cli
    apk add --no-cache aws-cli && \
    # Install kubectl
    curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl \
    && chmod +x ./kubectl \
    && mv ./kubectl /usr/local/bin
# COPY kubectl-testkube-mac /usr/local/bin
# Download testkube and move to bin directory
ENV PLATFORM="Linux_x86_64"

RUN VERSION=`curl -s -f https://api.github.com/repos/kubeshop/testkube/releases/latest | jq -r .tag_name | cut -c2-` \
    && curl -L "https://github.com/kubeshop/testkube/releases/download/v${VERSION}/testkube_${VERSION}_${PLATFORM}.tar.gz" | tar -xzvf - \
    && chmod +x ./kubectl-testkube \
    && mv ./kubectl-testkube /usr/local/bin \
    && mkdir .testkube && echo "{}" > .testkube/config.json \
    && rm -rf LICENSE README.md && apk del curl jq

# USER 1001
ENTRYPOINT ["/usr/local/bin/kubectl-testkube"]
